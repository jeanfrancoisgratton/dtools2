// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/28 12:39
// Original filename: src/auth/authSetup.go

package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"
)

func Load() (*dockerConfig, error) {
	p, err := configPath()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &dockerConfig{raw: map[string]any{"auths": map[string]any{}}}, nil
		}
		return nil, err
	}
	var raw map[string]any
	if len(b) == 0 {
		raw = map[string]any{}
	} else if err := json.Unmarshal(b, &raw); err != nil {
		// Corrupt file: keep only auths to avoid data loss
		raw = map[string]any{}
	}
	// Ensure auths map exists
	if _, ok := raw["auths"]; !ok {
		raw["auths"] = map[string]any{}
	}
	return &dockerConfig{raw: raw}, nil
}

func Save(cfg *dockerConfig) error {
	p, err := configPath()
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(cfg.raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}

func NormalizeRegistry(server string) string {
	s := strings.TrimSpace(server)
	if s == "" {
		return "https://index.docker.io/v1/"
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		s = "https://" + s
	}
	return s
}

func SetUserPass(cfg *dockerConfig, server, user, pass string) {
	key := NormalizeRegistry(server)
	entry := authEntry{Auth: base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))}
	auths := authsMap(cfg)
	auths[key] = entry
}

func SetToken(cfg *dockerConfig, server, token string) {
	key := NormalizeRegistry(server)
	entry := authEntry{IdentityToken: token}
	auths := authsMap(cfg)
	auths[key] = entry
}

func Logout(cfg *dockerConfig, server string) bool {
	key := NormalizeRegistry(server)
	auths := authsMap(cfg)
	if _, ok := auths[key]; ok {
		delete(auths, key)
		return true
	}
	return false
}
