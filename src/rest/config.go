// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/14 08:10
// Original filename: src/rest/auth.go

package rest

import "time"

// Config holds the connection parameters for the REST client.
type Config struct {
	Host       string // e.g. "", unix:///var/run/docker.sock, tcp://host:2376, https://host:2376
	APIVersion string // e.g. "1.43"; empty means "negotiate"

	UseTLS             bool
	CACertPath         string
	CertPath           string
	KeyPath            string
	InsecureSkipVerify bool

	Timeout time.Duration // optional; if zero, a sane default is used.
}
