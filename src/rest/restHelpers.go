// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/20 07:11
// Original filename: src/rest/restHelpers.go

package rest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// buildURL joins base + apiPrefix + p and applies query parameters.
func (c *Client) buildURL(p string, q url.Values) string {
	u := *c.BaseURL // copy
	// Clean join: /v1.41 + /containers/json
	fullPath := path.Join(c.APIprefix, p)
	// path.Join strips trailing slash; if you need to preserve it for some endpoints, handle here.
	if strings.HasSuffix(p, "/") && !strings.HasSuffix(fullPath, "/") {
		fullPath += "/"
	}
	u.Path = fullPath
	u.RawQuery = q.Encode()
	return u.String()
}

// GetJSON GETs and decodes JSON into out.
func (c *Client) GetJSON(ctx context.Context, p string, q url.Values, out any) *ce.CustomError {
	resp, err := c.Do(ctx, http.MethodGet, p, q, nil, nil)
	if err != nil {
		return &ce.CustomError{Code: 201, Title: "Error fetching JSON payload", Message: err.Error()}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		//return fmt.Errorf("GET %s: %s: %s", p, resp.Status, strings.TrimSpace(string(b)))
		return &ce.CustomError{Code: 201, Title: "Error fetching JSON payload",
			Message: fmt.Sprintf("GET %s: %s: %s", p, resp.Status, strings.TrimSpace(string(b)))}
	}
	if out == nil {
		// drain and return
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &ce.CustomError{Code: 202, Title: "Error decoding JSON payload", Message: err.Error()}
	}
	return nil
}

// PostJSON POSTs a JSON body (if in != nil) and decodes response JSON into out.
func (c *Client) PostJSON(ctx context.Context, p string, q url.Values, in any, out any, extra http.Header) error {
	var body io.Reader
	h := make(http.Header)
	for k, v := range extra {
		for _, vv := range v {
			h.Add(k, vv)
		}
	}
	if in != nil {
		pr, pw := io.Pipe()
		go func() {
			encErr := json.NewEncoder(pw).Encode(in)
			_ = pw.CloseWithError(encErr)
		}()
		body = pr
		h.Set("Content-Type", "application/json")
	}

	resp, err := c.Do(ctx, http.MethodPost, p, q, body, h)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("POST %s: %s: %s", p, resp.Status, strings.TrimSpace(string(b)))
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Do issues an HTTP request with context, returning the response and leaving
// the caller responsible for closing resp.Body.
func (c *Client) Do(ctx context.Context, method, p string, q url.Values, body io.Reader, headers http.Header) (*http.Response, error) {
	if c == nil || c.Http == nil {
		return nil, errors.New("nil client")
	}
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(p, q), body)
	if err != nil {
		return nil, err
	}
	// Default headers
	req.Header.Set("Accept", "application/json")
	if headers != nil {
		for k, vals := range headers {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}
	}
	return c.Http.Do(req)
}
