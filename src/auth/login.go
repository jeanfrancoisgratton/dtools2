// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/14 12:54
// Original filename: src/auth/loginCommands.go

// Login to a Docker registry (TLS or not), verify credentials, and update
// ~/.docker/config.json accordingly.

package auth

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Login performs a "docker login"-like operation:
//
//  1. Resolves the registry URL (defaulting to HTTPS if no scheme).
//  2. Sends GET /v2/ with Basic auth.
//  3. If the registry accepts the credentials (2xx/3xx status), updates
//     ~/.docker/config.json via UpdateAuth().
//  4. Returns an error on any failure.
//
// This does NOT attempt to fully emulate Docker's bearer-token dance;
// it uses the common "GET /v2/ with Basic auth" pattern used by
// private registries and most Docker setups.
func Login(ctx context.Context, opts LoginOptions) error {
	if opts.Registry == "" {
		return fmt.Errorf("registry is required")
	}
	if opts.Username == "" {
		return fmt.Errorf("username is required")
	}
	if opts.Password == "" {
		return fmt.Errorf("password is required")
	}

	regURL, err := normalizeRegistryURL(opts.Registry)
	if err != nil {
		return fmt.Errorf("invalid registry %q: %w", opts.Registry, err)
	}

	httpClient, err := buildHTTPClientForRegistry(opts)
	if err != nil {
		return fmt.Errorf("failed to build HTTP client for registry: %w", err)
	}

	// Build /v2/ URL.
	u := *regURL
	u.Path = "/v2/"
	u.RawQuery = ""

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("failed to build login request: %w", err)
	}

	// Basic auth: docker login just checks credentials by hitting /v2.
	authHeader := "Basic " + base64.StdEncoding.EncodeToString(
		[]byte(opts.Username+":"+opts.Password),
	)
	req.Header.Set("Authorization", authHeader)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read a small body for debugging; not strictly needed.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	// Most registries return 200 on success, but some may redirect (3xx).
	// Treat 2xx and 3xx as success; 401 clearly means bad credentials.
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		// On success, write/update auth entry in ~/.docker/config.json.
		configKey := registryConfigKey(opts.Registry)
		if err := UpdateAuth(configKey, opts.Username, opts.Password); err != nil {
			return fmt.Errorf("login succeeded but failed to update config.json: %w", err)
		}
		return nil
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("registry login failed: unauthorized (401). response: %s", strings.TrimSpace(string(body)))
	}

	return fmt.Errorf("registry login failed: %s. response: %s", resp.Status, strings.TrimSpace(string(body)))
}

// normalizeRegistryURL ensures we have a proper URL object with a scheme.
// If no scheme is present, HTTPS is assumed.
func normalizeRegistryURL(reg string) (*url.URL, error) {
	reg = strings.TrimSpace(reg)
	if reg == "" {
		return nil, fmt.Errorf("empty registry")
	}

	if !strings.HasPrefix(reg, "http://") && !strings.HasPrefix(reg, "https://") {
		reg = "https://" + reg
	}

	u, err := url.Parse(reg)
	if err != nil {
		return nil, err
	}

	if u.Host == "" {
		return nil, fmt.Errorf("registry %q has no host", reg)
	}

	// Normalize path; we always overwrite it to "/v2/" later anyway.
	if u.Path == "" {
		u.Path = "/"
	}

	return u, nil
}

// buildHTTPClientForRegistry builds an *http.Client configured for TLS (if HTTPS)
// according to LoginOptions.
func buildHTTPClientForRegistry(opts LoginOptions) (*http.Client, error) {
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	transport := &http.Transport{}

	// Only build TLS config for HTTPS URLs; plain HTTP just uses default.
	if !strings.HasPrefix(strings.ToLower(opts.Registry), "http://") {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: opts.Insecure, //nolint:gosec // user explicitly requested insecure
			MinVersion:         tls.VersionTLS12,
		}

		// Custom CA, if provided.
		if opts.CACertPath != "" {
			caPEM, err := os.ReadFile(opts.CACertPath)
			if err != nil {
				return nil, fmt.Errorf("unable to read registry CA cert %q: %w", opts.CACertPath, err)
			}

			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(caPEM) {
				return nil, fmt.Errorf("failed to parse registry CA cert %q", opts.CACertPath)
			}
			tlsConfig.RootCAs = pool
		} else {
			// Use system roots if available.
			sysPool, _ := x509.SystemCertPool()
			tlsConfig.RootCAs = sysPool
		}

		transport.TLSClientConfig = tlsConfig
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}, nil
}

// registryConfigKey decides what key to use in config.json's "auths" map.
// Docker typically uses the registry host (sometimes with scheme).
// Here we normalize to just host[:port] without scheme.
func registryConfigKey(reg string) string {
	reg = strings.TrimSpace(reg)
	if reg == "" {
		return reg
	}

	// If it parses as a URL, use its Host.
	if strings.HasPrefix(reg, "http://") || strings.HasPrefix(reg, "https://") {
		if u, err := url.Parse(reg); err == nil && u.Host != "" {
			return u.Host
		}
	}

	// Otherwise, assume it's already `host[:port]`.
	return reg
}
