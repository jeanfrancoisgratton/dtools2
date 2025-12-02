// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/14 08:10
// Original filename: src/rest/auth.go

package rest

import (
	"os"
)

// ConfigFromEnv is a helper if later I eventually want to mirror DOCKER_* envs more closely.
func ConfigFromEnv() Config {
	// This is intentionally minimal for now. You can extend it later.
	host := os.Getenv("DOCKER_HOST")
	if host == "" {
		host = "unix:///var/run/docker.sock"
	}
	return Config{
		Host: host,
	}
}
