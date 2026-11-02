// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/25 18:52
// Original filename: src/images/push.go

package images

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/docker/docker/pkg/jsonmessage"
	ce "github.com/jeanfrancoisgratton/customError/v3"
	"github.com/moby/term"

	"dtools2/auth"
	"dtools2/rest"
)

// ImagePush uses the daemon to push the image to its registry,
// streaming output EXACTLY like `docker push`.
func ImagePush(client *rest.Client, ref string) *ce.CustomError {
	repo, tag := splitRepoTag(ref)
	if tag == "" {
		tag = "latest"
	}

	registry := registryFromImageRef(ref)

	headers := http.Header{}
	if registry != "" {
		// No stored login: let the registry decide (it will report "denied" if auth is required)
		h, err := auth.BuildRegistryAuthHeader(registry)
		switch {
		case err == nil:
			headers.Set("X-Registry-Auth", h)
		case !errors.Is(err, auth.ErrNoCredentials):
			return &ce.CustomError{Title: "error building auth header", Message: err.Error()}
		}
	}

	q := url.Values{}
	q.Set("tag", tag)

	path := fmt.Sprintf("/images/%s/push", repo)

	resp, err := client.Do(rest.Context, http.MethodPost, path, q, nil, headers)
	if err != nil {
		return &ce.CustomError{Title: "error pushing image", Message: err.Error()}
	}
	defer resp.Body.Close()

	termFd, isTerm := term.GetFdInfo(os.Stdout)
	if err := jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stdout, termFd, isTerm, nil); err != nil {
		return &ce.CustomError{Title: "error displaying json messages", Message: err.Error()}
	}

	if resp.StatusCode >= 400 {
		return &ce.CustomError{Title: "http request error", Message: resp.Status}
	}

	return nil
}
