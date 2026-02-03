// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/31 02:46
// Original filename: src/registry/registryconfig.go

package registry

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"strings"
)

type dockerConfig struct {
	Auths map[string]dockerAuthEntry `json:"auths"`
}

type dockerAuthEntry struct {
	Auth string `json:"auth"`
}

func loadDockerConfig(path string) (*dockerConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg dockerConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (cfg *dockerConfig) CredentialsProvider() CredentialsProvider {
	// Build a lookup map keyed by host, so we can match regardless of scheme/path in config keys
	host2creds := map[string][2]string{}

	for k, v := range cfg.Auths {
		user, pass, ok := decodeDockerAuth(v.Auth)
		if !ok {
			continue
		}

		// k may be "https://index.docker.io/v1/" or "https://nexus:9820" or "nexus:9820"
		host := extractHostFromDockerAuthKey(k)
		if host == "" {
			continue
		}
		host2creds[host] = [2]string{user, pass}

		// Special-case: Docker Hub mapping
		// docker config often stores creds under "index.docker.io/v1", but v2 registry host is "registry-1.docker.io"
		if strings.Contains(host, "index.docker.io") {
			host2creds["registry-1.docker.io"] = [2]string{user, pass}
			host2creds["docker.io"] = [2]string{user, pass}
		}
	}

	return func(registryHost string) (string, string, bool) {
		registryHost = strings.TrimSpace(registryHost)
		if registryHost == "" {
			return "", "", false
		}

		if creds, ok := host2creds[registryHost]; ok {
			return creds[0], creds[1], true
		}

		// Try without port if exact not found (optional convenience)
		if h, _, ok2 := strings.Cut(registryHost, ":"); ok2 {
			if creds, ok := host2creds[h]; ok {
				return creds[0], creds[1], true
			}
		}

		return "", "", false
	}
}

func decodeDockerAuth(auth string) (user, pass string, ok bool) {
	auth = strings.TrimSpace(auth)
	if auth == "" {
		return "", "", false
	}
	raw, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return "", "", false
	}
	s := string(raw)
	u, p, ok := strings.Cut(s, ":")
	if !ok {
		return "", "", false
	}
	return u, p, true
}

func extractHostFromDockerAuthKey(k string) string {
	k = strings.TrimSpace(k)
	if k == "" {
		return ""
	}

	// Common exact strings:
	// - https://index.docker.io/v1/
	// - https://nexus:9820
	// - nexus:9820
	if strings.HasPrefix(k, "http://") || strings.HasPrefix(k, "https://") {
		// crude parse without net/url to keep this tiny (host is between "://" and next "/")
		rest := k[strings.Index(k, "://")+3:]
		if i := strings.IndexByte(rest, '/'); i >= 0 {
			return rest[:i]
		}
		return rest
	}

	// Looks like host[:port]
	if i := strings.IndexByte(k, '/'); i >= 0 {
		return k[:i]
	}
	return k
}
