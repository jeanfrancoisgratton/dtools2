// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/system/cp_helpers_test.go

package system

import "testing"

func TestSplitContainerPath(t *testing.T) {
	cases := []struct {
		in        string
		wantRef   string
		wantPath  string
		wantOK    bool
	}{
		{"web:/etc/hosts", "web", "/etc/hosts", true},
		{"abc123:/var/log", "abc123", "/var/log", true},
		{"/host/only/path", "", "", false}, // no colon
		{":/leading", "", "", false},       // empty ref
		{"web:", "", "", false},            // empty path
	}
	for _, c := range cases {
		ref, path, ok := splitContainerPath(c.in)
		if ref != c.wantRef || path != c.wantPath || ok != c.wantOK {
			t.Errorf("splitContainerPath(%q) = (%q, %q, %v); want (%q, %q, %v)",
				c.in, ref, path, ok, c.wantRef, c.wantPath, c.wantOK)
		}
	}
}

func TestNormalizeHostContentsOnly(t *testing.T) {
	cases := []struct {
		in           string
		wantPath     string
		wantContents bool
	}{
		{"dir/.", "dir", true},
		{"/abs/dir/.", "/abs/dir", true},
		{"dir", "dir", false},
		{"dir/", "dir/", false},
	}
	for _, c := range cases {
		path, contents := normalizeHostContentsOnly(c.in)
		if path != c.wantPath || contents != c.wantContents {
			t.Errorf("normalizeHostContentsOnly(%q) = (%q, %v); want (%q, %v)",
				c.in, path, contents, c.wantPath, c.wantContents)
		}
	}
}

func TestAddDotIfDir(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"dir", "dir/."},
		{"dir/", "dir/."},
		{"dir/.", "dir/."},
		{"nested/dir/", "nested/dir/."},
	}
	for _, c := range cases {
		if got := addDotIfDir(c.in); got != c.want {
			t.Errorf("addDotIfDir(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestSanitizeTarName(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"foo/bar.txt", "foo/bar.txt"},
		{"/leading/slash", "leading/slash"},
		{"./relative", "relative"},
		{".", ""},
		{"", ""},
	}
	for _, c := range cases {
		if got := sanitizeTarName(c.in); got != c.want {
			t.Errorf("sanitizeTarName(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestIsSubPath(t *testing.T) {
	cases := []struct {
		base   string
		target string
		want   bool
	}{
		{"/base", "/base/child", true},
		{"/base", "/base", true},
		{"/base", "/base/a/b/c", true},
		{"/base", "/base/../escape", false},
		{"/base", "/other", false},
	}
	for _, c := range cases {
		if got := isSubPath(c.base, c.target); got != c.want {
			t.Errorf("isSubPath(%q, %q) = %v; want %v", c.base, c.target, got, c.want)
		}
	}
}

func TestDecodeB64(t *testing.T) {
	// "hello" encoded in the various base64 flavours decodeB64 accepts.
	cases := []string{
		"aGVsbG8=", // std
		"aGVsbG8",  // raw std (no padding)
	}
	for _, enc := range cases {
		got, err := decodeB64(enc)
		if err != nil {
			t.Errorf("decodeB64(%q) unexpected error: %v", enc, err)
			continue
		}
		if string(got) != "hello" {
			t.Errorf("decodeB64(%q) = %q; want %q", enc, got, "hello")
		}
	}
}
