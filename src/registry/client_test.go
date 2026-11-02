// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/registry/client_test.go

package registry

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// isolateHome keeps NewClient from reading the user's real ~/.docker/config.json
func isolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DOCKER_CONFIG", "")
	return home
}

func newClient(t *testing.T, raw string, opts ...Option) *Client {
	t.Helper()
	c, err := NewClient(raw, opts...)
	if err != nil {
		t.Fatalf("NewClient(%q): %v", raw, err)
	}
	return c
}

func TestParseRegistryURL(t *testing.T) {
	cases := map[string]string{
		"nexus:9820":                     "https://nexus:9820",
		"http://insecure:5000/some/path": "http://insecure:5000",
		"  https://reg.example.com/?x=1": "https://reg.example.com",
	}
	for in, want := range cases {
		u, err := parseRegistryURL(in)
		if err != nil || u.String() != want {
			t.Errorf("parseRegistryURL(%q) = %v, %v; want %s", in, u, err, want)
		}
	}
	for _, bad := range []string{"", "https://", "http://[::1"} {
		if _, err := parseRegistryURL(bad); err == nil {
			t.Errorf("parseRegistryURL(%q): want an error", bad)
		}
	}
}

func TestCatalogAnonymous(t *testing.T) {
	isolateHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/_catalog" || r.URL.Query().Get("n") != "50" || r.Header.Get("Authorization") != "" {
			t.Errorf("unexpected request %s (auth %q)", r.URL, r.Header.Get("Authorization"))
		}
		io.WriteString(w, `{"repositories":["a","b"]}`)
	}))
	defer srv.Close()

	b, err := newClient(t, srv.URL).CatalogJSON(context.Background(), url.Values{"n": {"50"}})
	if err != nil || string(b) != `{"repositories":["a","b"]}` {
		t.Errorf("CatalogJSON = %s, %v", b, err)
	}
}

// bearerRegistry is a registry that requires a token, plus its token server
type bearerRegistry struct {
	registry, tokens *httptest.Server
	tokenCalls       atomic.Int32
	lastTokenQuery   atomic.Value
}

func newBearerRegistry(t *testing.T, expiresIn int) *bearerRegistry {
	t.Helper()
	b := &bearerRegistry{}
	b.tokens = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b.tokenCalls.Add(1)
		b.lastTokenQuery.Store(r.URL.Query())
		if u, p, ok := r.BasicAuth(); !ok || u != "alice" || p != "pw" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		io.WriteString(w, `{"access_token":"tok-1","expires_in":`+strconv.Itoa(expiresIn)+`}`)
	}))
	b.registry = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok-1" {
			w.Header().Set("WWW-Authenticate", `Bearer realm="`+b.tokens.URL+`/token",service="reg",scope="repository:team/app:pull"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		io.WriteString(w, `{"name":"team/app","tags":["v1","v2"]}`)
	}))
	t.Cleanup(b.tokens.Close)
	t.Cleanup(b.registry.Close)
	return b
}

func TestTagsBearerFlowAndCache(t *testing.T) {
	isolateHome(t)
	b := newBearerRegistry(t, 300)
	c := newClient(t, b.registry.URL, WithCredentials("alice", "pw"))

	for i := 0; i < 2; i++ {
		got, err := c.TagsJSON(context.Background(), "/team/app", nil)
		if err != nil || !strings.Contains(string(got), `"v2"`) {
			t.Fatalf("TagsJSON #%d = %s, %v", i, got, err)
		}
	}
	if n := b.tokenCalls.Load(); n != 1 {
		t.Errorf("token server called %d times; want 1 (the token must be cached)", n)
	}
	q := b.lastTokenQuery.Load().(url.Values)
	if q.Get("service") != "reg" || q.Get("scope") != "repository:team/app:pull" || q.Get("account") != "alice" {
		t.Errorf("token query = %v; want service, scope and account", q)
	}
}

func TestBearerFlowBadCredentials(t *testing.T) {
	isolateHome(t)
	b := newBearerRegistry(t, 300)
	c := newClient(t, b.registry.URL, WithCredentials("alice", "wrong"))
	if _, err := c.TagsJSON(context.Background(), "team/app", nil); err == nil {
		t.Error("TagsJSON with rejected credentials: want an error")
	}
}

func TestCachedTokenExpiry(t *testing.T) {
	c := &Client{tokenCache: map[string]cachedToken{}}
	c.setCachedToken("k", "old", time.Now().Add(-time.Second))
	if _, ok := c.getCachedToken("k"); ok {
		t.Error("an expired token must not be returned")
	}
	c.setCachedToken("k", "fresh", time.Now().Add(time.Minute))
	if tok, ok := c.getCachedToken("k"); !ok || tok != "fresh" {
		t.Errorf("getCachedToken = %q, %v", tok, ok)
	}
}

func TestFetchBearerTokenErrors(t *testing.T) {
	isolateHome(t)
	c := newClient(t, "https://reg.example.com")
	if _, _, err := c.fetchBearerToken(context.Background(), bearerChallenge{Realm: "not a url"}); err == nil {
		t.Error("an invalid realm: want an error")
	}

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, `{}`) }))
	defer empty.Close()
	if _, _, err := c.fetchBearerToken(context.Background(), bearerChallenge{Realm: empty.URL}); err == nil {
		t.Error("an empty token: want an error")
	}

	short := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, `{"token":"t"}`) }))
	defer short.Close()
	tok, exp, err := c.fetchBearerToken(context.Background(), bearerChallenge{Realm: short.URL})
	if err != nil || tok != "t" || time.Until(exp) > 61*time.Second {
		t.Errorf("token without expires_in = %q, %v, %v; want a ~60s default expiry", tok, time.Until(exp), err)
	}
}

func TestBasicChallenge(t *testing.T) {
	isolateHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); ok && u == "bob" && p == "pw" {
			io.WriteString(w, `{"repositories":[]}`)
			return
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="nexus"`)
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	if _, err := newClient(t, srv.URL, WithCredentials("bob", "pw")).CatalogJSON(context.Background(), nil); err != nil {
		t.Errorf("CatalogJSON with Basic credentials: %v", err)
	}
	if _, err := newClient(t, srv.URL).CatalogJSON(context.Background(), nil); err == nil {
		t.Error("CatalogJSON on a Basic challenge without credentials: want an error")
	}
	if _, err := newClient(t, srv.URL, WithCredentials("bob", "bad")).CatalogJSON(context.Background(), nil); err == nil {
		t.Error("CatalogJSON with wrong Basic credentials: want an error")
	}
}

func TestTagsEmptyRepo(t *testing.T) {
	isolateHome(t)
	if _, err := newClient(t, "reg:5000").TagsJSON(context.Background(), "/", nil); err == nil {
		t.Error("TagsJSON with an empty repo: want an error")
	}
}

func TestServerError(t *testing.T) {
	isolateHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) }))
	defer srv.Close()
	if _, err := newClient(t, srv.URL).CatalogJSON(context.Background(), nil); err == nil || !strings.Contains(err.Message, "500") {
		t.Errorf("CatalogJSON on HTTP 500 = %v; want the status", err)
	}
}

func TestWithInsecureSkipVerify(t *testing.T) {
	isolateHome(t)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, `{}`) }))
	defer srv.Close()
	srv.Config.ErrorLog = nil

	if _, err := newClient(t, srv.URL).CatalogJSON(context.Background(), nil); err == nil {
		t.Error("a self-signed registry without InsecureSkipVerify: want a TLS error")
	}
	if _, err := newClient(t, srv.URL, WithInsecureSkipVerify(true)).CatalogJSON(context.Background(), nil); err != nil {
		t.Errorf("with InsecureSkipVerify: %v", err)
	}
	if _, err := newClient(t, srv.URL, WithHTTPClient(srv.Client())).CatalogJSON(context.Background(), nil); err != nil {
		t.Errorf("WithHTTPClient(trusting client): %v", err)
	}
	// nil options are ignored
	newClient(t, srv.URL, nil, WithHTTPClient(nil), WithCredentials("", ""))
}

func writeDockerConfig(t *testing.T, path string) {
	t.Helper()
	auth := base64.StdEncoding.EncodeToString([]byte("carol:pw"))
	os.MkdirAll(filepath.Dir(path), 0o700)
	os.WriteFile(path, []byte(`{"auths":{"https://nexus:9820":{"auth":"`+auth+`"},"https://index.docker.io/v1/":{"auth":"`+auth+`"},"portless.example.com":{"auth":"`+auth+`"}}}`), 0o600)
}

func TestDockerConfigCredentials(t *testing.T) {
	home := isolateHome(t)
	writeDockerConfig(t, filepath.Join(home, ".docker", "config.json"))

	c := newClient(t, "nexus:9820")
	// portless.example.com:443 falls back to the credentials stored for the bare host
	for _, host := range []string{"nexus:9820", "registry-1.docker.io", "docker.io", "portless.example.com:443"} {
		if u, _, ok := c.creds(host); !ok || u != "carol" {
			t.Errorf("creds(%q) = %q, %v; want carol", host, u, ok)
		}
	}
	if _, _, ok := c.creds("other:5000"); ok {
		t.Error("creds for an unknown host: want none")
	}
	if _, _, ok := c.creds("nexus:1234"); ok {
		t.Error("credentials stored for nexus:9820 must not apply to another port")
	}
	if _, _, ok := c.creds("  "); ok {
		t.Error("creds for an empty host: want none")
	}
}

// DOCKER_CONFIG relocates config.json, as honored by the auth package and the docker CLI
func TestDockerConfigEnvHonored(t *testing.T) {
	isolateHome(t)
	dir := t.TempDir()
	t.Setenv("DOCKER_CONFIG", dir)
	writeDockerConfig(t, filepath.Join(dir, "config.json"))

	c := newClient(t, "nexus:9820")
	if c.creds == nil {
		t.Fatal("no credentials loaded: $DOCKER_CONFIG/config.json was ignored")
	}
	if u, _, ok := c.creds("nexus:9820"); !ok || u != "carol" {
		t.Errorf("creds(nexus:9820) = %q, %v; want carol from $DOCKER_CONFIG", u, ok)
	}
}

func TestDecodeDockerAuth(t *testing.T) {
	enc := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	cases := []struct {
		in         string
		user, pass string
		ok         bool
	}{
		{enc("u:p"), "u", "p", true},
		{enc("u:p:with:colons"), "u", "p:with:colons", true},
		{enc("nocolon"), "", "", false},
		{"", "", "", false},
		{"!!!", "", "", false},
	}
	for _, c := range cases {
		u, p, ok := decodeDockerAuth(c.in)
		if u != c.user || p != c.pass || ok != c.ok {
			t.Errorf("decodeDockerAuth(%q) = %q, %q, %v", c.in, u, p, ok)
		}
	}
}

func TestExtractHostFromDockerAuthKey(t *testing.T) {
	cases := map[string]string{
		"https://index.docker.io/v1/": "index.docker.io",
		"https://nexus:9820":          "nexus:9820",
		"nexus:9820":                  "nexus:9820",
		"nexus:9820/path":             "nexus:9820",
		"  ":                          "",
	}
	for in, want := range cases {
		if got := extractHostFromDockerAuthKey(in); got != want {
			t.Errorf("extractHostFromDockerAuthKey(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestHTTPError(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://x/v2/", nil)
	resp := &http.Response{Status: "404 Not Found", Request: req, Body: io.NopCloser(strings.NewReader(" missing "))}
	if got := httpError(resp, "/v2/").Error(); got != "GET /v2/ failed: 404 Not Found: missing" {
		t.Errorf("httpError = %q", got)
	}
	resp.Body = io.NopCloser(strings.NewReader(""))
	if got := httpError(resp, "/v2/").Error(); got != "GET /v2/ failed: 404 Not Found" {
		t.Errorf("httpError (no body) = %q", got)
	}
}

func TestBasicAuthHeader(t *testing.T) {
	if got := basicAuthHeader("u", "p"); got != "Basic "+base64.StdEncoding.EncodeToString([]byte("u:p")) {
		t.Errorf("basicAuthHeader = %q", got)
	}
}
