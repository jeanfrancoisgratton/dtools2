// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/22 14:04
// Original filename: src/rest/types.go

package rest

import (
	"net/http"
	"net/url"
)

// Client wraps an http.Client and knows how to talk to the Docker daemon
// via TCP (http/https) or a Unix socket, with an optional API version prefix.
type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	apiVersion string

	isUnix   bool
	unixPath string
}
