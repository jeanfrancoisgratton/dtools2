// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/30
// Original filename: src/images/load.go

package images

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"dtools2/rest"

	"github.com/docker/docker/pkg/jsonmessage"
	ce "github.com/jeanfrancoisgratton/customError/v3"
	"github.com/moby/term"
)

// ImageLoad loads image(s) from a tar archive (optionally compressed) into the daemon.
// This emulates `docker load` / `docker image load` behavior by locally decompressing
// based on file extension and streaming the uncompressed tar to POST /images/load.
func ImageLoad(client *rest.Client, tarball string) *ce.CustomError {
	r, err := openArchiveReader(tarball)
	if err != nil {
		return &ce.CustomError{Title: "error opening archive", Message: err.Error()}
	}
	defer r.Close()

	q := url.Values{}
	q.Set("quiet", strconv.FormatBool(rest.QuietOutput))

	headers := http.Header{}
	headers.Set("Content-Type", "application/x-tar")

	resp, derr := client.Do(rest.Context, http.MethodGet, "/images/get", q, nil, nil)
	if derr != nil {
		return &ce.CustomError{Title: "Error fetching images list", Message: derr.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := readSmallBody(resp.Body, 8192)
		if msg != "" {
			return &ce.CustomError{Title: "image load failed", Message: "http response is " + resp.Status + " (" + msg + ")"}
		}
		return &ce.CustomError{Title: "image load failed", Message: "http response is " + resp.Status}
	}

	if rest.QuietOutput {
		// Drain the body so the connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	termFd, isTerm := term.GetFdInfo(os.Stdout)
	if err := jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stdout, termFd, isTerm, nil); err != nil {
		if jerr, ok := err.(*jsonmessage.JSONError); ok {
			return &ce.CustomError{Title: "image load failed", Message: jerr.Error()}
		}
		return &ce.CustomError{Title: "image load failed", Message: err.Error()}
	}

	return nil
}

func readSmallBody(r io.Reader, limit int64) (string, error) {
	b, err := io.ReadAll(io.LimitReader(r, limit))
	if err != nil {
		return "", err
	}
	// Avoid trailing newlines; keep a single-line message.
	return string(bytesTrimSpace(b)), nil
}

func bytesTrimSpace(b []byte) []byte {
	start := 0
	for start < len(b) {
		switch b[start] {
		case ' ', '\t', '\n', '\r':
			start++
		default:
			goto endStart
		}
	}
endStart:
	end := len(b)
	for end > start {
		switch b[end-1] {
		case ' ', '\t', '\n', '\r':
			end--
		default:
			goto endEnd
		}
	}
endEnd:
	return b[start:end]
}
