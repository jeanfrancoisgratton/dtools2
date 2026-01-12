// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/11 17:55
// Original filename: src/system/cp.go

package system

import (
	"dtools2/rest"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

func CopyFile(client *rest.Client, source, destination string) *ce.CustomError {
	if strings.Contains(source, ":") {
		return copyFrom(client, source, destination)
	}
	return copyTo(client, source, destination)
}

// copyFrom implements: dtools cp <container>:<path> <host-dest>
func copyFrom(client *rest.Client, source string, destination string) *ce.CustomError {
	containerRef, containerPath, ok := splitContainerPath(source)
	if !ok {
		return &ce.CustomError{Title: "Invalid source", Message: "expected container:path (e.g. mycontainer:/file)"}
	}
	if strings.TrimSpace(destination) == "" {
		return &ce.CustomError{Title: "Invalid destination", Message: "destination path is empty"}
	}

	// Stat the source inside the container.
	st, exists, err := statContainerPath(client, containerRef, containerPath)
	if err != nil {
		return err
	}
	if !exists {
		return &ce.CustomError{Title: "Container path not found", Message: "no such file or directory: " + containerPath}
	}
	srcIsDir := (os.FileMode(st.Mode) & os.ModeDir) != 0

	// Host destination shape.
	destExists, destIsDir, derr := hostExistsIsDir(destination)
	if derr != nil {
		return &ce.CustomError{Title: "Unable to stat destination", Message: derr.Error()}
	}
	destHintDir := endsWithPathSep(destination)

	if srcIsDir && destExists && !destIsDir {
		return &ce.CustomError{Title: "Invalid destination", Message: "cannot copy a directory onto an existing file"}
	}
	if destHintDir && destExists && !destIsDir {
		return &ce.CustomError{Title: "Invalid destination", Message: "destination ends with '/', but it is an existing file"}
	}

	// Decide how to ask the daemon for the tar stream.
	// For directory -> new destination (doesn't exist, no trailing slash): copy contents only.
	// For directory -> existing dir (or trailing slash): copy the directory itself.
	containerPathForAPI := containerPath
	destTreatAsDir := destHintDir || (destExists && destIsDir)
	if srcIsDir && !destTreatAsDir {
		containerPathForAPI = addDotIfDir(containerPathForAPI)
	}

	q := url.Values{}
	q.Set("path", containerPathForAPI)
	endpoint := "/containers/" + url.PathEscape(containerRef) + "/archive"

	resp, rerr := client.Do(rest.Context, http.MethodGet, endpoint, q, nil, nil)
	if rerr != nil {
		return &ce.CustomError{Title: "Unable to fetch archive from daemon", Message: rerr.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ce.CustomError{Title: "HTTP request returned an error", Message: endpoint + " returned " + resp.Status + dockerMsg(resp.Body)}
	}

	// Destination write:
	// - If destination is a directory (existing or explicitly trailing slash), untar into it.
	// - If destination is a file path, tar must contain exactly one regular file.
	if srcIsDir {
		if !destExists {
			if mkErr := os.MkdirAll(destination, 0o755); mkErr != nil {
				return &ce.CustomError{Title: "Unable to create destination directory", Message: mkErr.Error()}
			}
		}
		if xerr := extractTarDir(resp.Body, destination); xerr != nil {
			return &ce.CustomError{Title: "Unable to write destination", Message: xerr.Error()}
		}
		if !rest.QuietOutput {
			fmt.Println(hftx.EnabledSign("Copied : " + source + " ==> " + destination))
		}
		return nil
	}

	// src is a file
	if destHintDir || (destExists && destIsDir) {
		if !destExists {
			if mkErr := os.MkdirAll(destination, 0o755); mkErr != nil {
				return &ce.CustomError{Title: "Unable to create destination directory", Message: mkErr.Error()}
			}
		}
		if xerr := extractTarDir(resp.Body, destination); xerr != nil {
			return &ce.CustomError{Title: "Unable to write destination", Message: xerr.Error()}
		}
		if !rest.QuietOutput {
			fmt.Println(hftx.EnabledSign("Copied : " + source + " ==> " + destination))
		}
		return nil
	}

	// Rename semantics.
	if xerr := extractSingleFile(resp.Body, destination); xerr != nil {
		return &ce.CustomError{Title: "Unable to write destination", Message: xerr.Error()}
	}
	if !rest.QuietOutput {
		fmt.Println(hftx.EnabledSign("Copied : " + source + " ==> " + destination))
	}
	return nil
}

// copyTo implements: dtools cp <host-src> <container>:<path>
func copyTo(client *rest.Client, source string, destination string) *ce.CustomError {
	containerRef, containerPath, ok := splitContainerPath(destination)
	if !ok {
		return &ce.CustomError{Title: "Invalid destination", Message: "expected container:path (e.g. mycontainer:/file)"}
	}
	if strings.TrimSpace(source) == "" {
		return &ce.CustomError{Title: "Invalid source", Message: "source path is empty"}
	}

	// Host source stats.
	srcPath, srcContentsOnly := normalizeHostContentsOnly(source)
	info, serr := os.Lstat(srcPath)
	if serr != nil {
		return &ce.CustomError{Title: "Unable to stat source", Message: serr.Error()}
	}
	srcIsDir := info.IsDir()

	// Destination hints.
	containerPath = strings.TrimSpace(containerPath)
	if containerPath == "" {
		return &ce.CustomError{Title: "Invalid destination", Message: "destination container path is empty"}
	}
	destExplicitContentsOnly := strings.HasSuffix(containerPath, "/.")
	destHintDir := strings.HasSuffix(containerPath, "/") || destExplicitContentsOnly

	containerPathNoDot := strings.TrimSuffix(containerPath, "/.")
	containerPathNoDot = strings.TrimRight(containerPathNoDot, "/")
	if containerPathNoDot == "" {
		containerPathNoDot = "/"
	}

	// Best-effort stat of destination path in container.
	dstStat, dstExists, derr := statContainerPath(client, containerRef, containerPathNoDot)
	if derr != nil {
		return derr
	}
	dstIsDir := dstExists && ((os.FileMode(dstStat.Mode) & os.ModeDir) != 0)

	if srcIsDir && dstExists && !dstIsDir {
		return &ce.CustomError{Title: "Invalid destination", Message: "cannot copy a directory onto an existing file in the container"}
	}
	if destHintDir && dstExists && !dstIsDir {
		return &ce.CustomError{Title: "Invalid destination", Message: "destination ends with '/', but it is an existing file in the container"}
	}

	// Decide PUT target directory (query path) and tar layout.
	var (
		putDir       string
		tarRootName  string
		tarNoRootDir bool
	)

	// In docker cp, contents-only is controlled by the SOURCE (dir/.) not the destination.
	// We'll also accept destination ending with '/.' as an explicit directory target.
	if destExplicitContentsOnly {
		destHintDir = true
	}

	// If destination is (or is explicitly intended to be) a directory, copy "into" it.
	if destHintDir || dstIsDir {
		putDir = containerPathNoDot
		if putDir == "" {
			putDir = "/"
		}

		if srcIsDir {
			if srcContentsOnly {
				// Copy directory contents directly into putDir.
				tarRootName = ""
				tarNoRootDir = true
			} else {
				// Copy directory itself into putDir/<basename>.
				tarRootName = filepath.Base(srcPath)
				tarNoRootDir = false
			}
		} else {
			tarRootName = filepath.Base(srcPath)
			tarNoRootDir = true
		}
	} else {
		// Rename semantics: destination is a file path (or a new directory path if src is dir).
		putDir = path.Dir(containerPath)
		if putDir == "." || putDir == "" {
			putDir = "/"
		}

		base := path.Base(containerPath)
		if base == "" || base == "/" || base == "." {
			return &ce.CustomError{Title: "Invalid destination", Message: "destination container path is invalid"}
		}

		if srcIsDir {
			// Create a directory named <base> and copy directory contents into it.
			tarRootName = base
			tarNoRootDir = false
			// If user asked src dir/. then it's still contents into <base> (same outcome).
		} else {
			tarRootName = base
			tarNoRootDir = true
		}
	}

	stream, terr := makeTarStream(rest.Context, srcPath, tarRootName, srcIsDir, srcContentsOnly, tarNoRootDir)
	if terr != nil {
		return &ce.CustomError{Title: "Unable to create tar stream", Message: terr.Error()}
	}
	defer stream.Close()

	q := url.Values{}
	q.Set("path", putDir)
	endpoint := "/containers/" + url.PathEscape(containerRef) + "/archive"

	headers := http.Header{}
	headers.Set("Content-Type", "application/x-tar")

	resp, rerr := client.Do(rest.Context, http.MethodPut, endpoint, q, stream, headers)
	if rerr != nil {
		return &ce.CustomError{Title: "Unable to upload archive to daemon", Message: rerr.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ce.CustomError{Title: "HTTP request returned an error", Message: endpoint + " returned " + resp.Status + dockerMsg(resp.Body)}
	}
	if !rest.QuietOutput {
		fmt.Println(hftx.EnabledSign("Copied : " + source + " ==> " + destination))
	}
	return nil
}
