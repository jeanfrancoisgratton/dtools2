// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/28 16:28
// Original filename: src/rest/restTypes.go

package rest

import (
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL     *url.URL
	httpClient  *http.Client
	apiVersion  string
	forced      bool
	dialTimeout time.Duration
}

// VersionInfo is a minimal subset of /version.
type VersionInfo struct {
	APIVersion    string `json:"ApiVersion"`
	MinAPIVersion string `json:"MinAPIVersion"`
	Version       string `json:"Version"`
}
