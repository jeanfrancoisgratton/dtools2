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

// parseDockerHost normalizes docker-style host strings into a URL and optional unix socket path.
// For unix sockets we normalize baseURL to http://unix and return unixSock for dialing.
func parseDockerHost(h string) (*url.URL, string, error) {
	if strings.HasPrefix(h, "unix://") {
		sock := strings.TrimPrefix(h, "unix://")
		if sock == "" {
			return nil, "", errors.New("empty unix socket path")
		}
		// Base URL used to build requests. Dialer will use unixSock.
		return &url.URL{Scheme: "http", Host: "unix"}, sock, nil
	}
	if strings.HasPrefix(h, "tcp://") {
		return &url.URL{Scheme: "http", Host: strings.TrimPrefix(h, "tcp://")}, "", nil
	}
	u, err := url.Parse(h)
	if err != nil {
		return nil, "", err
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	return u, "", nil
}

// makeTransport builds a transport for unix/http/https.
// HTTPS uses system roots by default and augments with ~/.docker/{ca.pem,cert.pem,key.pem} if present.
func makeTransport(scheme, host, unixSock string) (http.RoundTripper, error) {
	if unixSock != "" {
		return &http.Transport{
			DisableCompression: true,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", unixSock)
			},
		}, nil
	}

	tr := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DisableCompression:  true,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	if scheme == "https" {
		tcfg, err := LoadTLSconfig(host)
		if err != nil {
			return nil, err
		}
		if tcfg.ServerName == "" {
			tcfg.ServerName = host
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
