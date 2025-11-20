// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/14 08:13
// Original filename: src/auth/auth.go

// dtools2
// auth/auth.go
// Docker registry auth management, honoring ~/.docker/config.json.

package auth

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
)

// LoadDockerConfig loads ~/.docker/config.json (or $DOCKER_CONFIG/config.json).
// Returns an empty config if the file does not exist.
func LoadDockerConfig() (*DockerConfig, string, error) {
	configDir := os.Getenv("DOCKER_CONFIG")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, "", fmt.Errorf("cannot determine user home: %w", err)
		}
		configDir = filepath.Join(home, ".docker")
	}

	path := filepath.Join(configDir, "config.json")

	cfg := &DockerConfig{
		Auths: make(map[string]RegistryAuth),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No file yet; treat as empty config.
			return cfg, path, nil
		}
		return nil, "", fmt.Errorf("failed to read %s: %w", path, err)
	}

	if len(data) == 0 {
		return cfg, path, nil
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, "", fmt.Errorf("failed to unmarshal %s: %w", path, err)
	}

	if cfg.Auths == nil {
		cfg.Auths = make(map[string]RegistryAuth)
	}

	return cfg, path, nil
}

// SaveDockerConfig writes the config back, preserving directory structure.
func SaveDockerConfig(cfg *DockerConfig, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create docker config dir %q: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal docker config: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("failed to write temp config %q: %w", tmp, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to move temp config into place: %w", err)
	}

	return nil
}

// UpdateAuth updates or adds credentials for a registry in config.json.
// Existing entries are updated only if the credentials differ.
// auth/auth.go

func UpdateAuth(registry, username, password string) error {
	cfg, path, err := LoadDockerConfig()
	if err != nil {
		return err
	}

	if cfg.Auths == nil {
		cfg.Auths = make(map[string]RegistryAuth)
	}

	authStr := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))

	// Preserve any existing non-credential fields (email, identitytoken, etc.)
	entry := cfg.Auths[registry]
	if entry.Auth == authStr {
		// Already up to date; no change needed.
		return nil
	}

	entry.Auth = authStr
	entry.Username = "" // scrub any old cleartext if present
	entry.Password = "" // scrub any old cleartext if present

	cfg.Auths[registry] = entry

	return SaveDockerConfig(cfg, path)
}

// BuildRegistryAuthHeader builds the X-Registry-Auth header value for the given registry.
// This is the base64-encoded JSON payload expected by the daemon.
// auth/config.go

// BuildRegistryAuthHeader builds the X-Registry-Auth header value for the given registry.
// This is the base64-encoded JSON payload expected by the daemon.
func BuildRegistryAuthHeader(registry string) (string, error) {
	cfg, _, err := LoadDockerConfig()
	if err != nil {
		return "", err
	}

	// Try several key variants: raw, https://, http://
	keys := []string{
		registry,
		"https://" + registry,
		"http://" + registry,
	}

	var entry RegistryAuth
	found := false
	for _, k := range keys {
		if e, ok := cfg.Auths[k]; ok {
			entry = e
			found = true
			registry = k // use the same key value as serveraddress
			break
		}
	}

	if !found {
		return "", fmt.Errorf("no auth entry for registry %q in config.json", registry)
	}

	// Derive username/password from either explicit fields or from Auth.
	username := entry.Username
	password := entry.Password

	if username == "" && password == "" && entry.Auth != "" {
		decoded, err := base64.StdEncoding.DecodeString(entry.Auth)
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				username = parts[0]
				password = parts[1]
			}
		}
	}

	if username == "" && entry.IdentityToken == "" {
		return "", fmt.Errorf("no usable credentials for registry %q in config.json", registry)
	}

	// Docker expects base64(JSON) with username/password/serveraddress/identitytoken.
	payload := map[string]string{
		"username":      username,
		"password":      password,
		"serveraddress": registry,
	}

	if entry.IdentityToken != "" {
		payload["identitytoken"] = entry.IdentityToken
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal registry auth payload: %w", err)
	}

	return base64.URLEncoding.EncodeToString(data), nil
}

// Optional: simple secret hashing utility if you ever want to avoid storing raw passwords.
// Not used directly right now.
func hashSecret(secret string) string {
	salt := []byte("dtools2-static-salt") // if you ever use this for real, make it app-specific + random
	return base64.StdEncoding.EncodeToString(
		pbkdf2.Key([]byte(secret), salt, 4096, 32, sha3.New256),
	)
}
