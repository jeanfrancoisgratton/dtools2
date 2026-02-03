// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/06 20:00
// Original filename: src/registry/jsonOutput.go

package registry

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

func (c *Client) CatalogJSON(ctx context.Context, q url.Values) ([]byte, *ce.CustomError) {
	return c.getJSON(ctx, "/v2/_catalog", q)
}

func (c *Client) TagsJSON(ctx context.Context, repo string, q url.Values) ([]byte, *ce.CustomError) {
	repo = strings.TrimLeft(repo, "/")
	if repo == "" {
		return nil, &ce.CustomError{Title: "Unable to fetch repository tags", Message: "repo name is empty"}
	}
	return c.getJSON(ctx, "/v2/"+repo+"/tags/list", q)
}

func (c *Client) getJSON(ctx context.Context, path string, q url.Values) ([]byte, *ce.CustomError) {
	if q == nil {
		q = url.Values{}
	}

	// 1) First attempt: anonymous, no auth headers
	resp, err := c.do(ctx, http.MethodGet, path, q, nil)
	if err != nil {
		return nil, &ce.CustomError{Title: "Error handling http GET", Message: err.Error()}
	}
	defer resp.Body.Close()

	// Fast-path success
	if resp.StatusCode == http.StatusOK {
		return readAll(resp.Body)
	}

	// 2) If challenged, handle transparently then retry once
	if resp.StatusCode == http.StatusUnauthorized {
		chal := resp.Header.Get("WWW-Authenticate")
		// Drain body before retry (keep connections healthy)
		_, _ = io.Copy(io.Discard, resp.Body)

		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(chal)), "bearer") {
			authHeader, err := c.bearerAuthHeaderFromChallenge(ctx, chal)
			if err != nil {
				return nil, err
			}
			resp2, err := c.do(ctx, http.MethodGet, path, q, map[string]string{
				"Authorization": authHeader,
			})
			if err != nil {
				return nil, err
			}
			defer resp2.Body.Close()

			if resp2.StatusCode == http.StatusOK {
				return readAll(resp2.Body)
			}
			return nil, &ce.CustomError{Title: "Error in http response for path " + path, Message: "Returned status was " + resp.Status}

		} else if strings.HasPrefix(strings.ToLower(strings.TrimSpace(chal)), "basic") {
			// Basic challenge: retry with Basic if we have creds
			if c.creds != nil {
				user, pass, ok := c.creds(c.baseURL.Host)
				if ok {
					resp2, err := c.do(ctx, http.MethodGet, path, q, map[string]string{
						"Authorization": basicAuthHeader(user, pass),
					})
					if err != nil {
						return nil, err
					}
					defer resp2.Body.Close()

					if resp2.StatusCode == http.StatusOK {
						return readAll(resp2.Body)
					}
					return nil, &ce.CustomError{Title: "Error in http response for path " + path, Message: "Returned status was " + resp.Status}
				}
			}
		}
	}

	return nil, &ce.CustomError{Title: "Error in http response for path " + path, Message: "Returned status was " + resp.Status}
}
