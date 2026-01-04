// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 20:48
// Original filename: src/extras/logs.go

package extras

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"dtools2/rest"

	"github.com/docker/docker/pkg/stdcopy"
	ce "github.com/jeanfrancoisgratton/customError/v3"
)

func Logs(client *rest.Client, container string) *ce.CustomError {
	q := url.Values{}
	q.Set("stdout", "true")
	q.Set("stderr", "true")

	if LogFollow {
		q.Set("follow", "true")
	}
	if LogTimestamps {
		q.Set("timestamps", "true")
	}
	if LogTail >= 0 {
		q.Set("tail", strconv.Itoa(LogTail))
	} else {
		q.Set("tail", "all")
	}

	path := "/containers/" + container + "/logs"
	resp, err := client.Do(rest.Context, http.MethodGet, path, q, nil, nil)
	if err != nil {
		return &ce.CustomError{Title: "Unable to fetch logs", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(b))
		if msg == "" {
			msg = resp.Status
		}
		return &ce.CustomError{Title: "Logs request failed", Message: fmt.Sprintf("GET %s returned %s", path, msg)}
	}

	// The logs endpoint returns either:
	// - a raw stream (TTY containers)
	// - a multiplexed stream (non-TTY), same framing as attach (stdcopy)
	br := bufio.NewReader(resp.Body)

	peek, perr := br.Peek(8)
	if perr != nil && perr != io.EOF {
		return &ce.CustomError{Title: "Unable to read logs", Message: perr.Error()}
	}

	if looksLikeStdCopyMux(peek) {
		_, err = stdcopy.StdCopy(os.Stdout, os.Stderr, br)
	} else {
		_, err = io.Copy(os.Stdout, br)
	}
	if err != nil && err != io.EOF {
		return &ce.CustomError{Title: "Error while streaming logs", Message: err.Error()}
	}

	return nil
}

func looksLikeStdCopyMux(hdr []byte) bool {
	if len(hdr) < 8 {
		return false
	}
	// header format:
	//   [0] stream (1=stdout,2=stderr)
	//   [1..3] 0x00
	//   [4..7] big-endian uint32 payload size
	if hdr[0] != 1 && hdr[0] != 2 {
		return false
	}
	if hdr[1] != 0 || hdr[2] != 0 || hdr[3] != 0 {
		return false
	}

	// Basic sanity on size to reduce false positives.
	sz := binary.BigEndian.Uint32(hdr[4:8])
	if sz == 0 {
		// Could be valid, but it's unusual as the first frame; allow it anyway.
		return true
	}
	if sz > 128*1024*1024 {
		return false
	}

	// Also reduce false positives by ensuring the header isn't all-zero-ish after stream byte.
	if bytes.Equal(hdr, make([]byte, 8)) {
		return false
	}

	return true
}
