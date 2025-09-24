// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/18 06:15
// Original filename: src/auth/dockerConfig.go

package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v2"
)

// NormalizeRegistry maps common aliases to Docker's canonical keys.
func NormalizeRegistry(reg string) string {
	r := strings.TrimSpace(reg)
	r = strings.TrimRight(r, "/")
	switch r {
	case "docker.io", "index.docker.io":
		return "https://index.docker.io/v1/"
	case "https://docker.io", "http://docker.io", "https://index.docker.io", "http://index.docker.io":
		return "https://index.docker.io/v1/"
	default:
		return r
	}
}

// WriteDockerConfigAuth writes/overwrites auth for registry with username:password.
func WriteDockerConfigAuth(registry, username, password string) *ce.CustomError {
	if registry == "" {
		return &ce.CustomError{Code: 501, Title: "Error handling registry config", Message: "registry name is empty"}
	}
	registry = NormalizeRegistry(registry)
	return writeDockerConfigEntry(registry, AuthEntry{Auth: EncodeAuth(username, password)})
}

// WriteDockerConfigToken writes/overwrites auth for registry with token-based secret
// using Docker's common "token:<value>" convention (base64-encoded).
func WriteDockerConfigToken(registry, token string) *ce.CustomError {
	if registry == "" {
		return &ce.CustomError{Code: 501, Title: "Error handling registry config", Message: "registry name is empty"}
	}
	registry = NormalizeRegistry(registry)
	entry := AuthEntry{Auth: EncodeAuth("token", token)}
	return writeDockerConfigEntry(registry, entry)
}

func writeDockerConfigEntry(registry string, entry AuthEntry) *ce.CustomError {
	cfgDir, cfgPath := dockerConfigPath()
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return &ce.CustomError{Code: 502, Title: "Error creating the config directory/file", Message: err.Error()}
	}

	root := map[string]json.RawMessage{}
	if b, err := os.ReadFile(cfgPath); err == nil {
		if len(b) > 0 {
			if err := json.Unmarshal(b, &root); err != nil {
				return &ce.CustomError{Code: 503, Title: "Error parsing the config file", Message: err.Error()}
			}
		}
	} else if !os.IsNotExist(err) {
		return &ce.CustomError{Code: 503, Title: "Error reading the config file", Message: fmt.Sprintf("read %s: %s", cfgPath, err.Error())}
	}

	// Tolerant extraction of "auths"
	auths := map[string]AuthEntry{}
	if raw, ok := root["auths"]; ok && len(raw) > 0 {
		tmp := map[string]map[string]string{}
		if err := json.Unmarshal(raw, &tmp); err == nil {
			for k, v := range tmp {
				if a, ok := v["auth"]; ok {
					auths[k] = AuthEntry{Auth: a}
				}
			}
		} else {
			// malformed "auths" — reset to empty
			auths = map[string]AuthEntry{}
		}
	}

	// Overwrite/insert this registry
	auths[registry] = entry

	authsRaw, err := json.Marshal(auths)
	if err != nil {
		return &ce.CustomError{Code: 504, Title: "Error marshaling auths", Message: err.Error()}
	}
	root["auths"] = authsRaw

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return &ce.CustomError{Code: 504, Title: "Error marshaling config file", Message: err.Error()}
	}

	// Atomic-ish write
	tmpPath := cfgPath + ".tmp"
	if err := os.WriteFile(tmpPath, out, 0o600); err != nil {
		return &ce.CustomError{Code: 505, Title: "Error writing the temp file", Message: err.Error()}
	}
	if err := os.Rename(tmpPath, cfgPath); err != nil {
		return &ce.CustomError{Code: 505, Title: "Error renaming the temp file", Message: err.Error()}
	}
	return nil
}

// dockerConfigPath resolves the Docker config.json path, honoring $DOCKER_CONFIG.
func dockerConfigPath() (dir, path string) {
	if dc := os.Getenv("DOCKER_CONFIG"); dc != "" {
		return dc, filepath.Join(dc, "config.json")
	}
	home, _ := os.UserHomeDir()
	dir = filepath.Join(home, ".docker")
	return dir, filepath.Join(dir, "config.json")
}
