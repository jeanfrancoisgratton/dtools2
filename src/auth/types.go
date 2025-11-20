// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/14 12:55
// Original filename: src/auth/types.go

package auth

import "time"

// DockerConfig represents ~/.docker/config.json (simplified).
type DockerConfig struct {
	Auths map[string]RegistryAuth `json:"auths,omitempty"`
}

// RegistryAuth is a single registry auth entry.
type RegistryAuth struct {
	Auth          string `json:"auth,omitempty"` // base64("username:password")
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Email         string `json:"email,omitempty"`
	IdentityToken string `json:"identitytoken,omitempty"`
}

// LoginOptions describes how to log in to a registry.
type LoginOptions struct {
	// Registry can be:
	//   - "registry-1.docker.io"
	//   - "my-registry.example.com:5000"
	//   - "https://my-registry.example.com"
	//   - "http://insecure-registry.example.com:5000"
	//
	// If no scheme is present, HTTPS is assumed for the HTTP request.
	Registry string
	Username string
	Password string

	// Insecure, if true, disables TLS verification when using HTTPS.
	// Does NOT affect plain HTTP registries.
	Insecure bool

	// CACertPath, if non-empty, is used to build a custom Root CA pool
	// for the registry connection.
	CACertPath string

	// Timeout for the login HTTP request. If zero, a sane default is used.
	Timeout time.Duration
}
