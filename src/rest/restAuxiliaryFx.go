// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/29 08:57
// Original filename: src/rest/restAuxiliaryFx.go

package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"
)

// Ping calls GET /_ping and expects 200 OK.
// Ping calls GET /_ping and expects 200 OK.
func (c *Client) Ping(ctx context.Context) error {
	if c.baseURL == nil {
		return errors.New("client not initialized: baseURL is nil")
	}
	u := *c.baseURL
	u.Path = path.Clean(u.Path + "/_ping")
	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("ping failed: %s", resp.Status)
	}
	return nil
}

// NegotiateAPIversion sets a compatible API version by calling /version.
func (c *Client) NegotiateAPIversion(ctx context.Context) error {
	u := *c.baseURL
	u.Path = path.Clean(u.Path + "/version")
	req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("version negotiation failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("version endpoint failed: %s", resp.Status)
	}
	var vi VersionInfo
	if err := decodeJSON(resp.Body, &vi); err != nil {
		return err
	}
	if vi.APIVersion == "" {
		c.apiVersion = ""
		return nil
	}
	c.apiVersion = "v" + strings.TrimPrefix(vi.APIVersion, "v")
	return nil
}
