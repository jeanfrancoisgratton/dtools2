// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/28 12:07
// Original filename: src/rest/client.go

package rest

import (
	"context"
	"errors"
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
	if strings.TrimSpace(rawHost) == "" {
		rawHost = "unix:///var/run/docker.sock"
	}
	u, unixSock, err := parseDockerHost(rawHost)
	if err != nil {
		return nil, err
	}

	tr, err := makeTransport(u.Scheme, u.Hostname(), unixSock)
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL:     u, // guaranteed non-nil
		unixSock:    unixSock,
		httpClient:  &http.Client{Transport: tr, Timeout: 60 * time.Second},
		apiVersion:  normalizeVersion(forcedAPIVersion),
		forced:      forcedAPIVersion != "",
		dialTimeout: 5 * time.Second,
	}
	return c, nil
}

// Do performs a request without query string.
func (c *Client) Do(ctx context.Context, method string, pathParts []string, body io.Reader, headers map[string]string) (*http.Response, error) {
	return c.DoQ(ctx, method, pathParts, "", body, headers)
}

// DoQ performs a request with a raw query string like "fromImage=alpine&tag=latest".
func (c *Client) DoQ(ctx context.Context, method string, pathParts []string, rawQuery string, body io.Reader, headers map[string]string) (*http.Response, error) {
	if c.baseURL == nil {
		return nil, errors.New("client not initialized: baseURL is nil")
	}

	// Negotiate version before versioned paths if not forced
	if !c.forced && c.apiVersion == "" && len(pathParts) > 0 && pathParts[0] != "_ping" && pathParts[0] != "version" {
		if err := c.NegotiateAPIversion(ctx); err != nil {
			return nil, err
		}
	}

	// Build request URL from a copy of baseURL.
	reqURL := *c.baseURL
	p := "/" + strings.Trim(strings.Join(pathParts, "/"), "/")
	if c.apiVersion != "" && (len(pathParts) == 0 || (pathParts[0] != "_ping" && pathParts[0] != "version")) {
		p = "/" + c.apiVersion + p
	}
	reqURL.Path = path.Clean(reqURL.Path + p)
	reqURL.RawQuery = rawQuery

	req, err := http.NewRequestWithContext(ctx, method, reqURL.String(), body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return c.httpClient.Do(req)
}
