// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/28 12:39
// Original filename: src/rest/restHelpers.go

package rest

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// parseDockerHost normalizes docker-style host strings into a URL.
func parseDockerHost(h string) (*url.URL, error) {
	if strings.HasPrefix(h, "unix://") {
		p := strings.TrimPrefix(h, "unix://")
		if p == "" {
			return nil, errors.New("empty unix socket path")
		}
		// Stash the socket path in Path; makeTransport will rewrite URL.
		return &url.URL{Scheme: "http+unix", Host: "unix", Path: p}, nil
	}
	if strings.HasPrefix(h, "tcp://") {
		// Docker's tcp:// implies plain HTTP unless user wrote https:// explicitly.
		return &url.URL{Scheme: "http", Host: strings.TrimPrefix(h, "tcp://")}, nil
	}
	u, err := url.Parse(h)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	return u, nil
}

// makeTransport builds a transport for unix/http/https.
// HTTPS uses system roots by default, optional mTLS/CA from tlsConfig.go.
func makeTransport(u *url.URL) (http.RoundTripper, error) {
	// Unix socket: rewrite to http://unix and dial the saved socket path.
	if u.Scheme == "http+unix" {
		sock := u.Path
		u.Scheme = "http"
		u.Host = "unix"
		u.Path = ""

		return &http.Transport{
			DisableCompression: true,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sock)
			},
		}, nil
	}

	// TCP HTTP/HTTPS
	tr := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DisableCompression:  true,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	if u.Scheme == "https" {
		// Uses system roots by default; augments with ~/.docker/{ca.pem,cert.pem,key.pem} if present.
		tcfg, err := LoadTLSconfig(u.Hostname())
		if err != nil {
			return nil, err
		}
		if tcfg.ServerName == "" {
			tcfg.ServerName = u.Hostname()
		}
		tr.TLSClientConfig = tcfg
	}
	return tr, nil
}

func normalizeVersion(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func trimPort(host string) string {
	// url.Host can be "host:port"
	if i := strings.LastIndex(host, ":"); i > -1 {
		// IPv6 with port like "[::1]:2376" will be handled by URL.Hostname() elsewhere.
		return host[:i]
	}
	return host
}
