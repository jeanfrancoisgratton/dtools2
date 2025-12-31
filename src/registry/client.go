// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/31 02:42
// Original filename: src/registry/client.go

package registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

func NewClient(registryURL string, opts ...Option) (*Client, *ce.CustomError) {
	u, err := parseRegistryURL(registryURL)
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL: u,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		},
		tokenCache: make(map[string]cachedToken),
	}

	// Default: load creds from ~/.docker/config.json (best-effort; no hard fail)
	_ = WithDockerConfigDefault()(c)

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(c); err != nil {
			return nil, &ce.CustomError{Title: "Options cannot be nil", Message: err.Error()}
		}
	}

	return c, nil
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) error {
		if hc == nil {
			return nil
		}
		c.httpClient = hc
		return nil
	}
}

// Useful with self-signed registries.

func WithInsecureSkipVerify(skip bool) Option {
	return func(c *Client) error {
		tr, ok := c.httpClient.Transport.(*http.Transport)
		if !ok || tr == nil {
			tr = &http.Transport{Proxy: http.ProxyFromEnvironment}
		}
		if tr.TLSClientConfig == nil {
			tr.TLSClientConfig = &tls.Config{}
		}
		tr.TLSClientConfig.InsecureSkipVerify = skip
		c.httpClient.Transport = tr
		return nil
	}
}

func WithCredentials(username, password string) Option {
	return func(c *Client) error {
		if username == "" {
			return nil
		}
		c.creds = func(_ string) (string, string, bool) {
			return username, password, true
		}
		return nil
	}
}

func WithDockerConfigDefault() Option {
	return func(c *Client) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil
		}
		return WithDockerConfig(filepath.Join(home, ".docker", "config.json"))(c)
	}
}

func WithDockerConfig(configPath string) Option {
	return func(c *Client) error {
		cfg, err := loadDockerConfig(configPath)
		if err != nil {
			// best-effort: do not fail client creation because docker config is missing/unreadable
			return nil
		}
		c.creds = cfg.CredentialsProvider()
		return nil
	}
}

func (c *Client) CatalogJSON(ctx context.Context, q url.Values) ([]byte, *ce.CustomError) {
	return c.getJSON(ctx, "/v2/_catalog", q)
}

func (c *Client) TagsJSON(ctx context.Context, repo string, q url.Values) ([]byte, error) {
	repo = strings.TrimLeft(repo, "/")
	if repo == "" {
		return nil, errors.New("repo name is empty")
	}
	return c.getJSON(ctx, "/v2/"+repo+"/tags/list", q)
}

func (c *Client) getJSON(ctx context.Context, path string, q url.Values) ([]byte, *ce.CustomError) {
	if q == nil {
		q = url.Values{}
	}

	// 1) First attempt: anonymous, no auth headers
	resp, err := c.do(ctx, http.MethodGet, path, q, nil)
	if err != nil {
		return nil, &ce.CustomError{Title: "Error handling http GET", Message: err.Error()}
	}
	defer resp.Body.Close()

	// Fast-path success
	if resp.StatusCode == http.StatusOK {
		return readAll(resp.Body)
	}

	// 2) If challenged, handle transparently then retry once
	if resp.StatusCode == http.StatusUnauthorized {
		chal := resp.Header.Get("WWW-Authenticate")
		// Drain body before retry (keep connections healthy)
		_, _ = io.Copy(io.Discard, resp.Body)

		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(chal)), "bearer") {
			authHeader, err := c.bearerAuthHeaderFromChallenge(ctx, chal)
			if err != nil {
				return nil, err
			}
			resp2, err := c.do(ctx, http.MethodGet, path, q, map[string]string{
				"Authorization": authHeader,
			})
			if err != nil {
				return nil, err
			}
			defer resp2.Body.Close()

			if resp2.StatusCode == http.StatusOK {
				return readAll(resp2.Body)
			}
			return nil, &ce.CustomError{Title: "Error in http response for path " + path, Message: "Returned status was " + resp.Status}

		} else if strings.HasPrefix(strings.ToLower(strings.TrimSpace(chal)), "basic") {
			// Basic challenge: retry with Basic if we have creds
			if c.creds != nil {
				user, pass, ok := c.creds(c.baseURL.Host)
				if ok {
					resp2, err := c.do(ctx, http.MethodGet, path, q, map[string]string{
						"Authorization": basicAuthHeader(user, pass),
					})
					if err != nil {
						return nil, err
					}
					defer resp2.Body.Close()

					if resp2.StatusCode == http.StatusOK {
						return readAll(resp2.Body)
					}
					return nil, &ce.CustomError{Title: "Error in http response for path " + path, Message: "Returned status was " + resp.Status}
				}
			}
		}
	}

	return nil, &ce.CustomError{Title: "Error in http response for path " + path, Message: "Returned status was " + resp.Status}
}

func (c *Client) do(ctx context.Context, method, path string, q url.Values, headers map[string]string) (*http.Response, *ce.CustomError) {
	u := *c.baseURL
	u.Path = path
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, &ce.CustomError{Title: "Error creating http request", Message: err.Error()}
	}
	req.Header.Set("Accept", "application/json")

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if rresp, derr := c.httpClient.Do(req); derr != nil {
		return nil, &ce.CustomError{Title: "Unable to execute request", Message: derr.Error()}
	} else {
		return rresp, nil
	}
}

func (c *Client) bearerAuthHeaderFromChallenge(ctx context.Context, wwwAuth string) (string, *ce.CustomError) {
	ch, err := parseBearerChallenge(wwwAuth)
	if err != nil {
		return "", err
	}

	// Cache key: realm|service|scope|registryHost
	cacheKey := strings.Join([]string{
		ch.Realm, ch.Service, ch.Scope, c.baseURL.Host,
	}, "|")

	// Use cached token if still valid
	if tok, ok := c.getCachedToken(cacheKey); ok {
		return "Bearer " + tok, nil
	}

	// Fetch a token
	tok, exp, err := c.fetchBearerToken(ctx, ch)
	if err != nil {
		return "", err
	}
	c.setCachedToken(cacheKey, tok, exp)

	return "Bearer " + tok, nil
}

func (c *Client) fetchBearerToken(ctx context.Context, ch bearerChallenge) (token string, expiry time.Time, customError *ce.CustomError) {
	realmURL, err := url.Parse(ch.Realm)
	if err != nil || realmURL.Scheme == "" || realmURL.Host == "" {
		return "", time.Time{}, &ce.CustomError{Title: "Invalid bearer realm url", Message: "url is " + ch.Realm}
	}

	q := realmURL.Query()
	if ch.Service != "" {
		q.Set("service", ch.Service)
	}
	if ch.Scope != "" {
		q.Set("scope", ch.Scope)
	}
	realmURL.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, realmURL.String(), nil)
	if err != nil {
		return "", time.Time{}, &ce.CustomError{Title: "Invalid http request", Message: err.Error()}
	}
	req.Header.Set("Accept", "application/json")

	// If we have creds for registry host, use them for token exchange (common)
	if c.creds != nil {
		user, pass, ok := c.creds(c.baseURL.Host)
		if ok {
			req.Header.Set("Authorization", basicAuthHeader(user, pass))
			// Some token services accept/expect "account"
			if user != "" {
				q2 := realmURL.Query()
				q2.Set("account", user)
				realmURL.RawQuery = q2.Encode()
				req.URL.RawQuery = realmURL.RawQuery
			}
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", time.Time{}, &ce.CustomError{Title: "Unable to execute http request", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, &ce.CustomError{Title: "Http request returned an error on path " + realmURL.Path,
			Message: "http response: " + resp.Status}
	}

	body, raerr := readAll(resp.Body)
	if err != nil {
		return "", time.Time{}, raerr
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return "", time.Time{}, &ce.CustomError{Title: "Error unmarshaling JSON", Message: err.Error()}
	}

	tok := tr.Token
	if tok == "" {
		tok = tr.AccessToken
	}
	if tok == "" {
		return "", time.Time{},
			&ce.CustomError{Title: "Error unmarshaling JSON", Message: "token endpoint returned empty token"}
	}

	// Expiry: use expires_in if provided; otherwise cache briefly
	exp := time.Now().Add(60 * time.Second)
	if tr.ExpiresIn > 0 {
		// subtract a little to avoid edge expiry
		exp = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second).Add(-10 * time.Second)
	}

	return tok, exp, nil
}

func (c *Client) getCachedToken(key string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ct, ok := c.tokenCache[key]
	if !ok {
		return "", false
	}
	if !ct.expiry.IsZero() && time.Now().After(ct.expiry) {
		delete(c.tokenCache, key)
		return "", false
	}
	return ct.token, true
}

func (c *Client) setCachedToken(key, token string, expiry time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokenCache[key] = cachedToken{token: token, expiry: expiry}
}

func parseRegistryURL(raw string) (*url.URL, *ce.CustomError) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, &ce.CustomError{Title: "Error parsing the registry URL", Message: "registry url is empty"}
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		// default to TLS
		raw = "https://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, &ce.CustomError{Title: "Error parsing the registry URL", Message: err.Error()}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, &ce.CustomError{Title: "Invalid URL scheme",
			Message: u.Scheme + " is not a supported (https/http) scheme"}
	}
	if u.Host == "" {
		return nil, &ce.CustomError{Title: "Invalid registry url %q", Message: raw + " is not a valid URL"}
	}

	// Normalize: ensure no trailing path; registry base should be scheme://host[:port]
	u.Path = ""
	u.RawQuery = ""
	u.Fragment = ""

	return u, nil
}

func readAll(r io.Reader) ([]byte, *ce.CustomError) {
	bytes, err := io.ReadAll(io.LimitReader(r, 50<<20)) // 50MB safety cap
	if err != nil {
		return nil, &ce.CustomError{Title: "Error reading response", Message: err.Error()}
	} else {
		return bytes, nil
	}
}

func httpError(resp *http.Response, path string) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10)) // 8KB snippet
	msg := strings.TrimSpace(string(b))
	if msg == "" {
		return fmt.Errorf("%s %s failed: %s", resp.Request.Method, path, resp.Status)
	}
	return fmt.Errorf("%s %s failed: %s: %s", resp.Request.Method, path, resp.Status, msg)
}

func basicAuthHeader(user, pass string) string {
	// http.Request.SetBasicAuth uses base64; we want header string here
	req, _ := http.NewRequest(http.MethodGet, "http://x", nil)
	req.SetBasicAuth(user, pass)
	return req.Header.Get("Authorization")
}
