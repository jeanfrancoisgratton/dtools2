// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/11/02
// Original filename: src/internal/testutil/fakedaemon.go

// Package testutil provides test helpers shared across packages: a fake docker/podman daemon
// the real rest.Client can talk to, and stdout capture. It is only imported by _test.go files.
package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"dtools2/rest"
)

// Request is a request received by the fake daemon
type Request struct {
	Method string
	Path   string
	Query  url.Values
	Header http.Header
	Body   []byte
}

// Daemon is a fake docker/podman REST API. Routes use http.ServeMux patterns,
// e.g. "POST /containers/{id}/kill"; registering a pattern again replaces its handler, so tests
// can start from healthy routes and override one to inject a failure.
// Unrouted requests get a 404 with a docker-style JSON message.
type Daemon struct {
	t      *testing.T
	server *httptest.Server

	mu       sync.Mutex
	routes   map[string]http.HandlerFunc
	mux      *http.ServeMux
	requests []Request
}

// NewDaemon starts a fake daemon, closed automatically at the end of the test
func NewDaemon(t *testing.T) *Daemon {
	t.Helper()
	d := &Daemon{t: t, mux: http.NewServeMux(), routes: map[string]http.HandlerFunc{}}
	d.server = httptest.NewServer(http.HandlerFunc(d.serve))
	t.Cleanup(d.server.Close)
	return d
}

func (d *Daemon) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	d.mu.Lock()
	d.requests = append(d.requests, Request{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.Query(),
		Header: r.Header.Clone(),
		Body:   body,
	})
	mux := d.mux
	d.mu.Unlock()
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Dispatch through the mux itself (not the handler returned by Handler()), so that r.PathValue works
	if _, pattern := mux.Handler(r); pattern != "" {
		mux.ServeHTTP(w, r)
		return
	}
	JSON(w, http.StatusNotFound, map[string]string{"message": "fake daemon: no route for " + r.Method + " " + r.URL.Path})
}

// Handle registers (or replaces) the handler for a ServeMux pattern
func (d *Daemon) Handle(pattern string, h http.HandlerFunc) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.routes[pattern] = h
	// ServeMux refuses duplicate patterns, so it is rebuilt from the route table
	mux := http.NewServeMux()
	for p, fn := range d.routes {
		mux.HandleFunc(p, fn)
	}
	d.mux = mux
}

// Reply registers a handler that always replies status with v JSON-encoded (no body if v is nil)
func (d *Daemon) Reply(pattern string, status int, v any) {
	d.Handle(pattern, func(w http.ResponseWriter, _ *http.Request) { JSON(w, status, v) })
}

// Client returns a rest.Client talking to the fake daemon, without an API version prefix
func (d *Daemon) Client() *rest.Client {
	d.t.Helper()
	c, err := rest.NewClient(rest.Config{Host: d.server.URL})
	if err != nil {
		d.t.Fatalf("rest.NewClient: %v", err)
	}
	return c
}

// URL returns the fake daemon's base URL
func (d *Daemon) URL() string { return d.server.URL }

// Requests returns a copy of every request received so far
func (d *Daemon) Requests() []Request {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]Request(nil), d.requests...)
}

// Calls returns the requests matching method and path
func (d *Daemon) Calls(method, path string) []Request {
	var out []Request
	for _, r := range d.Requests() {
		if r.Method == method && r.Path == path {
			out = append(out, r)
		}
	}
	return out
}

// Sequence returns "METHOD path" for every request received, in order
func (d *Daemon) Sequence() []string {
	var out []string
	for _, r := range d.Requests() {
		out = append(out, r.Method+" "+r.Path)
	}
	return out
}

// JSON writes v as a JSON response; a nil v writes no body
func JSON(w http.ResponseWriter, status int, v any) {
	if v == nil {
		w.WriteHeader(status)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Container builds a minimal /containers/json entry
func Container(id, name, state string, volumes ...string) map[string]any {
	mounts := []map[string]any{}
	for _, v := range volumes {
		mounts = append(mounts, map[string]any{"Type": "volume", "Name": v, "Destination": "/data"})
	}
	return map[string]any{"Id": id, "Names": []string{"/" + name}, "State": state, "Image": "alpine:latest", "Mounts": mounts}
}

// CaptureStdout runs fn and returns what it wrote to os.Stdout
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w

	done := make(chan []byte)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()

	defer func() { os.Stdout = orig }()
	fn()
	w.Close()
	return string(<-done)
}

// IsolateHome points HOME (and DOCKER_CONFIG) to a temp dir holding an empty dtools config
// directory, so tests never read or write the user's real files. Returns the dtools config dir.
func IsolateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("DOCKER_CONFIG", filepath.Join(home, ".docker"))
	dir := filepath.Join(home, ".config", "JFG", "dtools")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	return dir
}
