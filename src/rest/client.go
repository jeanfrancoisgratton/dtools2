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

	fastFail := cfg.FastFailTimeout
	if fastFail <= 0 {
		fastFail = time.Duration(FastFailTimeoutSeconds) * time.Second
		if fastFail <= 0 {
			fastFail = 30 * time.Second
		}
	}

	// Session timeout may be zero (explicitly disabled).
	sessionTimeout := cfg.SessionTimeout
	if sessionTimeout < 0 {
		sessionTimeout = 0
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

		dialer := &net.Dialer{Timeout: fastFail}
		transport = &http.Transport{
			Proxy: nil,
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return dialer.DialContext(ctx, "unix", unixPath)
			},
			// This matters even for unix sockets: it prevents hanging forever
			// while waiting for the first headers.
			ResponseHeaderTimeout: fastFail,
			TLSHandshakeTimeout:   fastFail,
			DisableCompression:    false,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
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

		dialer := &net.Dialer{Timeout: fastFail, KeepAlive: 30 * time.Second}
		transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, addr)
			},
			TLSClientConfig:       tlsConfig,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   fastFail,
			ResponseHeaderTimeout: fastFail,
			ExpectContinueTimeout: 1 * time.Second,
			DisableCompression:    false,
		}

		baseURL = u
	}

	// IMPORTANT: http.Client.Timeout is deliberately disabled.
	// We enforce timeouts per request with contexts so streaming operations
	// (pull/build/cp/save/load) are not killed mid-transfer.
	httpClient := &http.Client{Transport: transport, Timeout: 0}

	return &Client{
		httpClient:      httpClient,
		baseURL:         baseURL,
		apiVersion:      strings.TrimSpace(cfg.APIVersion),
		fastFailTimeout: fastFail,
		sessionTimeout:  sessionTimeout,
		isUnix:          isUnix,
		unixPath:        unixPath,
	}, nil
}

// cancelOnClose wraps a response body so we can cancel the associated context
// once the caller closes the body.
// This avoids leaking timers for context.WithTimeout() used inside Do().
//
// NOTE: Do() only uses this wrapper when it creates its own timeout context.
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c cancelOnClose) Close() error {
	// Cancel first to stop timers and allow any pending reads to unblock.
	if c.cancel != nil {
		c.cancel()
	}
	return c.ReadCloser.Close()
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
	// Always clear query; url.URL is a struct copy so this is safe.
	u.RawQuery = ""
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	// Apply a per-request timeout if the caller did not set a deadline.
	reqCtx := ctx
	if reqCtx == nil {
		reqCtx = context.Background()
	}

	var cancel context.CancelFunc
	if _, hasDeadline := reqCtx.Deadline(); !hasDeadline {
		// Fast-fail for /version negotiation.
		if path == "/version" {
			reqCtx, cancel = context.WithTimeout(reqCtx, c.fastFailTimeout)
		} else if !c.isNoTimeoutEndpoint(path, query) {
			// Session timeout for finite operations.
			if c.sessionTimeout > 0 {
				reqCtx, cancel = context.WithTimeout(reqCtx, c.sessionTimeout)
			}
		}
	}

	req, err := http.NewRequestWithContext(reqCtx, method, u.String(), body)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, err
	}

	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		return nil, err
	}

	// If we created a timeout context, ensure it gets canceled when the caller
	// is done with the body.
	if cancel != nil {
		resp.Body = cancelOnClose{ReadCloser: resp.Body, cancel: cancel}
	}
	return resp, nil
}

// isNoTimeoutEndpoint returns true for operations that are expected to stream
// indefinitely under normal usage, so applying a session timeout would be wrong.
func (c *Client) isNoTimeoutEndpoint(path string, query url.Values) bool {
	// Events stream.
	if path == "/events" {
		return true
	}

	// Container logs: if follow=1|true, the request is intentionally unbounded.
	if strings.Contains(path, "/logs") {
		v := ""
		if query != nil {
			v = strings.ToLower(query.Get("follow"))
		}
		if v == "1" || v == "true" {
			return true
		}
	}

	// Container stats: by default stream=true, which is unbounded.
	if strings.Contains(path, "/stats") {
		if query == nil {
			return true
		}
		v := strings.ToLower(query.Get("stream"))
		if v == "" || v == "1" || v == "true" {
			return true
		}
	}

	// Wait can legitimately run as long as the container runs.
	if strings.HasSuffix(path, "/wait") {
		return true
	}

	return false
}

// SocketPath returns the Unix socket path, if using a Unix transport.
func (c *Client) SocketPath() string {
	if !c.isUnix {
		return ""
	}
	return c.unixPath
}
