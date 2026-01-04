// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 22:19
// Original filename: src/extras/run_pull.go

package extras

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"dtools2/auth"
	"dtools2/rest"

	"github.com/docker/docker/pkg/jsonmessage"
	mobyterm "github.com/moby/term"
)

// pullImageViaDaemon pulls an image through the daemon (same endpoint as `docker pull`).
// Implemented here to avoid import cycles (extras must not import images).
func pullImageViaDaemon(client *rest.Client, ref string) error {
	if ref == "" {
		return fmt.Errorf("image reference is required")
	}

	repo, tag := splitRepoTag(ref)
	if tag == "" {
		tag = "latest"
	}

	q := url.Values{}
	q.Set("fromImage", repo)
	q.Set("tag", tag)

	headers := http.Header{}

	reg := registryFromImageRef(ref)
	if reg != "" {
		h, err := auth.BuildRegistryAuthHeader(reg)
		if err != nil {
			return fmt.Errorf("building auth header for registry %q: %w", reg, err)
		}
		headers.Set("X-Registry-Auth", h)
	}

	resp, err := client.Do(rest.Context, http.MethodPost, "/images/create", q, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out io.Writer = os.Stdout
	if rest.QuietOutput {
		out = io.Discard
	}

	termFd, isTerm := mobyterm.GetFdInfo(out)

	if err := jsonmessage.DisplayJSONMessagesStream(resp.Body, out, termFd, isTerm, nil); err != nil {
		if jerr, ok := err.(*jsonmessage.JSONError); ok {
			return fmt.Errorf("image pull failed: %s", jerr.Message)
		}
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("image pull failed: %s", resp.Status)
	}

	return nil
}

// splitRepoTag splits an image reference into repository and tag.
// It treats the last ':' after the last '/' as the tag separator.
// If no such ':' exists, tag is empty.
func splitRepoTag(ref string) (repo, tag string) {
	// Strip any @digest part.
	if i := strings.Index(ref, "@"); i != -1 {
		ref = ref[:i]
	}

	slash := strings.LastIndex(ref, "/")
	colon := strings.LastIndex(ref, ":")

	if colon > slash {
		// repo:tag
		return ref[:colon], ref[colon+1:]
	}

	// no tag
	return ref, ""
}

// registryFromImageRef attempts to extract the registry host from an image reference.
// Same heuristic as Docker:
//
//   - Take the first path component (before the first '/').
//   - If it contains a '.' or ':', or is "localhost", treat it as a registry.
//   - Otherwise, assume no explicit registry (Docker Hub style) and return "".
func registryFromImageRef(ref string) string {
	// Strip any @digest part.
	if i := strings.Index(ref, "@"); i != -1 {
		ref = ref[:i]
	}

	slash := strings.Index(ref, "/")
	if slash == -1 {
		return ""
	}

	first := ref[:slash]
	if strings.Contains(first, ".") || strings.Contains(first, ":") || first == "localhost" {
		return first
	}

	return ""
}
