// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/11 20:16
// Original filename: src/system/cp_helpers.go

package system

import (
	"archive/tar"
	"dtools2/rest"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// --- helpers ---
func splitContainerPath(s string) (string, string, bool) {
	ndx := strings.IndexByte(s, ':')
	if ndx <= 0 || ndx == len(s)-1 {
		return "", "", false
	}
	return s[:ndx], s[ndx+1:], true
}

func endsWithPathSep(p string) bool {
	return strings.HasSuffix(p, string(os.PathSeparator)) || strings.HasSuffix(p, "/")
}

func normalizeHostContentsOnly(p string) (string, bool) {
	// docker cp supports dir/. to mean "contents only".
	if strings.HasSuffix(p, string(os.PathSeparator)+".") {
		return strings.TrimSuffix(p, string(os.PathSeparator)+"."), true
	}
	if strings.HasSuffix(p, "/.") {
		return strings.TrimSuffix(p, "/."), true
	}
	return p, false
}

func addDotIfDir(p string) string {
	if strings.HasSuffix(p, "/.") {
		return p
	}
	trim := strings.TrimRight(p, "/")
	if trim == "" {
		trim = "/"
	}
	return trim + "/."
}

func dockerMsg(r io.Reader) string {
	b, _ := io.ReadAll(io.LimitReader(r, 64*1024))
	s := strings.TrimSpace(string(b))
	if s == "" {
		return ""
	}
	var der dockerErrorResponse
	if json.Unmarshal(b, &der) == nil && strings.TrimSpace(der.Message) != "" {
		return ": " + strings.TrimSpace(der.Message)
	}
	return ": " + s
}

func hostExistsIsDir(p string) (bool, bool, error) {
	statPath := strings.TrimRight(p, string(os.PathSeparator))
	if statPath == "" {
		statPath = string(os.PathSeparator)
	}
	st, err := os.Stat(statPath)
	if err == nil {
		return true, st.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, false, nil
	}
	return false, false, err
}

func statContainerPath(client *rest.Client, containerRef, containerPath string) (*containerPathStat, bool, *ce.CustomError) {
	q := url.Values{}
	q.Set("path", containerPath)
	endpoint := "/containers/" + url.PathEscape(containerRef) + "/archive"

	resp, err := client.Do(rest.Context, http.MethodHead, endpoint, q, nil, nil)
	if err != nil {
		return nil, false, &ce.CustomError{Title: "Unable to stat container path", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, &ce.CustomError{Title: "HTTP request returned an error", Message: endpoint + " returned " + resp.Status}
	}

	h := resp.Header.Get("X-Docker-Container-Path-Stat")
	if h == "" {
		return nil, false, &ce.CustomError{Title: "Unexpected daemon response", Message: "missing X-Docker-Container-Path-Stat header"}
	}

	decoded, derr := decodeB64(h)
	if derr != nil {
		return nil, false, &ce.CustomError{Title: "Unable to decode stat header", Message: derr.Error()}
	}

	var st containerPathStat
	if jerr := json.Unmarshal(decoded, &st); jerr != nil {
		return nil, false, &ce.CustomError{Title: "Unable to parse stat header JSON", Message: jerr.Error()}
	}
	return &st, true, nil
}

func decodeB64(s string) ([]byte, error) {
	// Docker uses standard base64; tolerate raw variants.
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	if b, err := base64.URLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}

// --- tar extraction (container -> host) ---

func extractTarDir(r io.Reader, destinationDir string) error {
	base := filepath.Clean(destinationDir)
	if err := os.MkdirAll(base, 0o755); err != nil {
		return err
	}

	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		rel := sanitizeTarName(hdr.Name)
		if rel == "" {
			continue
		}
		target := filepath.Join(base, rel)
		if !isSubPath(base, target) {
			return errors.New("tar entry escapes destination directory")
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			mode := os.FileMode(hdr.Mode) & 0o777
			if mode == 0 {
				mode = 0o755
			}
			if err := os.MkdirAll(target, mode); err != nil {
				return err
			}

		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(hdr.Mode) & 0o777
			if mode == 0 {
				mode = 0o644
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			_, cErr := io.Copy(f, tr)
			cErr2 := f.Close()
			if cErr != nil {
				return cErr
			}
			if cErr2 != nil {
				return cErr2
			}

		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				return err
			}

		case tar.TypeLink:
			// Hardlink within extracted tree.
			linkRel := sanitizeTarName(hdr.Linkname)
			if linkRel == "" {
				return errors.New("invalid hardlink target")
			}
			linkTarget := filepath.Join(base, linkRel)
			if !isSubPath(base, linkTarget) {
				return errors.New("hardlink target escapes destination directory")
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Link(linkTarget, target); err != nil {
				return err
			}
		default:
			// Ignore special entries (devices, pax headers, etc.)
			continue
		}
	}
}

func extractSingleFile(r io.Reader, destinationFile string) error {
	tr := tar.NewReader(r)
	found := false
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			if !found {
				return errors.New("tar stream contained no regular file")
			}
			return nil
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		if found {
			return errors.New("destination is a file but tar contains multiple files")
		}
		found = true

		if err := os.MkdirAll(filepath.Dir(destinationFile), 0o755); err != nil {
			return err
		}
		mode := os.FileMode(hdr.Mode) & 0o777
		if mode == 0 {
			mode = 0o644
		}
		f, err := os.OpenFile(destinationFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		_, cErr := io.Copy(f, tr)
		cErr2 := f.Close()
		if cErr != nil {
			return cErr
		}
		return cErr2
	}
}

func sanitizeTarName(name string) string {
	// Tar names are POSIX paths.
	n := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	n = strings.TrimPrefix(n, "/")
	n = strings.TrimPrefix(n, "./")
	if n == "." || n == "" {
		return ""
	}
	return filepath.FromSlash(n)
}

func isSubPath(base, target string) bool {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// --- tar creation (host -> container) ---

// makeTarStream creates a tar stream of srcPath.
// - If srcIsDir:
//   - if srcContentsOnly and tarRootName is empty: tar contains just directory contents.
//   - if tarRootName non-empty: tar contains a top-level directory tarRootName/ with contents.
//
// - If src is file: tar contains one entry named tarRootName.
func makeTarStream(ctx interface{ Done() <-chan struct{} }, srcPath, tarRootName string, srcIsDir, srcContentsOnly, tarNoRootDir bool) (io.ReadCloser, error) {
	_ = tarNoRootDir // (kept for readability at call sites; not needed for current implementation)

	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)
		defer func() {
			_ = tw.Close()
			_ = pw.Close()
		}()

		select {
		case <-ctx.Done():
			_ = pw.CloseWithError(errors.New("operation cancelled"))
			return
		default:
		}

		if !srcIsDir {
			if tarRootName == "" {
				tarRootName = filepath.Base(srcPath)
				if tarRootName == "" {
					tarRootName = "data"
				}
			}
			if err := tarWriteOne(tw, srcPath, tarRootName); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			return
		}

		// Directory
		baseDir := filepath.Clean(srcPath)
		if srcContentsOnly && tarRootName == "" {
			if err := tarWriteDirTree(tw, ctx, baseDir, ""); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			return
		}

		if tarRootName == "" {
			tarRootName = filepath.Base(baseDir)
			if tarRootName == "" {
				tarRootName = "data"
			}
		}

		if err := tarWriteDirHeader(tw, baseDir, tarRootName); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
		if err := tarWriteDirTree(tw, ctx, baseDir, tarRootName); err != nil {
			_ = pw.CloseWithError(err)
			return
		}
	}()

	return pr, nil
}

func tarWriteDirHeader(tw *tar.Writer, dirPath, tarName string) error {
	info, err := os.Lstat(dirPath)
	if err != nil {
		return err
	}
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = filepath.ToSlash(tarName)
	if !strings.HasSuffix(hdr.Name, "/") {
		hdr.Name += "/"
	}
	return tw.WriteHeader(hdr)
}

func tarWriteOne(tw *tar.Writer, srcPath, tarName string) error {
	info, err := os.Lstat(srcPath)
	if err != nil {
		return err
	}

	var linkTarget string
	if info.Mode()&os.ModeSymlink != 0 {
		lt, lerr := os.Readlink(srcPath)
		if lerr != nil {
			return lerr
		}
		linkTarget = lt
	}

	hdr, err := tar.FileInfoHeader(info, linkTarget)
	if err != nil {
		return err
	}
	hdr.Name = filepath.ToSlash(tarName)
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	if info.Mode().IsRegular() {
		f, err := os.Open(srcPath)
		if err != nil {
			return err
		}
		_, cErr := io.Copy(tw, f)
		_ = f.Close()
		return cErr
	}

	return nil
}

func tarWriteDirTree(tw *tar.Writer, ctx interface{ Done() <-chan struct{} }, dirPath, prefix string) error {
	return filepath.WalkDir(dirPath, func(p string, d os.DirEntry, walkErr error) error {
		_ = d
		if walkErr != nil {
			return walkErr
		}

		select {
		case <-ctx.Done():
			return errors.New("operation cancelled")
		default:
		}

		rel, err := filepath.Rel(dirPath, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		info, err := os.Lstat(p)
		if err != nil {
			return err
		}

		var linkTarget string
		if info.Mode()&os.ModeSymlink != 0 {
			lt, lerr := os.Readlink(p)
			if lerr != nil {
				return lerr
			}
			linkTarget = lt
		}

		hdr, err := tar.FileInfoHeader(info, linkTarget)
		if err != nil {
			return err
		}

		name := filepath.ToSlash(rel)
		if prefix != "" {
			name = filepath.ToSlash(filepath.Join(prefix, rel))
		}
		hdr.Name = name
		if info.IsDir() && !strings.HasSuffix(hdr.Name, "/") {
			hdr.Name += "/"
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			_, cErr := io.Copy(tw, f)
			_ = f.Close()
			if cErr != nil {
				return cErr
			}
		}

		return nil
	})
}
