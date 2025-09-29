// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/29 09:05
// Original filename: src/image/pull.go

package image

import (
	"context"
	"dtools2/auth"
	"dtools2/rest"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type progressDetail struct {
	Current int64 `json:"current"`
	Total   int64 `json:"total"`
}

// Pull pulls an image via Engine API: POST /images/create?fromImage=&tag=
// It streams JSON progress and renders simple per-layer progress lines.
func Pull(ctx context.Context, c *rest.Client, images []string) error {
	for _, image := range images {
		reg := inferRegistry(image) // for auth lookup only
		xra, err := auth.BuildXRegistryAuth(reg)
		if err != nil {
			return err
		}

		refNoRegistry := stripRegistry(image)
		name, tag := splitNameTag(refNoRegistry)

		q := url.Values{}
		q.Set("fromImage", name)
		if tag != "" {
			q.Set("tag", tag)
		}

		headers := map[string]string{
			"X-Registry-Auth": xra,
			"Content-Type":    "application/json",
		}
		// Body may be empty for pull
		resp, err := c.Do(ctx, http.MethodPost, []string{"images", "create" + "?" + q.Encode()}, nil, headers)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 400 {
			b, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("pull failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
		}

		if err := renderProgress(ctx, resp.Body); err != nil {
			return err
		}
	}
	return nil
}
