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

// Client is a thin Docker Engine REST client.
type Client struct {
	baseURL     *url.URL     // always http://unix or http[s]://host
	unixSock    string       // /var/run/docker.sock when using unix
	httpClient  *http.Client // transport set for unix/http/https
	apiVersion  string       // e.g. v1.45; empty => unversioned
	forced      bool
	dialTimeout time.Duration
}

// VersionInfo is a minimal subset of /version.
type VersionInfo struct {
	APIVersion    string `json:"ApiVersion"`
	MinAPIVersion string `json:"MinAPIVersion"`
	Version       string `json:"Version"`
}
