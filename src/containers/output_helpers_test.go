// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/containers/output_helpers_test.go

package containers

import (
	"strings"
	"testing"
)

func TestGetImageTag(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"alpine", "alpine:latest"},
		{"alpine:3.19", "alpine:3.19"},
		{"registry:5000/repo/image", "registry:5000/repo/image:latest"},
		{"registry:5000/repo/image:1.0", "registry:5000/repo/image:1.0"},
	}
	for _, c := range cases {
		if got := getImageTag(c.name); got != c.want {
			t.Errorf("getImageTag(%q) = %q; want %q", c.name, got, c.want)
		}
	}
}

func TestPrettifyPortsList(t *testing.T) {
	ports := []PortsStruct{
		{Type: "tcp", PublicPort: 8080, PrivatePort: 80},
		{Type: "tcp", PublicPort: 0, PrivatePort: 443},
		// Duplicate of the first entry, placed last: must be de-duplicated
		// without leaving a trailing delimiter.
		{Type: "tcp", PublicPort: 8080, PrivatePort: 80},
	}
	got := prettifyPortsList(ports, ", ")
	want := "tcp/8080->80, tcp/443"
	if got != want {
		t.Errorf("prettifyPortsList = %q; want %q", got, want)
	}
}

func TestPrettifyPortsListEmpty(t *testing.T) {
	if got := prettifyPortsList(nil, ", "); got != "" {
		t.Errorf("prettifyPortsList(nil) = %q; want empty", got)
	}
}

func TestPrettifyMounts(t *testing.T) {
	mounts := []MountsStruct{
		{Type: "bind", Source: "/host/data", Destination: "/data", RW: true},
		{Type: "volume", Name: "myvol", Destination: "/var/lib", RW: false},
	}
	got := prettifyMounts(mounts, " | ")

	// Colour/symbol prefixes are environment-dependent, so assert on the
	// stable path fragments rather than exact bytes.
	for _, frag := range []string{"/host/data:/data", "[myvol]:/var/lib"} {
		if !strings.Contains(got, frag) {
			t.Errorf("prettifyMounts output %q missing fragment %q", got, frag)
		}
	}
}

func TestFormatSize(t *testing.T) {
	cases := []struct {
		sz   int64
		want string
	}{
		{5_000_000, "5.000 MB"},
		{5_000_000_000, "5.000 GB"},
	}
	for _, c := range cases {
		if got := formatSize(c.sz); got != c.want {
			t.Errorf("formatSize(%d) = %q; want %q", c.sz, got, c.want)
		}
	}
}
