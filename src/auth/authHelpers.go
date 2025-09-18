// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/18 06:24
// Original filename: src/auth/authHelpers.go

package auth

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// WhoAmIResult describes the stored auth for a registry.
type WhoAmIResult struct {
	Registry     string // normalized registry key used in config.json
	Mode         string // "basic", "token", "missing", "helper", "unknown"
	Username     string // for Mode == "basic"
	TokenPreview string // for Mode == "token" (non-sensitive short prefix)
}

// RemoveDockerConfigAuth deletes the auth entry for the given registry from config.json.
// Returns (true, nil) if an entry existed and was removed, (false, nil) if nothing to remove.
func RemoveDockerConfigAuth(registry string) (bool, error) {
	if registry == "" {
		return false, errors.New("registry must not be empty")
	}
	registry = NormalizeRegistry(registry)

	cfgDir, cfgPath := dockerConfigPath()
	// Fast path: if file doesn't exist, nothing to remove.
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return false, nil
	}

	root := map[string]json.RawMessage{}
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", cfgPath, err)
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &root); err != nil {
			return false, fmt.Errorf("parse existing config.json: %w", err)
		}
	}

	// Load current "auths"
	auths := map[string]map[string]string{}
	changed := false
	if raw, ok := root["auths"]; ok && len(raw) > 0 {
		_ = json.Unmarshal(raw, &auths) // be tolerant; if it fails, treat as empty
		if _, ok := auths[registry]; ok {
			delete(auths, registry)
			changed = true
		}
	}

	if !changed {
		// Nothing to do.
		return false, nil
	}

	// Re-write "auths"
	authsRaw, err := json.Marshal(auths)
	if err != nil {
		return false, fmt.Errorf("marshal auths: %w", err)
	}
	root["auths"] = authsRaw

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return false, fmt.Errorf("marshal config.json: %w", err)
	}

	tmpPath := cfgPath + ".tmp"
	if err := os.WriteFile(tmpPath, out, 0o600); err != nil {
		return false, fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmpPath, cfgPath); err != nil {
		return false, fmt.Errorf("rename temp config: %w", err)
	}
	_ = os.MkdirAll(cfgDir, 0o700) // ensure perms exist; ignore error (path should already exist)

	return true, nil
}

// WhoAmI inspects the stored credentials for a registry.
// It does not reveal secrets. For token mode, it shows a short non-sensitive prefix.
func WhoAmI(registry string) (*WhoAmIResult, error) {
	if registry == "" {
		return nil, errors.New("registry must not be empty")
	}
	registry = NormalizeRegistry(registry)

	_, cfgPath := dockerConfigPath()
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return &WhoAmIResult{Registry: registry, Mode: "missing"}, nil
	}

	root := map[string]json.RawMessage{}
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cfgPath, err)
	}
	if len(b) == 0 {
		return &WhoAmIResult{Registry: registry, Mode: "missing"}, nil
	}
	if err := json.Unmarshal(b, &root); err != nil {
		return nil, fmt.Errorf("parse config.json: %w", err)
	}

	// Check for an explicit auth entry.
	var auths map[string]map[string]string
	if raw, ok := root["auths"]; ok && len(raw) > 0 {
		_ = json.Unmarshal(raw, &auths) // tolerant
		if ent, ok := auths[registry]; ok {
			if b64, ok := ent["auth"]; ok && b64 != "" {
				mode, user, preview := classifyAuth(b64)
				switch mode {
				case "token":
					return &WhoAmIResult{
						Registry:     registry,
						Mode:         "token",
						TokenPreview: preview,
					}, nil
				case "basic":
					return &WhoAmIResult{
						Registry: registry,
						Mode:     "basic",
						Username: user,
					}, nil
				default:
					return &WhoAmIResult{
						Registry: registry,
						Mode:     "unknown",
					}, nil
				}
			}
		}
	}

	// No explicit auths entry: see if a helper is configured
	// (we can't query the helper here; just surface that one is likely in use).
	type helperCfg struct {
		CredsStore  string            `json:"credsStore"`
		CredHelpers map[string]string `json:"credHelpers"`
	}
	var hc helperCfg
	_ = json.Unmarshal(b, &hc) // tolerant
	if hc.CredsStore != "" || len(hc.CredHelpers) > 0 {
		return &WhoAmIResult{
			Registry: registry,
			Mode:     "helper",
		}, nil
	}

	// Otherwise, it's simply missing.
	return &WhoAmIResult{Registry: registry, Mode: "missing"}, nil
}

// classifyAuth decodes base64("user:pass") and determines whether it's token-style or basic.
func classifyAuth(b64 string) (mode, username, tokenPreview string) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return "unknown", "", ""
	}
	plain := string(raw)
	// Split at the first colon only (password/token may contain colons).
	user, rest, found := strings.Cut(plain, ":")
	if !found {
		return "unknown", "", ""
	}
	if user == "token" {
		// Show only a short non-sensitive preview (first 6 chars + length).
		pfx := rest
		if len(pfx) > 6 {
			pfx = pfx[:6]
		}
		return "token", "", fmt.Sprintf("%s… (len=%d)", pfx, len(rest))
	}
	return "basic", user, ""
}
