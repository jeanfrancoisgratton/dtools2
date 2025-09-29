// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/29 09:13
// Original filename: src/auth/authRegistryHandling.go

package auth

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// BuildXRegistryAuth builds the header value for X-Registry-Auth.
// If no credentials exist, returns base64("{}") for anonymous pulls.
func BuildXRegistryAuth(server string) (string, error) {
	cfg, err := Load()
	if err != nil {
		return "", err
	}
	key := NormalizeRegistry(server)

	// Defaults to anonymous
	payload := map[string]string{
		"serveraddress": key,
	}

	// Look up entry
	auths := authsMap(cfg)
	if raw, ok := auths[key]; ok {
		// raw could be map[string]any (decoded) or already our struct type
		switch v := raw.(type) {
		case map[string]any:
			if s, ok := v["identitytoken"].(string); ok && s != "" {
				payload["identitytoken"] = s
			}
			if s, ok := v["auth"].(string); ok && s != "" {
				// decode "user:pass"
				if up, err := base64.StdEncoding.DecodeString(s); err == nil {
					parts := strings.SplitN(string(up), ":", 2)
					if len(parts) == 2 {
						payload["username"] = parts[0]
						payload["password"] = parts[1]
					}
				}
			}
		default:
			// try JSON round-trip
			b, _ := json.Marshal(v)
			tmp := map[string]any{}
			_ = json.Unmarshal(b, &tmp)
			if s, ok := tmp["identitytoken"].(string); ok && s != "" {
				payload["identitytoken"] = s
			}
			if s, ok := tmp["auth"].(string); ok && s != "" {
				if up, err := base64.StdEncoding.DecodeString(s); err == nil {
					parts := strings.SplitN(string(up), ":", 2)
					if len(parts) == 2 {
						payload["username"] = parts[0]
						payload["password"] = parts[1]
					}
				}
			}
		}
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
