// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/14 08:14
// Original filename: src/images/pull.go

// dtools2
// images/pull.go
// Image pull logic using the Docker REST API.

package images

import (
	"context"
	"dtools2/auth"
	"dtools2/rest"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

// PullImage pulls an image from a registry via the daemon, streaming progress
// using Docker's own jsonmessage renderer (same output as `docker pull`).
func PullImage(ctx context.Context, client *rest.Client, opts PullOptions, out io.Writer) error {
	if opts.ImageTag == "" {
		return fmt.Errorf("image reference is required")
	}

	repo, tag := splitRepoTag(opts.ImageTag)
	if tag == "" {
		tag = "latest"
	}

	q := url.Values{}
	q.Set("fromImage", repo)
	q.Set("tag", tag)

	headers := http.Header{}

	if opts.Registry != "" {
		h, err := auth.BuildRegistryAuthHeader(opts.Registry)
		if err != nil {
			return fmt.Errorf("building auth header for registry %q: %w", opts.Registry, err)
		}
		headers.Set("X-Registry-Auth", h)
	}

	resp, err := client.Do(ctx, http.MethodPost, "/images/create", q, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Use Docker's JSON progress machinery to render output.
	termFd, isTerm := term.GetFdInfo(out)

	if err := jsonmessage.DisplayJSONMessagesStream(resp.Body, out, termFd, isTerm, nil); err != nil {
		// If the error is a JSONError, surface its message like docker does.
		if jerr, ok := err.(*jsonmessage.JSONError); ok {
			return fmt.Errorf("image pull failed: %s", jerr.Message)
		}
		return err
	}

	// If the daemon returned a plain HTTP error without a JSON body,
	// still surface that.
	if resp.StatusCode >= 400 {
		return fmt.Errorf("image pull failed: %s", resp.Status)
	}

	// No extra "Image pull completed." message: docker doesn't print one.
	return nil
}

// Simple CLI helper for Cobra.
func PullImageFromCLI(ctx context.Context, client *rest.Client, ref string) error {
	opts := PullOptions{
		ImageTag: ref,
		Registry: registryFromImageRef(ref),
	}
	return PullImage(ctx, client, opts, os.Stdout)
}
