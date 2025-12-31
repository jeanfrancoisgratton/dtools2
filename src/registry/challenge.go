// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/31 02:45
// Original filename: src/registry/challenge.go

package registry

import (
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

func parseBearerChallenge(wwwAuth string) (bearerChallenge, *ce.CustomError) {
	wwwAuth = strings.TrimSpace(wwwAuth)
	if wwwAuth == "" {
		return bearerChallenge{}, &ce.CustomError{Title: "bearer challenge error", Message: "missing WWW-Authenticate header"}
	}

	// Typical: Bearer realm="...",service="...",scope="..."
	parts := strings.SplitN(wwwAuth, " ", 2)
	if len(parts) < 2 || !strings.EqualFold(parts[0], "Bearer") {
		return bearerChallenge{}, &ce.CustomError{Title: "bearer challenge error",
			Message: wwwAuth + " is not a Bearer challenge"}
	}

	params := parseAuthParams(parts[1])

	realm := params["realm"]
	if realm == "" {
		return bearerChallenge{}, &ce.CustomError{Title: "bearer challenge error",
			Message: "bearer challenge missing realm: " + wwwAuth}
	}

	return bearerChallenge{
		Realm:   realm,
		Service: params["service"],
		Scope:   params["scope"],
	}, nil
}

// Parses: key="value",key2="value2",key3=value3 ...
func parseAuthParams(s string) map[string]string {
	out := map[string]string{}
	s = strings.TrimSpace(s)
	if s == "" {
		return out
	}

	// Split on commas that are not inside quotes
	var cur strings.Builder
	inQuotes := false
	flush := func() {
		item := strings.TrimSpace(cur.String())
		cur.Reset()
		if item == "" {
			return
		}
		k, v, ok := strings.Cut(item, "=")
		if !ok {
			return
		}
		k = strings.ToLower(strings.TrimSpace(k))
		v = strings.TrimSpace(v)
		v = strings.Trim(v, `"`)
		out[k] = v
	}

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			inQuotes = !inQuotes
			cur.WriteByte(ch)
			continue
		}
		if ch == ',' && !inQuotes {
			flush()
			continue
		}
		cur.WriteByte(ch)
	}
	flush()

	return out
}
