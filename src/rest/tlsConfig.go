// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/29 08:54
// Original filename: src/rest/tlsConfig.go

package rest

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"
)

func LoadTLSconfig(serverHost string) (*tls.Config, error) {
	certDir := os.Getenv("DOCKER_CERT_PATH")
	if certDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		certDir = filepath.Join(home, ".docker")
	}

	// CA pool
	var rootPool *x509.CertPool
	caPath := filepath.Join(certDir, "ca.pem")
	if fileExists(caPath) {
		b, err := os.ReadFile(caPath)
		if err != nil {
			return nil, err
		}
		rootPool = x509.NewCertPool()
		if !rootPool.AppendCertsFromPEM(b) {
			return nil, errors.New("failed to add CA cert")
		}
	}

	// Client cert
	var certs []tls.Certificate
	certPath := filepath.Join(certDir, "cert.pem")
	keyPath := filepath.Join(certDir, "key.pem")
	if fileExists(certPath) && fileExists(keyPath) {
		cl, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}
		certs = []tls.Certificate{cl}
	}

	cfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: certs,
		RootCAs:      rootPool, // nil => system roots
		ServerName:   trimPort(serverHost),
	}

	// Opt-in insecure mode (not Docker default). Use only if you know what you do.
	if os.Getenv("DOCKER_TLS_INSECURE") == "1" {
		cfg.InsecureSkipVerify = true // #nosec G402
	}

	return cfg, nil
}
