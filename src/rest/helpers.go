// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/02 02:51
// Original filename: src/rest/helpers.go

package rest

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// buildTLSConfig constructs a *tls.Config from the given settings.
// For Unix sockets, this is ignored by NewClient.
func buildTLSConfig(cfg Config) (*tls.Config, error) {
	if !cfg.UseTLS {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	// Root CAs
	if cfg.CACertPath != "" {
		caPEM, err := os.ReadFile(cfg.CACertPath)
		if err != nil {
			return nil, fmt.Errorf("unable to read CA cert %q: %w", cfg.CACertPath, err)
		}

		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("failed to parse CA cert %q", cfg.CACertPath)
		}
		tlsConfig.RootCAs = pool
	} else {
		// Use extras roots if available.
		sysPool, _ := x509.SystemCertPool()
		tlsConfig.RootCAs = sysPool
	}

	// Client certificate
	if cfg.CertPath != "" && cfg.KeyPath != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertPath, cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load client cert/key (%q, %q): %w", cfg.CertPath, cfg.KeyPath, err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

func joinURLPath(basePath, addPath string) string {
	if basePath == "" || basePath == "/" {
		return addPath
	}
	return strings.TrimRight(basePath, "/") + "/" + strings.TrimLeft(addPath, "/")
}

// DumpURL is a small helper for debugging.
func (c *Client) DumpURL(path string) string {
	u := *c.baseURL
	u.Path = joinURLPath(c.baseURL.Path, path)
	return u.String()
}

// NormalizePath is a helper to clean a host path (e.g. for certs).
func NormalizePath(p string) string {
	if p == "" {
		return ""
	}
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			p = filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}

// Shows where the client is connecting at (unix/localhost or remote daemon)
func ShowHost(uri string, showNow bool) string {
	//if uri == "" {
	//	uri = BuildConnectURI()
	//}
	if strings.HasPrefix(uri, "unix://") {
		uri = "localhost (unix socket)"
	}
	if showNow {
		fmt.Printf("\nDocker host is: %s.\n", hftx.White(uri))
	}
	return uri
}
