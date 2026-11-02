// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/auth/auth_test.go

package auth

import (
	"context"
	"dtools2/rest"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// isolate points DOCKER_CONFIG at a temp dir and returns the config.json path
func isolate(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "docker")
	t.Setenv("DOCKER_CONFIG", dir)
	ctx := rest.Context
	rest.Context = context.Background()
	t.Cleanup(func() { rest.Context = ctx })
	return filepath.Join(dir, "config.json")
}

func writeConfig(t *testing.T, path, content string) {
	t.Helper()
	os.MkdirAll(filepath.Dir(path), 0o700)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadDockerConfig(t *testing.T) {
	path := isolate(t)

	cfg, p, err := LoadDockerConfig()
	if err != nil || p != path || cfg.Auths == nil || len(cfg.Auths) != 0 {
		t.Errorf("missing file: %+v, %q, %v; want an empty config at %s", cfg, p, err, path)
	}

	writeConfig(t, path, "")
	if cfg, _, err := LoadDockerConfig(); err != nil || cfg.Auths == nil {
		t.Errorf("empty file: %+v, %v", cfg, err)
	}

	writeConfig(t, path, `{"credsStore":"pass"}`)
	if cfg, _, err := LoadDockerConfig(); err != nil || cfg.Auths == nil {
		t.Errorf("no auths key: %+v, %v; want an initialized map", cfg, err)
	}

	writeConfig(t, path, `{oops`)
	if _, _, err := LoadDockerConfig(); err == nil {
		t.Error("invalid JSON: want an error")
	}
}

func TestLoadDockerConfigDefaultsToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DOCKER_CONFIG", "")
	t.Setenv("HOME", home)
	if _, p, err := LoadDockerConfig(); err != nil || p != filepath.Join(home, ".docker", "config.json") {
		t.Errorf("path = %q, %v; want ~/.docker/config.json", p, err)
	}
}

func TestUpdateAuth(t *testing.T) {
	path := isolate(t)
	writeConfig(t, path, `{"auths":{"reg:5000":{"username":"old","password":"cleartext","email":"a@b.c"}}}`)

	if err := UpdateAuth("reg:5000", "user", "pass"); err != nil {
		t.Fatalf("UpdateAuth: %v", err)
	}
	cfg, _, _ := LoadDockerConfig()
	e := cfg.Auths["reg:5000"]
	if e.Auth != base64.StdEncoding.EncodeToString([]byte("user:pass")) {
		t.Errorf("auth = %q; want base64(user:pass)", e.Auth)
	}
	if e.Username != "" || e.Password != "" {
		t.Error("cleartext credentials must be scrubbed")
	}
	if e.Email != "a@b.c" {
		t.Error("non-credential fields must be preserved")
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o600 {
		t.Errorf("config mode = %v; want 0600", fi.Mode().Perm())
	}

	// Same credentials: no rewrite
	before, _ := os.Stat(path)
	time.Sleep(10 * time.Millisecond)
	if err := UpdateAuth("reg:5000", "user", "pass"); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.Stat(path); !after.ModTime().Equal(before.ModTime()) {
		t.Error("unchanged credentials must not rewrite the file")
	}
}

// `dtools login` must not wipe the rest of the user's docker configuration
func TestUpdateAuthPreservesOtherSettings(t *testing.T) {
	path := isolate(t)
	writeConfig(t, path, `{"auths":{},"credsStore":"desktop","currentContext":"remote","proxies":{"default":{"httpProxy":"http://proxy:3128"}}}`)

	if err := UpdateAuth("reg:5000", "user", "pass"); err != nil {
		t.Fatalf("UpdateAuth: %v", err)
	}
	b, _ := os.ReadFile(path)
	var raw map[string]any
	json.Unmarshal(b, &raw)
	for _, key := range []string{"credsStore", "currentContext", "proxies"} {
		if _, ok := raw[key]; !ok {
			t.Errorf("config.json lost its %q setting after UpdateAuth: %s", key, b)
		}
	}
}

func decodeHeader(t *testing.T, h string) map[string]string {
	t.Helper()
	b, err := base64.URLEncoding.DecodeString(h)
	if err != nil {
		t.Fatalf("header is not URL-safe base64: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("header is not base64(JSON): %v", err)
	}
	return m
}

func TestBuildRegistryAuthHeader(t *testing.T) {
	path := isolate(t)
	writeConfig(t, path, `{"auths":{
		"https://reg.example.com":{"auth":"`+base64.StdEncoding.EncodeToString([]byte("alice:s3:cret"))+`"},
		"plain:5000":{"username":"bob","password":"pw"},
		"token.example.com":{"identitytoken":"tok123"},
		"broken.example.com":{"auth":"!!!notbase64"}
	}}`)

	h, err := BuildRegistryAuthHeader("reg.example.com")
	if err != nil {
		t.Fatalf("BuildRegistryAuthHeader: %v", err)
	}
	// Passwords may contain ':'; only the first one separates the username
	if m := decodeHeader(t, h); m["username"] != "alice" || m["password"] != "s3:cret" || m["serveraddress"] != "https://reg.example.com" {
		t.Errorf("payload = %v", m)
	}

	h, _ = BuildRegistryAuthHeader("plain:5000")
	if m := decodeHeader(t, h); m["username"] != "bob" || m["password"] != "pw" {
		t.Errorf("cleartext entry payload = %v", m)
	}

	h, err = BuildRegistryAuthHeader("token.example.com")
	if err != nil {
		t.Fatalf("identity token entry: %v", err)
	}
	if m := decodeHeader(t, h); m["identitytoken"] != "tok123" {
		t.Errorf("token payload = %v", m)
	}

	if _, err := BuildRegistryAuthHeader("broken.example.com"); err == nil {
		t.Error("an undecodable entry: want an error")
	}
	// "No login" must be distinguishable, so that pulls can fall back to anonymous
	if _, err := BuildRegistryAuthHeader("unknown.example.com"); !errors.Is(err, ErrNoCredentials) {
		t.Errorf("no entry = %v; want ErrNoCredentials", err)
	}
}

func TestNormalizeRegistryURL(t *testing.T) {
	cases := map[string]string{
		"reg:5000":                 "https://reg:5000/",
		"  http://insecure:5000  ": "http://insecure:5000/",
		"https://reg.example.com":  "https://reg.example.com/",
	}
	for in, want := range cases {
		u, err := normalizeRegistryURL(in)
		if err != nil || u.String() != want {
			t.Errorf("normalizeRegistryURL(%q) = %v, %v; want %s", in, u, err, want)
		}
	}
	for _, bad := range []string{"", "   ", "https://"} {
		if _, err := normalizeRegistryURL(bad); err == nil {
			t.Errorf("normalizeRegistryURL(%q): want an error", bad)
		}
	}
}

func TestRegistryConfigKey(t *testing.T) {
	cases := map[string]string{
		"https://reg.example.com:5000/v2/": "reg.example.com:5000",
		"http://insecure:5000":             "insecure:5000",
		" reg:5000 ":                       "reg:5000",
		"":                                 "",
	}
	for in, want := range cases {
		if got := registryConfigKey(in); got != want {
			t.Errorf("registryConfigKey(%q) = %q; want %q", in, got, want)
		}
	}
}

// fakeRegistry answers /v2/ with 200 for user:pass, 401 otherwise
func fakeRegistry(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v2/" {
		http.NotFound(w, r)
		return
	}
	if u, p, ok := r.BasicAuth(); ok && u == "user" && p == "pass" {
		w.Write([]byte("{}"))
		return
	}
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"errors":[{"code":"UNAUTHORIZED"}]}`))
}

func TestLogin(t *testing.T) {
	path := isolate(t)
	srv := httptest.NewServer(http.HandlerFunc(fakeRegistry))
	defer srv.Close()

	if err := Login(LoginOptions{Registry: srv.URL, Username: "user", Password: "pass"}); err != nil {
		t.Fatalf("Login: %v", err)
	}
	cfg, _, _ := LoadDockerConfig()
	key := strings.TrimPrefix(srv.URL, "http://")
	if _, ok := cfg.Auths[key]; !ok {
		t.Errorf("auths = %v; want an entry keyed %q (host:port, no scheme)", cfg.Auths, key)
	}

	err := Login(LoginOptions{Registry: srv.URL, Username: "user", Password: "wrong"})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("Login with a bad password = %v; want a 401 error", err)
	}

	os.Remove(path)
	_ = Login(LoginOptions{Registry: srv.URL, Username: "user", Password: "wrong"})
	if _, serr := os.Stat(path); serr == nil {
		t.Error("a failed login must not write config.json")
	}
}

func TestLoginValidation(t *testing.T) {
	isolate(t)
	cases := []LoginOptions{
		{Username: "u", Password: "p"},
		{Registry: "r", Password: "p"},
		{Registry: "r", Username: "u"},
		{Registry: "https://", Username: "u", Password: "p"},
		{Registry: "r", Username: "u", Password: "p", CACertPath: "/does/not/exist.pem"},
	}
	for _, o := range cases {
		if err := Login(o); err == nil {
			t.Errorf("Login(%+v): want an error", o)
		}
	}
}

func TestLoginTLSWithCustomCA(t *testing.T) {
	isolate(t)
	srv := httptest.NewTLSServer(http.HandlerFunc(fakeRegistry))
	defer srv.Close()
	host := strings.TrimPrefix(srv.URL, "https://")

	// Without the CA, verification fails
	if err := Login(LoginOptions{Registry: host, Username: "user", Password: "pass"}); err == nil {
		t.Error("Login to a self-signed registry without its CA: want a TLS error")
	}

	ca := filepath.Join(t.TempDir(), "ca.pem")
	os.WriteFile(ca, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: srv.Certificate().Raw}), 0o600)
	if err := Login(LoginOptions{Registry: host, Username: "user", Password: "pass", CACertPath: ca}); err != nil {
		t.Errorf("Login with the registry CA: %v", err)
	}
	if err := Login(LoginOptions{Registry: host, Username: "user", Password: "pass", Insecure: true}); err != nil {
		t.Errorf("Login with Insecure: %v", err)
	}

	garbage := filepath.Join(t.TempDir(), "garbage.pem")
	os.WriteFile(garbage, []byte("nope"), 0o600)
	if err := Login(LoginOptions{Registry: host, Username: "user", Password: "pass", CACertPath: garbage}); err == nil {
		t.Error("Login with an unparseable CA: want an error")
	}
}

func TestHashSecret(t *testing.T) {
	a, b := hashSecret("secret"), hashSecret("secret")
	if a != b || a == hashSecret("other") || a == "secret" {
		t.Errorf("hashSecret must be deterministic, input-dependent and not the input itself")
	}
}

func TestGetUser(t *testing.T) {
	if u, err := GetUser(); err != nil || u == "" {
		t.Errorf("GetUser() = %q, %v", u, err)
	}
}
