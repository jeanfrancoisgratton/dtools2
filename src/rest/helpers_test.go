// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/rest/helpers_test.go

package rest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJoinURLPath(t *testing.T) {
	cases := []struct {
		base string
		add  string
		want string
	}{
		{"", "/containers/json", "/containers/json"},
		{"/", "/containers/json", "/containers/json"},
		{"/v1.43", "/containers/json", "/v1.43/containers/json"},
		{"/v1.43/", "containers/json", "/v1.43/containers/json"},
		{"/v1.43/", "/containers/json", "/v1.43/containers/json"},
	}
	for _, c := range cases {
		if got := joinURLPath(c.base, c.add); got != c.want {
			t.Errorf("joinURLPath(%q, %q) = %q; want %q", c.base, c.add, got, c.want)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	if got := NormalizePath(""); got != "" {
		t.Errorf("NormalizePath(\"\") = %q; want empty", got)
	}
	if got := NormalizePath("/etc/certs"); got != "/etc/certs" {
		t.Errorf("NormalizePath(absolute) = %q; want unchanged", got)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		got := NormalizePath("~/certs/ca.pem")
		want := filepath.Join(home, "certs/ca.pem")
		if got != want {
			t.Errorf("NormalizePath(~) = %q; want %q", got, want)
		}
	}
}

func TestShowHost(t *testing.T) {
	// showNow=false must not print, and returns the (possibly rewritten) URI.
	if got := ShowHost("unix:///var/run/docker.sock", false); got != "localhost (unix socket)" {
		t.Errorf("ShowHost(unix) = %q; want localhost label", got)
	}
	remote := "tcp://10.0.0.5:2376"
	if got := ShowHost(remote, false); got != remote {
		t.Errorf("ShowHost(remote) = %q; want %q", got, remote)
	}
}

func TestBuildTLSConfigDisabled(t *testing.T) {
	// UseTLS=false must return (nil, nil) regardless of other fields.
	cfg := Config{UseTLS: false, CACertPath: "/does/not/exist"}
	tlsCfg, err := buildTLSConfig(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tlsCfg != nil {
		t.Errorf("expected nil tls.Config when UseTLS is false")
	}
}

func TestBuildTLSConfigBadCA(t *testing.T) {
	cfg := Config{UseTLS: true, CACertPath: filepath.Join(t.TempDir(), "missing-ca.pem")}
	if _, err := buildTLSConfig(cfg); err == nil {
		t.Errorf("expected error for missing CA cert path")
	} else if !strings.Contains(err.Error(), "CA cert") {
		t.Errorf("error should mention the CA cert: %v", err)
	}
}
