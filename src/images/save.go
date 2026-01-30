// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/30
// Original filename: src/images/save.go

package images

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"dtools2/rest"
)

// ImageSave saves one or more images from the daemon to a tar archive.
// The output file extension controls compression:
//   - .tar, .tar.gz/.tgz, .tar.bz2/.tbz2
//
// xz output is intentionally not supported.
func ImageSave(client *rest.Client, images []string, outFile string) error {
	if outFile == "" {
		return fmt.Errorf("output archive path is required")
	}
	if len(images) == 0 {
		return fmt.Errorf("at least one image reference is required")
	}

	w, err := openArchiveWriter(outFile)
	if err != nil {
		return fmt.Errorf("opening output archive %q: %w", outFile, err)
	}
	defer w.Close()

	q := url.Values{}
	for _, img := range images {
		if img == "" {
			continue
		}
		q.Add("names", img)
	}
	if len(q) == 0 {
		return fmt.Errorf("at least one non-empty image reference is required")
	}

	resp, err := client.Do(rest.Context, http.MethodGet, "/images/get", q, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := readSmallBody(resp.Body, 8192)
		if msg != "" {
			return fmt.Errorf("image save failed: %s: %s", resp.Status, msg)
		}
		return fmt.Errorf("image save failed: %s", resp.Status)
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		return err
	}

	return nil
}
