// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/30
// Original filename: src/images/load.go

package images

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"dtools2/rest"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

// ImageLoad loads image(s) from a tar archive (optionally compressed) into the daemon.
// This emulates `docker load` / `docker image load` behavior by locally decompressing
// based on file extension and streaming the uncompressed tar to POST /images/load.
func ImageLoad(client *rest.Client, tarball string) error {
	if tarball == "" {
		return fmt.Errorf("tar archive path is required")
	}

	r, err := openArchiveReader(tarball)
	if err != nil {
		return fmt.Errorf("opening archive %q: %w", tarball, err)
	}
	defer r.Close()

	q := url.Values{}
	q.Set("quiet", strconv.FormatBool(rest.QuietOutput))

	headers := http.Header{}
	headers.Set("Content-Type", "application/x-tar")

	resp, err := client.Do(rest.Context, http.MethodPost, "/images/load", q, r, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		msg, _ := readSmallBody(resp.Body, 8192)
		if msg != "" {
			return fmt.Errorf("image load failed: %s: %s", resp.Status, msg)
		}
		return fmt.Errorf("image load failed: %s", resp.Status)
	}

	if rest.QuietOutput {
		// Drain the body so the connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	termFd, isTerm := term.GetFdInfo(os.Stdout)
	if err := jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stdout, termFd, isTerm, nil); err != nil {
		if jerr, ok := err.(*jsonmessage.JSONError); ok {
			return fmt.Errorf("image load failed: %s", jerr.Message)
		}
		return err
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
