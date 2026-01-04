// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 19:54
// Original filename: src/rest/hijack.go

package rest

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Hijack opens a raw connection to the daemon, sends an HTTP request, and returns the
// underlying connection if the server accepts the request.
//
// For upgrade endpoints, the returned Reader is positioned at the beginning of the raw
// multiplexed/TTY stream.
//
// If expectUpgrade is true, Hijack automatically adds the standard Docker upgrade headers
// (Connection: Upgrade, Upgrade: tcp) unless already present.
func (c *Client) Hijack(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	headers http.Header,
	body []byte,
	expectUpgrade bool,
) (*HijackedConn, error) {
	if headers == nil {
		headers = http.Header{}
	}

	// Build final API path (including /v<version> prefix when applicable).
	if path == "" || path[0] != '/' {
		path = "/" + path
	}
	finalPath := path
	if path != "/version" && c.apiVersion != "" {
		finalPath = "/v" + c.apiVersion + path
	}
	reqPath := joinURLPath(c.baseURL.Path, finalPath)
	if len(query) > 0 {
		reqPath = reqPath + "?" + query.Encode()
	}

	// Dial.
	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}

	// Ensure we close on errors.
	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)

	// Host header is required for HTTP/1.1.
	host := c.baseURL.Host
	if host == "" {
		host = "docker"
	}
	if headers.Get("Host") == "" {
		headers.Set("Host", host)
	}

	// For hijacked endpoints, Docker expects these upgrade headers.
	if expectUpgrade {
		if headers.Get("Connection") == "" {
			headers.Set("Connection", "Upgrade")
		}
		if headers.Get("Upgrade") == "" {
			headers.Set("Upgrade", "tcp")
		}
	}

	if headers.Get("User-Agent") == "" {
		headers.Set("User-Agent", "dtools2")
	}
	if len(body) > 0 && headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "application/json")
	}
	if headers.Get("Content-Length") == "" {
		headers.Set("Content-Length", strconv.Itoa(len(body)))
	}

	// Write request.
	if _, err := fmt.Fprintf(bw, "%s %s HTTP/1.1\r\n", method, reqPath); err != nil {
		_ = conn.Close()
		return nil, err
	}
	for k, vals := range headers {
		for _, v := range vals {
			if _, err := fmt.Fprintf(bw, "%s: %s\r\n", k, v); err != nil {
				_ = conn.Close()
				return nil, err
			}
		}
	}
	if _, err := io.WriteString(bw, "\r\n"); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if len(body) > 0 {
		if _, err := bw.Write(body); err != nil {
			_ = conn.Close()
			return nil, err
		}
	}
	if err := bw.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	// Read status line.
	statusLine, err := br.ReadString('\n')
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	statusLine = strings.TrimRight(statusLine, "\r\n")
	parts := strings.SplitN(statusLine, " ", 3)
	if len(parts) < 2 {
		_ = conn.Close()
		return nil, fmt.Errorf("invalid HTTP response status line: %q", statusLine)
	}
	code, err := strconv.Atoi(parts[1])
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("invalid HTTP status code in %q: %w", statusLine, err)
	}

	// Read headers.
	tp := textproto.NewReader(br)
	mh, err := tp.ReadMIMEHeader()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	respHdr := http.Header(mh)

	// Success criteria:
	// - for hijacked/upgrade endpoints: Docker replies 101 Switching Protocols
	// - for normal endpoints: expect 2xx
	if expectUpgrade {
		if code == http.StatusSwitchingProtocols || (code >= 200 && code < 300) {
			return &HijackedConn{Conn: conn, Reader: br, Header: respHdr, Code: code}, nil
		}
	} else {
		if code >= 200 && code < 300 {
			return &HijackedConn{Conn: conn, Reader: br, Header: respHdr, Code: code}, nil
		}
	}

	// Error: read a bounded amount with a short deadline so we never hang on a stream.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	defer func() { _ = conn.SetReadDeadline(time.Time{}) }()

	b, _ := io.ReadAll(io.LimitReader(br, 64*1024))
	_ = conn.Close()

	msg := strings.TrimSpace(string(b))
	if msg != "" {
		return nil, fmt.Errorf("%s %s returned HTTP %d: %s", method, finalPath, code, msg)
	}
	return nil, fmt.Errorf("%s %s returned HTTP %d", method, finalPath, code)
}

func (c *Client) dial(ctx context.Context) (net.Conn, error) {
	d := &net.Dialer{Timeout: 30 * time.Second}

	if c.isUnix {
		if c.unixPath == "" {
			return nil, errors.New("unix socket path is empty")
		}
		return d.DialContext(ctx, "unix", c.unixPath)
	}

	host := c.baseURL.Host
	if host == "" {
		return nil, errors.New("tcp host is empty")
	}

	if strings.EqualFold(c.baseURL.Scheme, "https") {
		t, ok := c.httpClient.Transport.(*http.Transport)
		if !ok {
			return nil, errors.New("unexpected transport type")
		}
		tlsCfg := t.TLSClientConfig
		if tlsCfg == nil {
			tlsCfg = &tls.Config{MinVersion: tls.VersionTLS12}
		}

		cfg := tlsCfg.Clone()
		if cfg.ServerName == "" {
			cfg.ServerName = stripPort(host)
		}

		return tls.DialWithDialer(d, "tcp", host, cfg)
	}

	return d.DialContext(ctx, "tcp", host)
}

func stripPort(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
}
