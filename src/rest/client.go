// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/14 08:11
// Original filename: src/rest/client.go

package rest

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// NewClient builds a Client from Config.
// If Host is empty, DOCKER_HOST or the standard Docker default is used.
func NewClient(cfg Config) (*Client, error) {
	host := cfg.Host
	if host == "" {
		host = os.Getenv("DOCKER_HOST")
		if host == "" {
			// Standard default for local Docker.
			host = "unix:///var/run/docker.sock"
		}
	}

	isUnix := strings.HasPrefix(host, "unix://")

	// Allow bare host[:port] (e.g. "vps:2475") and treat it as tcp://.
	if !isUnix && !strings.Contains(host, "://") {
		host = "tcp://" + host
	}
	if !QuietOutput && ConnectURI != "" {
		fmt.Println(fmt.Sprintf("%s: %s\n", hftx.InfoSign("Connected to"), hftx.Blue(host)))
	}

	var (
		transport *http.Transport
		baseURL   *url.URL
		unixPath  string
	)

	if isUnix {
		// Strip unix:// prefix and keep the socket path.
		unixPath = strings.TrimPrefix(host, "unix://")
		if unixPath == "" {
			return nil, fmt.Errorf("unix host %q has empty socket path", host)
		}

		transport = &http.Transport{
			Proxy: nil,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				// Keep the same 30s dial timeout as before.
				return net.DialTimeout("unix", unixPath, 30*time.Second)
			},
		}

		// Fake URL; only the path is used when we build requests.
		baseURL, _ = url.Parse("http://d")
	} else {
		u, err := url.Parse(host)
		if err != nil {
			return nil, fmt.Errorf("invalid host %q: %w", host, err)
		}
		if u.Host == "" {
			return nil, fmt.Errorf("host %q is missing hostname", host)
		}

		scheme := u.Scheme
		switch scheme {
		case "tcp":
			if cfg.UseTLS {
				scheme = "https"
			} else {
				scheme = "http"
			}
		case "http", "https":
			// Respect explicit scheme, UseTLS only controls TLS config.
		default:
			return nil, fmt.Errorf("unsupported scheme %q in host %q", scheme, host)
		}
		u.Scheme = scheme

		tlsConfig, err := buildTLSConfig(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to build TLS config: %w", err)
		}

		transport = &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			TLSClientConfig:     tlsConfig,
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		}

		baseURL = u
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
		apiVersion: strings.TrimSpace(cfg.APIVersion),
		isUnix:     isUnix,
		unixPath:   unixPath,
	}, nil
}

// Do issues an HTTP request to the daemon.
// `path` should be the API path, e.g. "/containers/json" or "/version".
// For most endpoints, a "/v<version>" prefix is automatically added.
// `/version` is called without a version prefix for negotiation.
func (c *Client) Do(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body io.Reader,
	headers http.Header,
) (*http.Response, error) {
	if path == "" || path[0] != '/' {
		path = "/" + path
	}

	// /version is unversioned; everything else gets /v<APIVersion>.
	finalPath := path
	if path != "/version" && c.apiVersion != "" {
		finalPath = "/v" + c.apiVersion + path
	}

	u := *c.baseURL
	u.Path = joinURLPath(c.baseURL.Path, finalPath)
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}

	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	return c.httpClient.Do(req)
}

// SocketPath returns the Unix socket path, if using a Unix transport.
func (c *Client) SocketPath() string {
	if !c.isUnix {
		return ""
	}
	return c.unixPath
}
