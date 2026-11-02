// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/rest/client_test.go

package rest

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, cfg Config) *Client {
	t.Helper()
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient(%+v): %v", cfg, err)
	}
	return c
}

func TestNewClientHosts(t *testing.T) {
	cases := []struct {
		name       string
		host       string
		tls        bool
		wantScheme string
		wantHost   string
		wantUnix   string
	}{
		{"bare host:port is tcp", "vps:2475", false, "http", "vps:2475", ""},
		{"tcp without TLS", "tcp://10.0.0.5:2375", false, "http", "10.0.0.5:2375", ""},
		{"tcp with TLS", "tcp://10.0.0.5:2376", true, "https", "10.0.0.5:2376", ""},
		{"explicit https", "https://host:2376", false, "https", "host:2376", ""},
		{"explicit http", "http://host:2375", true, "http", "host:2375", ""},
		{"unix socket", "unix:///run/podman/podman.sock", false, "http", "d", "/run/podman/podman.sock"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cl := newTestClient(t, Config{Host: c.host, UseTLS: c.tls})
			if cl.baseURL.Scheme != c.wantScheme || cl.baseURL.Host != c.wantHost {
				t.Errorf("baseURL = %s; want %s://%s", cl.baseURL, c.wantScheme, c.wantHost)
			}
			if cl.SocketPath() != c.wantUnix {
				t.Errorf("SocketPath() = %q; want %q", cl.SocketPath(), c.wantUnix)
			}
		})
	}
}

func TestNewClientDefaultHost(t *testing.T) {
	t.Setenv("DOCKER_HOST", "tcp://from-env:2375")
	if cl := newTestClient(t, Config{}); cl.baseURL.Host != "from-env:2375" {
		t.Errorf("host = %q; want DOCKER_HOST", cl.baseURL.Host)
	}

	t.Setenv("DOCKER_HOST", "")
	if cl := newTestClient(t, Config{}); cl.SocketPath() != "/var/run/docker.sock" {
		t.Errorf("SocketPath() = %q; want the docker default socket", cl.SocketPath())
	}
}

func TestNewClientErrors(t *testing.T) {
	for _, host := range []string{"unix://", "ftp://host:21", "tcp://"} {
		if _, err := NewClient(Config{Host: host}); err == nil {
			t.Errorf("NewClient(%q): want an error", host)
		}
	}
	bad := Config{Host: "tcp://h:2376", UseTLS: true, CACertPath: filepath.Join(t.TempDir(), "nope.pem")}
	if _, err := NewClient(bad); err == nil || !strings.Contains(err.Error(), "TLS") {
		t.Errorf("NewClient with a missing CA = %v; want a TLS config error", err)
	}
}

func TestNewClientTimeouts(t *testing.T) {
	saved := FastFailTimeoutSeconds
	t.Cleanup(func() { FastFailTimeoutSeconds = saved })

	FastFailTimeoutSeconds = 7
	cl := newTestClient(t, Config{Host: "tcp://h:1", SessionTimeout: -5 * time.Second})
	if cl.fastFailTimeout != 7*time.Second {
		t.Errorf("fastFailTimeout = %v; want the global 7s", cl.fastFailTimeout)
	}
	if cl.sessionTimeout != 0 {
		t.Errorf("sessionTimeout = %v; a negative value must disable it", cl.sessionTimeout)
	}

	FastFailTimeoutSeconds = 0
	if cl := newTestClient(t, Config{Host: "tcp://h:1"}); cl.fastFailTimeout != 30*time.Second {
		t.Errorf("fastFailTimeout = %v; want the 30s fallback", cl.fastFailTimeout)
	}
	if cl := newTestClient(t, Config{Host: "tcp://h:1", FastFailTimeout: time.Second}); cl.fastFailTimeout != time.Second {
		t.Errorf("fastFailTimeout = %v; an explicit value must win", cl.fastFailTimeout)
	}
}

// echoServer replies with the request's method, path, query and a header
func echoServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery+" "+r.Header.Get("X-Test"))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func doString(t *testing.T, c *Client, method, path string, q url.Values, h http.Header) string {
	t.Helper()
	resp, err := c.Do(context.Background(), method, path, q, nil, h)
	if err != nil {
		t.Fatalf("Do(%s %s): %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return string(b)
}

func TestDoVersionPrefix(t *testing.T) {
	srv := echoServer(t)
	c := newTestClient(t, Config{Host: srv.URL, APIVersion: " 1.45 "})

	h := http.Header{}
	h.Set("X-Test", "yes")
	if got := doString(t, c, http.MethodGet, "containers/json", url.Values{"all": {"true"}}, h); got != "GET /v1.45/containers/json?all=true yes" {
		t.Errorf("versioned call = %q", got)
	}
	// /version is always unversioned, for negotiation
	if got := doString(t, c, http.MethodGet, "/version", nil, nil); got != "GET /version? " {
		t.Errorf("/version call = %q", got)
	}

	c.SetAPIVersion("")
	if got := doString(t, c, http.MethodGet, "/info", nil, nil); got != "GET /info? " {
		t.Errorf("unversioned call = %q", got)
	}
	if c.APIVersion() != "" {
		t.Errorf("APIVersion() = %q; want empty", c.APIVersion())
	}
}

func TestDoSessionTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, Config{Host: srv.URL, SessionTimeout: 100 * time.Millisecond})

	// A finite operation gets the session timeout: reading the body must fail
	resp, err := c.Do(context.Background(), http.MethodGet, "/images/json", nil, nil, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	start := time.Now()
	_, rerr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if rerr == nil || time.Since(start) > time.Second {
		t.Errorf("read error = %v after %v; want a timeout around 100ms", rerr, time.Since(start))
	}

	// A caller deadline takes precedence and is not replaced
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	resp, err = c.Do(ctx, http.MethodGet, "/images/json", nil, nil, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	start = time.Now()
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if d := time.Since(start); d < 200*time.Millisecond {
		t.Errorf("stream ended after %v; the caller's 300ms deadline should apply, not the session timeout", d)
	}
}

func TestIsNoTimeoutEndpoint(t *testing.T) {
	c := &Client{}
	cases := []struct {
		path  string
		query url.Values
		want  bool
	}{
		{"/events", nil, true},
		{"/containers/x/logs", url.Values{"follow": {"1"}}, true},
		{"/containers/x/logs", url.Values{"follow": {"true"}}, true},
		{"/containers/x/logs", url.Values{"follow": {"false"}}, false},
		{"/containers/x/logs", nil, false},
		{"/containers/x/stats", nil, true},
		{"/containers/x/stats", url.Values{"stream": {"false"}}, false},
		{"/containers/x/stats", url.Values{}, true},
		{"/containers/x/wait", nil, true},
		{"/images/create", nil, false},
	}
	for _, cs := range cases {
		if got := c.isNoTimeoutEndpoint(cs.path, cs.query); got != cs.want {
			t.Errorf("isNoTimeoutEndpoint(%s, %v) = %v; want %v", cs.path, cs.query, got, cs.want)
		}
	}
}

func TestDoUnixSocket(t *testing.T) {
	// Unix socket paths are length-limited; t.TempDir() can be too long, so use a short dir
	dir, err := os.MkdirTemp("", "dt")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "d.sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Skipf("unix sockets unavailable: %v", err)
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, r.URL.Path)
	})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })

	c := newTestClient(t, Config{Host: "unix://" + sock, APIVersion: "1.50"})
	if got := doString(t, c, http.MethodGet, "/_ping", nil, nil); got != "/v1.50/_ping" {
		t.Errorf("unix call path = %q", got)
	}
}

func TestNegotiateAPIVersion(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   string
		err    bool
	}{
		{"ok", http.StatusOK, `{"ApiVersion":"1.51","Version":"28.0"}`, "1.51", false},
		{"http error", http.StatusInternalServerError, `{}`, "", true},
		{"bad json", http.StatusOK, `{`, "", true},
		{"no version", http.StatusOK, `{"Version":"28.0"}`, "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/version" {
					t.Errorf("path = %s; want unversioned /version", r.URL.Path)
				}
				w.WriteHeader(c.status)
				io.WriteString(w, c.body)
			}))
			defer srv.Close()
			got, err := NegotiateAPIVersion(context.Background(), newTestClient(t, Config{Host: srv.URL, APIVersion: "1.40"}))
			if (err != nil) != c.err || got != c.want {
				t.Errorf("NegotiateAPIVersion = %q, %v; want %q, error=%v", got, err, c.want, c.err)
			}
		})
	}
}

// rawServer accepts one connection, hands the parsed request to check, then writes reply verbatim
func rawServer(t *testing.T, reply string, check func(*http.Request)) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		req, err := http.ReadRequest(bufio.NewReader(conn))
		if err == nil && check != nil {
			check(req)
		}
		io.WriteString(conn, reply)
	}()
	return "tcp://" + ln.Addr().String()
}

func TestHijackUpgrade(t *testing.T) {
	var got *http.Request
	host := rawServer(t, "HTTP/1.1 101 UPGRADED\r\nContent-Type: application/vnd.docker.raw-stream\r\n\r\nstream-data", func(r *http.Request) { got = r })
	c := newTestClient(t, Config{Host: host, APIVersion: "1.50"})

	hc, err := c.Hijack(context.Background(), http.MethodPost, "/exec/abc/start", url.Values{"x": {"1"}}, nil, []byte(`{"Tty":true}`), true)
	if err != nil {
		t.Fatalf("Hijack: %v", err)
	}
	defer hc.Conn.Close()

	if hc.Code != http.StatusSwitchingProtocols || hc.Header.Get("Content-Type") != "application/vnd.docker.raw-stream" {
		t.Errorf("code %d, headers %v", hc.Code, hc.Header)
	}
	data, _ := io.ReadAll(hc.Reader)
	if string(data) != "stream-data" {
		t.Errorf("stream = %q; want the bytes following the headers", data)
	}
	if got == nil {
		t.Fatal("server did not receive a parseable request")
	}
	if got.URL.Path != "/v1.50/exec/abc/start" || got.URL.RawQuery != "x=1" {
		t.Errorf("request URL = %s", got.URL)
	}
	if got.Header.Get("Upgrade") != "tcp" || got.Header.Get("Connection") != "Upgrade" || got.Header.Get("Content-Type") != "application/json" {
		t.Errorf("request headers = %v; want upgrade headers and a JSON content type", got.Header)
	}
}

func TestHijackErrors(t *testing.T) {
	cases := []struct {
		name, reply, want string
	}{
		{"http error with body", "HTTP/1.1 404 Not Found\r\nContent-Length: 18\r\n\r\nno such exec found", "returned HTTP 404: no such exec found"},
		{"http error without body", "HTTP/1.1 500 Internal Server Error\r\nContent-Length: 0\r\n\r\n", "returned HTTP 500"},
		{"garbage status line", "garbage\r\n\r\n", "invalid HTTP response status line"},
		{"non-numeric status", "HTTP/1.1 abc OK\r\n\r\n", "invalid HTTP status code"},
	}
	for _, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			c := newTestClient(t, Config{Host: rawServer(t, cs.reply, nil)})
			_, err := c.Hijack(context.Background(), http.MethodPost, "/exec/abc/start", nil, nil, nil, false)
			if err == nil || !strings.Contains(err.Error(), cs.want) {
				t.Errorf("Hijack error = %v; want it to contain %q", err, cs.want)
			}
		})
	}
}

func TestHijackPlainSuccess(t *testing.T) {
	c := newTestClient(t, Config{Host: rawServer(t, "HTTP/1.1 200 OK\r\n\r\n", nil)})
	hc, err := c.Hijack(context.Background(), http.MethodPost, "containers/x/attach", nil, nil, nil, false)
	if err != nil {
		t.Fatalf("Hijack: %v", err)
	}
	hc.Conn.Close()
}

func TestHijackDialFailure(t *testing.T) {
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	addr := ln.Addr().String()
	ln.Close() // nothing listens there any more
	c := newTestClient(t, Config{Host: "tcp://" + addr, FastFailTimeout: time.Second})
	if _, err := c.Hijack(context.Background(), http.MethodPost, "/x", nil, nil, nil, true); err == nil {
		t.Error("Hijack to a closed port: want an error")
	}
}

func TestEffectiveFastFail(t *testing.T) {
	c := &Client{fastFailTimeout: 10 * time.Second}
	if got := c.effectiveFastFail(nil); got != 10*time.Second {
		t.Errorf("nil ctx: %v", got)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if got := c.effectiveFastFail(ctx); got > 2*time.Second || got < time.Second {
		t.Errorf("ctx with a shorter deadline: %v; want ~2s", got)
	}
	if got := (&Client{}).effectiveFastFail(context.Background()); got != 30*time.Second {
		t.Errorf("unset fast-fail: %v; want 30s", got)
	}
}

func TestStripPort(t *testing.T) {
	cases := map[string]string{"host:2376": "host", "host": "host", "[::1]:2376": "::1", "10.0.0.5:22": "10.0.0.5"}
	for in, want := range cases {
		if got := stripPort(in); got != want {
			t.Errorf("stripPort(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestTLSConfigForDial(t *testing.T) {
	c := newTestClient(t, Config{Host: "https://docker.example:2376"})
	cfg, err := c.tlsConfigForDial("docker.example:2376")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ServerName != "docker.example" {
		t.Errorf("ServerName = %q; want the host without its port", cfg.ServerName)
	}
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("DOCKER_HOST", "")
	if got := ConfigFromEnv().Host; got != "unix:///var/run/docker.sock" {
		t.Errorf("default host = %q", got)
	}
	t.Setenv("DOCKER_HOST", "tcp://x:2375")
	if got := ConfigFromEnv().Host; got != "tcp://x:2375" {
		t.Errorf("host from env = %q", got)
	}
}

func TestDumpURL(t *testing.T) {
	c := newTestClient(t, Config{Host: "tcp://h:2375"})
	if got := c.DumpURL("/info"); got != "http://h:2375/info" {
		t.Errorf("DumpURL = %q", got)
	}
}

// writeCertPair writes a self-signed certificate and its key as PEM files
func writeCertPair(t *testing.T, dir, name string, isCA bool) (certPath, keyPath string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  isCA,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	certPath, keyPath = filepath.Join(dir, name+".pem"), filepath.Join(dir, name+"-key.pem")
	os.WriteFile(certPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600)
	os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600)
	return certPath, keyPath
}

func TestBuildTLSConfig(t *testing.T) {
	dir := t.TempDir()
	ca, _ := writeCertPair(t, dir, "ca", true)
	cert, key := writeCertPair(t, dir, "client", false)

	cfg, err := buildTLSConfig(Config{UseTLS: true, CACertPath: ca, CertPath: cert, KeyPath: key, InsecureSkipVerify: true})
	if err != nil {
		t.Fatalf("buildTLSConfig: %v", err)
	}
	if cfg.RootCAs == nil || len(cfg.Certificates) != 1 || !cfg.InsecureSkipVerify {
		t.Errorf("config = %+v; want the CA pool, one client certificate and InsecureSkipVerify", cfg)
	}

	// No CA given: the system pool is used
	if cfg, err := buildTLSConfig(Config{UseTLS: true}); err != nil || len(cfg.Certificates) != 0 {
		t.Errorf("buildTLSConfig without CA/cert = %+v, %v", cfg, err)
	}

	garbage := filepath.Join(dir, "garbage.pem")
	os.WriteFile(garbage, []byte("not a certificate"), 0o600)
	if _, err := buildTLSConfig(Config{UseTLS: true, CACertPath: garbage}); err == nil {
		t.Error("an unparseable CA: want an error")
	}
	if _, err := buildTLSConfig(Config{UseTLS: true, CertPath: cert, KeyPath: garbage}); err == nil {
		t.Error("an unparseable client key: want an error")
	}
}
