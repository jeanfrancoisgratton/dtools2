// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/18 06:37
// Original filename: src/rest/client.go

package rest

import (
	"crypto/tls"
	"crypto/x509"
	ce "github.com/jeanfrancoisgratton/customError/v2"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// NewClient builds an HTTP(S) client from Config.
func NewClient(cfg Config) (*Client, *ce.CustomError) {
	if cfg.Host == "" {
		return nil, &ce.CustomError{Code: 101, Title: "Unable to set client Host"}
	}
	scheme := strings.ToLower(strings.TrimSpace(cfg.Scheme))
	if scheme == "" {
		scheme = "https"
	}
	//base := &url.URL{Scheme: scheme, Host: cfg.Host}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}, //nolint:gosec
	}
	// If HTTPS, wire CA and/or client cert
	if scheme == "https" {
		// Custom CA
		if cfg.CAFile != "" {
			caPEM, err := os.ReadFile(cfg.CAFile)
			if err != nil {
				return nil, &ce.CustomError{Code: 102, Title: "Error reading the CA file", Message: err.Error()}
			}
			pool, err := x509.SystemCertPool()
			if err != nil || pool == nil {
				pool = x509.NewCertPool()
			}
			if ok := pool.AppendCertsFromPEM(caPEM); !ok {
				return nil, &ce.CustomError{Code: 103, Title: "Failed to append CA file"}
			}
			tr.TLSClientConfig.RootCAs = pool
		}
		// mTLS
		if cfg.ClientCertFile != "" && cfg.ClientKeyFile != "" {
			cert, err := tls.LoadX509KeyPair(cfg.ClientCertFile, cfg.ClientKeyFile)
			if err != nil {
				return nil, &ce.CustomError{Code: 104, Title: "Error loading client cert/key", Message: err.Error()}
			}
			tr.TLSClientConfig.Certificates = []tls.Certificate{cert}
		}
	}

	to := cfg.Timeout
	if to <= 0 {
		to = 20 * time.Second
	}
	apiPrefix := ""
	if v := strings.TrimSpace(cfg.APIVersion); v != "" {
		apiPrefix = "/" + strings.TrimLeft(v, "/")
	}

	return &Client{
		Http: &http.Client{
			Timeout:   to,
			Transport: tr,
		},
		//BaseURL:   base,
		APIprefix: apiPrefix,
	}, nil
}

// NewClientFromURL builds a REST client from a parsed URL and API version.
// Example: u := &url.URL{Scheme: "https", Host: "myreg:3281"}; rest.NewClientFromURL(u, "", 15*time.Second, tlsOpts)
func NewClientFromURL(u *url.URL, apiVersion string, timeout time.Duration, tlsOpts TLSOptions) (*Client, *ce.CustomError) {
	if u == nil {
		return nil, &ce.CustomError{Code: 104, Title: "Error creating the client", Message: "nil URL"}
	}
	cfg := Config{
		Host:               u.Host,
		Scheme:             u.Scheme,
		APIVersion:         apiVersion,
		Timeout:            timeout,
		CAFile:             tlsOpts.CAFile,
		ClientCertFile:     tlsOpts.ClientCertFile,
		ClientKeyFile:      tlsOpts.ClientKeyFile,
		InsecureSkipVerify: tlsOpts.InsecureSkipVerify,
	}
	return NewClient(cfg)
}

// NewHTTPClient builds a plain *http.Client that applies the same TLS behavior as Config.
// Useful for hitting Bearer token realms that may live on a different host than your base client.
func NewHTTPClient(tlsOpts TLSOptions, timeout time.Duration) (*http.Client, *ce.CustomError) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: tlsOpts.InsecureSkipVerify}, //nolint:gosec
	}

	// Custom CA
	if tlsOpts.CAFile != "" {
		caPEM, err := os.ReadFile(tlsOpts.CAFile)
		if err != nil {
			return nil, &ce.CustomError{Code: 101, Title: "Error reading CA file", Message: err.Error()}
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if ok := pool.AppendCertsFromPEM(caPEM); !ok {
			return nil, &ce.CustomError{Code: 106, Title: "Failed to append CA file"}
		}
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		}
		tr.TLSClientConfig.RootCAs = pool
	}

	// mTLS
	if tlsOpts.ClientCertFile != "" && tlsOpts.ClientKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsOpts.ClientCertFile, tlsOpts.ClientKeyFile)
		if err != nil {
			return nil, &ce.CustomError{Code: 106, Title: "Error loading cert/key file", Message: err.Error()}
		}
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		}
		tr.TLSClientConfig.Certificates = []tls.Certificate{cert}
	}

	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: tr,
	}, nil
}

// Helper: parse a string URL and pass to NewClientFromURL.
func NewClientFromURLString(rawURL, apiVersion string, timeout time.Duration, tlsOpts TLSOptions) (*Client, *ce.CustomError) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, &ce.CustomError{Code: 106, Title: "Error parsing URL", Message: err.Error()}
	}
	return NewClientFromURL(u, apiVersion, timeout, tlsOpts)
}
