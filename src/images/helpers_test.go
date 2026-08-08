// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/images/helpers_test.go

package images

import (
	"strings"
	"testing"
)

func TestSplitRepoTag(t *testing.T) {
	cases := []struct {
		ref      string
		wantRepo string
		wantTag  string
	}{
		{"alpine", "alpine", ""},
		{"alpine:3.19", "alpine", "3.19"},
		{"registry:5000/repo/image", "registry:5000/repo/image", ""},
		{"registry:5000/repo/image:1.0", "registry:5000/repo/image", "1.0"},
		{"alpine@sha256:deadbeef", "alpine", ""},
		{"alpine:3.19@sha256:deadbeef", "alpine", "3.19"},
	}
	for _, c := range cases {
		repo, tag := splitRepoTag(c.ref)
		if repo != c.wantRepo || tag != c.wantTag {
			t.Errorf("splitRepoTag(%q) = (%q, %q); want (%q, %q)", c.ref, repo, tag, c.wantRepo, c.wantTag)
		}
	}
}

func TestRegistryFromImageRef(t *testing.T) {
	cases := []struct {
		ref  string
		want string
	}{
		{"alpine", ""},
		{"library/nginx", ""},
		{"docker.io/library/nginx", "docker.io"},
		{"registry.example.com/app", "registry.example.com"},
		{"registry:5000/app", "registry:5000"},
		{"localhost/app", "localhost"},
		{"registry.example.com/app@sha256:abc", "registry.example.com"},
	}
	for _, c := range cases {
		if got := registryFromImageRef(c.ref); got != c.want {
			t.Errorf("registryFromImageRef(%q) = %q; want %q", c.ref, got, c.want)
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
	// A sub-megabyte value must still be reported in MB, not GB.
	if got := formatSize(250_000); !strings.HasSuffix(got, "MB") {
		t.Errorf("formatSize(250000) = %q; want a MB value", got)
	}
}
