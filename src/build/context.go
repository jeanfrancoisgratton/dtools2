// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/04 00:03
// Original filename: src/build/context.go

package build

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func makeContextTarStream(ctx context.Context, contextDir, dockerfileRel string) (io.ReadCloser, error) {
	contextDir = filepath.Clean(contextDir)

	st, err := os.Stat(contextDir)
	if err != nil {
		return nil, fmt.Errorf("context path error: %w", err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("context path %q is not a directory", contextDir)
	}

	matcher, err := newIgnoreMatcher(contextDir, dockerfileRel)
	if err != nil {
		return nil, fmt.Errorf("reading .dockerignore: %w", err)
	}

	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)
		defer func() {
			_ = tw.Close()
			_ = pw.Close()
		}()

		walkErr := filepath.WalkDir(contextDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			rel, err := filepath.Rel(contextDir, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}

			relTar := filepath.ToSlash(rel)

			// Apply dockerignore.
			if matcher.isIgnored(relTar) {
				// Do NOT SkipDir: negation patterns may re-include files deeper down.
				return nil
			}

			info, err := os.Lstat(path)
			if err != nil {
				return err
			}

			var linkTarget string
			if info.Mode()&os.ModeSymlink != 0 {
				lt, lerr := os.Readlink(path)
				if lerr != nil {
					return lerr
				}
				linkTarget = lt
			}

			hdr, err := tar.FileInfoHeader(info, linkTarget)
			if err != nil {
				return err
			}

			// Ensure stable names inside tar.
			hdr.Name = relTar
			if info.IsDir() && !strings.HasSuffix(hdr.Name, "/") {
				hdr.Name += "/"
			}

			// Normalize timestamps to avoid weirdness (optional, but keeps output stable-ish).
			hdr.AccessTime = time.Time{}
			hdr.ChangeTime = time.Time{}

			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}

			if info.Mode().IsRegular() {
				f, err := os.Open(path)
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

		if walkErr != nil {
			_ = pw.CloseWithError(walkErr)
			return
		}
	}()

	return pr, nil
}
