// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/22 14:04
// Original filename: src/rest/types.go

package rest

import (
	"bufio"
	"context"
	"net"
	"net/http"
	"net/url"
	"time"
)

var QuietOutput = false
var Context context.Context
var ConnectURI string

// Client wraps an http.Client and knows how to talk to the Docker daemon
// via TCP (http/https) or a Unix socket, with an optional API version prefix.
type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	apiVersion string

	isUnix   bool
	unixPath string
}

// versionInfo matches the JSON returned by /version.
type versionInfo struct {
	ApiVersion    string `json:"ApiVersion"`
	MinAPIVersion string `json:"MinAPIVersion"`
	Version       string `json:"Version"`
}

// Config holds the connection parameters for the REST client.
type Config struct {
	Host       string // e.g. "", unix:///var/run/docker.sock, tcp://host:2376, https://host:2376
	APIVersion string // e.g. "1.43"; empty means "negotiate"

	UseTLS             bool
	CACertPath         string
	CertPath           string
	KeyPath            string
	InsecureSkipVerify bool

	Timeout time.Duration // optional; if zero, a sane default is used.
}

// HijackedConn holds the underlying connection and a reader positioned right after the
// HTTP response headers (i.e., ready to read the raw stream).
type HijackedConn struct {
	Conn   net.Conn
	Reader *bufio.Reader
	Header http.Header
	Code   int
}
