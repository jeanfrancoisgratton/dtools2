// dtools2
// src/auth/registry.go
// Registry probing and login logic (Bearer/Basic) + config write

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// LoginAndStoreSmartWithClient authenticates against the registry using the provided HTTP client.
// It prefers Bearer token flows (if advertised), else falls back to Basic.
// On success, it stores the chosen auth under ~/.docker/config.json (or $DOCKER_CONFIG/config.json).
func LoginAndStoreSmartWithClient(ctx context.Context, client *http.Client, registry, username, password string) error {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	if registry == "" {
		return errors.New("registry must not be empty")
	}
	if username == "" {
		return errors.New("username must not be empty")
	}

	// Respect caller's scheme if present; otherwise default to HTTPS.
	registryURL := ensureHTTPS(registry)

	scheme, params, err := ProbeAuthScheme(ctx, client, registryURL)
	if err != nil {
		return fmt.Errorf("probe auth scheme: %w", err)
	}

	switch strings.ToLower(scheme) {
	case "none":
		// Open registry; store user:pass if provided
		if err := WriteDockerConfigAuth(registry, username, password); err != nil {
			return fmt.Errorf("write docker config: %w", err)
		}
		return nil

	case "basic":
		if err := tryBasic(ctx, client, strings.TrimRight(registryURL, "/")+"/v2/", username, password); err != nil {
			return fmt.Errorf("basic auth failed: %w", err)
		}
		if err := WriteDockerConfigAuth(registry, username, password); err != nil {
			return fmt.Errorf("write docker config: %w", err)
		}
		return nil

	case "bearer":
		token, _, _, err := FetchBearerToken(ctx, client, params, username, password)
		if err != nil {
			// Some registries advertise bearer but accept basic; try basic as a fallback.
			if be := tryBasic(ctx, client, strings.TrimRight(registryURL, "/")+"/v2/", username, password); be == nil {
				if err := WriteDockerConfigAuth(registry, username, password); err != nil {
					return fmt.Errorf("write docker config: %w", err)
				}
				return nil
			}
			return fmt.Errorf("bearer token fetch failed: %w", err)
		}
		if err := WriteDockerConfigToken(registry, token); err != nil {
			return fmt.Errorf("write docker config token: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unsupported auth scheme: %s", scheme)
	}
}

// LoginAndStoreSmart is a convenience wrapper using a default client.
func LoginAndStoreSmart(ctx context.Context, registry, username, password string) error {
	client := &http.Client{Timeout: 15 * time.Second}
	return LoginAndStoreSmartWithClient(ctx, client, registry, username, password)
}

// ProbeAuthScheme calls GET {registry}/v2/ and returns:
//   - "none"   if 200 OK (no auth required)
//   - "basic"  if 401 with Basic
//   - "bearer" if 401 with Bearer (and challenge params: realm, service, scope?)
//
// Returns params for bearer (realm/service/scope) when applicable.
func ProbeAuthScheme(ctx context.Context, client *http.Client, registry string) (scheme string, params map[string]string, err error) {
	base := ensureHTTPS(registry)
	pingURL := strings.TrimRight(base, "/") + "/v2/"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pingURL, nil)
	if err != nil {
		return "", nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer res.Body.Close()
	io.Copy(io.Discard, res.Body)

	switch res.StatusCode {
	case http.StatusOK:
		return "none", nil, nil
	case http.StatusUnauthorized:
		chal := res.Header.Get("Www-Authenticate")
		if chal == "" {
			// No explicit scheme; assume Basic.
			return "basic", nil, nil
		}
		s, p := parseAuthChallenge(chal)
		switch strings.ToLower(s) {
		case "basic":
			return "basic", nil, nil
		case "bearer":
			return "bearer", p, nil
		default:
			return "", nil, fmt.Errorf("unknown WWW-Authenticate scheme: %s", s)
		}
	default:
		return "", nil, fmt.Errorf("unexpected status from %s: %s", pingURL, res.Status)
	}
}

// FetchBearerToken performs the token exchange per the Bearer challenge.
// params should include params["realm"], and optionally ["service"], ["scope"].
func FetchBearerToken(ctx context.Context, client *http.Client, params map[string]string, user, pass string) (token string, expiresIn int, issuedAt string, err error) {
	realm := params["realm"]
	if realm == "" {
		return "", 0, "", errors.New("bearer challenge missing realm")
	}

	q := url.Values{}
	if svc := params["service"]; svc != "" {
		q.Set("service", svc)
	}
	if scope := params["scope"]; scope != "" {
		q.Set("scope", scope)
	}
	tokenURL := realm
	if enc := q.Encode(); enc != "" {
		tokenURL = realm + "?" + enc
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return "", 0, "", err
	}
	// Most token services accept Basic creds to exchange for a bearer token.
	if user != "" {
		req.SetBasicAuth(user, pass)
	}

	res, err := client.Do(req)
	if err != nil {
		return "", 0, "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4<<10))
		return "", 0, "", fmt.Errorf("token service response %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	var tr struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		IssuedAt    string `json:"issued_at"`
	}
	if err := json.NewDecoder(res.Body).Decode(&tr); err != nil {
		return "", 0, "", fmt.Errorf("decode token response: %w", err)
	}
	tok := tr.Token
	if tok == "" && tr.AccessToken != "" {
		tok = tr.AccessToken
	}
	if tok == "" {
		return "", 0, "", errors.New("token response missing token")
	}
	return tok, tr.ExpiresIn, tr.IssuedAt, nil
}

// --- helpers ---
