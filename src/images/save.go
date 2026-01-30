// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/30
// Original filename: src/images/save.go

package images

import (
	"io"
	"net/http"
	"net/url"

	"dtools2/rest"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// ImageSave saves one or more images from the daemon to a tar archive.
// The output file extension controls compression:
//   - .tar, .tar.gz/.tgz, .tar.bz2/.tbz2
//
// xz output is intentionally not supported.
func ImageSave(client *rest.Client, images []string, outFile string) *ce.CustomError {
	w, err := openArchiveWriter(outFile)
	if err != nil {
		return err
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
		return &ce.CustomError{Title: "Unable to same image(s)", Message: "at least one non-empty image reference is required"}
	}

	resp, derr := client.Do(rest.Context, http.MethodGet, "/images/get", q, nil, nil)
	if derr != nil {
		return &ce.CustomError{Title: "Error fetching images list", Message: derr.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := readSmallBody(resp.Body, 8192)
		if msg != "" {
			return &ce.CustomError{Title: "image save failed", Message: "http response is " + resp.Status + " (" + msg + ")"}
		}
		return &ce.CustomError{Title: "image save failed", Message: "http response is " + resp.Status}
	}

	if _, err := io.Copy(w, resp.Body); err != nil {
		return &ce.CustomError{Title: "image save failed", Message: err.Error()}
	}

	return nil
}
