// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/28 16:19
// Original filename: src/auth/authHelpers.go

package auth

import (
	"os"
	"path/filepath"
)

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docker", "config.json"), nil
}

func authsMap(cfg *dockerConfig) map[string]any {
	m, _ := cfg.raw["auths"].(map[string]any)
	if m == nil {
		m = map[string]any{}
		cfg.raw["auths"] = m
	}
	return m
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
