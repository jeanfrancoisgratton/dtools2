// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/28 12:07
// Original filename: src/rest/client.go

package rest

import (
	"context"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
)

// New creates a REST client for unix/http/https endpoints.
// Examples:
//
//	-H unix:///var/run/docker.sock
//	-H http://host:2375
//	-H https://host:2376
func New(rawHost string, forcedAPIVersion string) (*Client, error) {
	if rawHost == "" {
		rawHost = "unix:///var/run/docker.sock"
	}
	u, err := parseDockerHost(rawHost)
	if err != nil {
		return nil, err
	}
	tr, err := makeTransport(u)
	if err != nil {
		return nil, err
	}
	return &Client{
		baseURL:     u,
		httpClient:  &http.Client{Transport: tr, Timeout: 60 * time.Second},
		apiVersion:  normalizeVersion(forcedAPIVersion),
		forced:      forcedAPIVersion != "",
		dialTimeout: 5 * time.Second,
	}, nil
}

// Do performs an HTTP request to the Engine.
// Example: Do(ctx, "GET", []string{"containers", "json"}, nil, nil)
func (c *Client) Do(ctx context.Context, method string, pathParts []string, body io.Reader, headers map[string]string) (*http.Response, error) {
	// Negotiate version before versioned paths if not forced
	if !c.forced && c.apiVersion == "" && len(pathParts) > 0 && pathParts[0] != "_ping" && pathParts[0] != "version" {
		if err := c.NegotiateAPIversion(ctx); err != nil {
			return nil, err
		}
	}

	reqURL := *c.baseURL
	p := "/" + strings.Trim(strings.Join(pathParts, "/"), "/")
	if c.apiVersion != "" && (len(pathParts) == 0 || (pathParts[0] != "_ping" && pathParts[0] != "version")) {
		p = "/" + c.apiVersion + p
	}
	reqURL.Path = path.Clean(reqURL.Path + p)

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.httpClient.Do(req)
}
