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

// Global timeout defaults.
//
// fast-fail: protects connect / TLS handshake / header waits (seconds)
// session-timeout: protects long-running operations that should complete (minutes)
//
// NOTE: a session timeout is intentionally NOT applied to operations that are
// meant to run indefinitely (e.g. logs -f, events, stats streaming, wait).
var FastFailTimeoutSeconds = 30
var SessionTimeoutMinutes = 60

// Client wraps an http.Client and knows how to talk to the Docker daemon
// via TCP (http/https) or a Unix socket, with an optional API version prefix.
type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	apiVersion string

	fastFailTimeout time.Duration
	sessionTimeout  time.Duration

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

	// FastFailTimeout is used to quickly fail when the daemon is unreachable or
	// stalls before responding (dial/TLS handshake/headers).
	// If <= 0, a default is applied.
	FastFailTimeout time.Duration

	// SessionTimeout is applied as a *per-request* timeout for operations that
	// are expected to complete (pull/build/cp/save/load/etc).
	// If 0, session timeouts are disabled (caller can still cancel via context).
	SessionTimeout time.Duration
}

// HijackedConn holds the underlying connection and a reader positioned right after the
// HTTP response headers (i.e., ready to read the raw stream).
type HijackedConn struct {
	Conn   net.Conn
	Reader *bufio.Reader
	Header http.Header
	Code   int
}
