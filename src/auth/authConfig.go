// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/29 17:37
// Original filename: src/auth/authConfig.go

package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

func LookupBasic(cfg *dockerConfig, server string) (user, pass string, ok bool) {
	key := NormalizeRegistry(server)
	m := authsMap(cfg)
	raw, exists := m[key]
	if !exists {
		return "", "", false
	}
	x, _ := raw.(map[string]any)
	if x == nil {
		// try to JSON round-trip unknown type
		b, _ := json.Marshal(raw)
		_ = json.Unmarshal(b, &x)
	}
	if x == nil {
		return "", "", false
	}
	if s, ok2 := x["auth"].(string); ok2 && s != "" {
		if up, err := base64.StdEncoding.DecodeString(s); err == nil {
			parts := strings.SplitN(string(up), ":", 2)
			if len(parts) == 2 {
				return parts[0], parts[1], true
			}
		}
	}
	return "", "", false
}
