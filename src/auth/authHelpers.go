// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/18 06:24
// Original filename: src/auth/authHelpers.go

package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v2"
)

func tryBasic(ctx context.Context, client *http.Client, pingURL, user, pass string) *ce.CustomError {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, pingURL, nil)
	req.SetBasicAuth(user, pass)
	res, err := client.Do(req)
	if err != nil {
		return &ce.CustomError{Code: 301, Title: "Error Setting Basic Auth header", Message: err.Error()}
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)
	if res.StatusCode == http.StatusOK {
		return nil
	}
	return &ce.CustomError{Code: 302, Title: "Basic Auth failed", Message: fmt.Sprintf("Status Code: %d", res.Status)}
}

// parseAuthChallenge parses WWW-Authenticate headers like:
// Bearer realm="https://auth...",service="registry...",scope="repository:*,pull"
func parseAuthChallenge(h string) (scheme string, params map[string]string) {
	params = map[string]string{}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) == 0 {
		return "", params
	}
	scheme = strings.TrimSpace(parts[0])
	if len(parts) == 1 {
		return scheme, params
	}
	for _, kv := range splitCommaKVs(parts[1]) {
		k, v, _ := strings.Cut(kv, "=")
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"`)
		if k != "" {
			params[k] = v
		}
	}
	return scheme, params
}

func splitCommaKVs(s string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	for _, r := range s {
		switch r {
		case '"':
			inQuote = !inQuote
			cur.WriteRune(r)
		case ',':
			if inQuote {
				cur.WriteRune(r)
			} else {
				out = append(out, strings.TrimSpace(cur.String()))
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, strings.TrimSpace(cur.String()))
	}
	return out
}

// ensureHTTPS respects an explicit scheme if the caller already provided http:// or https://.
// If no scheme is present, it defaults to https://.
func ensureHTTPS(registry string) string {
	u := registry
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	return u
}

// EncodeAuth returns base64("username:password") as stored in Docker's config.json.
func EncodeAuth(username, password string) string {
	raw := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// classifyAuth decodes base64("user:pass") and determines whether it's token-style or basic.
func classifyAuth(b64 string) (mode, username, tokenPreview string) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return "unknown", "", ""
	}
	plain := string(raw)
	user, rest, found := strings.Cut(plain, ":")
	if !found {
		return "unknown", "", ""
	}
	if user == "token" {
		pfx := rest
		if len(pfx) > 6 {
			pfx = pfx[:6]
		}
		return "token", "", fmt.Sprintf("%s… (len=%d)", pfx, len(rest))
	}
	return "basic", user, ""
}
