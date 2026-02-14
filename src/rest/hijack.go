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

// Hijack opens a raw connection to the daemon, sends an HTTP/1.1 request, and returns the
// underlying connection if the server accepts the request.
//
// For upgrade endpoints (attach/exec), the returned Reader is positioned at the beginning of the
// raw multiplexed/TTY stream.
//
// Timeout strategy:
//   - Dial + TLS handshake + initial HTTP response headers are bounded by the client's
//     fast-fail timeout.
//   - Once the headers are read and the call succeeds, deadlines are cleared so the
//     hijacked stream can run indefinitely (until the caller closes the connection or
//     the provided context is canceled).
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

	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}

	// Ensure we close on errors.
	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)

	// Bound the handshake phase (write request + read status line + headers).
	handshake := c.effectiveFastFail(ctx)
	if handshake > 0 {
		_ = conn.SetDeadline(time.Now().Add(handshake))
	}
	defer func() {
		// Clear deadline for the streaming phase (success path).
		if handshake > 0 {
			_ = conn.SetDeadline(time.Time{})
		}
	}()

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
			// Clear deadlines now; the caller will stream until it is done.
			if handshake > 0 {
				_ = conn.SetDeadline(time.Time{})
			}
			return &HijackedConn{Conn: conn, Reader: br, Header: respHdr, Code: code}, nil
		}
	} else {
		if code >= 200 && code < 300 {
			if handshake > 0 {
				_ = conn.SetDeadline(time.Time{})
			}
			return &HijackedConn{Conn: conn, Reader: br, Header: respHdr, Code: code}, nil
		}
	}

	// Error: read a bounded amount so we never hang.
	// Keep the existing handshake deadline in place (or set one if unset).
	if handshake > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(handshake))
	}
	b, _ := io.ReadAll(io.LimitReader(br, 64*1024))
	_ = conn.Close()

	msg := strings.TrimSpace(string(b))
	if msg != "" {
		return nil, fmt.Errorf("%s %s returned HTTP %d: %s", method, finalPath, code, msg)
	}
	return nil, fmt.Errorf("%s %s returned HTTP %d", method, finalPath, code)
}

func (c *Client) effectiveFastFail(ctx context.Context) time.Duration {
	fast := c.fastFailTimeout
	if fast <= 0 {
		fast = 30 * time.Second
	}
	if ctx == nil {
		return fast
	}
	if dl, ok := ctx.Deadline(); ok {
		remain := time.Until(dl)
		if remain > 0 && remain < fast {
			return remain
		}
	}
	return fast
}

func (c *Client) dial(ctx context.Context) (net.Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	fast := c.effectiveFastFail(ctx)
	d := &net.Dialer{Timeout: fast, KeepAlive: 30 * time.Second}

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

	raw, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(c.baseURL.Scheme, "https") {
		return raw, nil
	}

	// TLS: reuse the same TLS config as the HTTP transport.
	tlsCfg, err := c.tlsConfigForDial(host)
	if err != nil {
		_ = raw.Close()
		return nil, err
	}

	tlsConn := tls.Client(raw, tlsCfg)

	// Ensure TLS handshake is bounded even if ctx has no deadline.
	hctx := ctx
	var cancel context.CancelFunc
	if dl, ok := ctx.Deadline(); !ok || time.Until(dl) > fast {
		hctx, cancel = context.WithTimeout(ctx, fast)
	}
	if cancel != nil {
		defer cancel()
	}

	if err := tlsConn.HandshakeContext(hctx); err != nil {
		_ = raw.Close()
		return nil, err
	}
	return tlsConn, nil
}

func (c *Client) tlsConfigForDial(hostport string) (*tls.Config, error) {
	t, ok := c.httpClient.Transport.(*http.Transport)
	if !ok {
		return nil, errors.New("unexpected transport type")
	}
	cfg := t.TLSClientConfig
	if cfg == nil {
		cfg = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	clone := cfg.Clone()
	if clone.ServerName == "" {
		clone.ServerName = stripPort(hostport)
	}
	return clone, nil
}

func stripPort(hostport string) string {
	if h, _, err := net.SplitHostPort(hostport); err == nil {
		return h
	}
	return hostport
}
