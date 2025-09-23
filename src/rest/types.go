// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/19 22:50
// Original filename: src/rest/types.go

package rest

import (
	"net/http"
	"net/url"
	"time"
)

// Client is a small wrapper around http.Client + base URL building.
type Client struct {
	Http      *http.Client
	BaseURL   *url.URL // e.g., https://host:2376
	APIprefix string   // e.g., /v1.41
}

// TLSOptions mirrors the TLS-related fields of Config for convenience when constructing
// either a REST client from a URL or a plain *http.Client for out-of-band calls (e.g. token realms).
type TLSOptions struct {
	CAFile             string
	ClientCertFile     string
	ClientKeyFile      string
	InsecureSkipVerify bool
}

// Config holds connection options for Docker/Podman daemon over HTTP(S).
type Config struct {
	// Base host: "127.0.0.1:2375", "mydaemon:2376"
	Host string

	// Scheme: "https" (default) or "Http"
	Scheme string

	// API version prefix (e.g., "v1.41"). Empty means "no prefix".
	APIVersion string

	// HTTP timeouts
	Timeout time.Duration

	// TLS options (HTTPS only)
	CAFile             string // optional
	ClientCertFile     string // optional (for mTLS)
	ClientKeyFile      string // optional (for mTLS)
	InsecureSkipVerify bool   // allow self-signed/invalid (lab only)
}
