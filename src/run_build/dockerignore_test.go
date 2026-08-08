// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/run_build/dockerignore_test.go

package run_build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobToRegex(t *testing.T) {
	cases := []struct {
		glob string
		want string
	}{
		{"foo", "foo"},
		{"*.log", `[^/]*\.log`},
		{"a?b", `a[^/]b`},
		{"**/bar", `.*/bar`},
		{"a.b+c", `a\.b\+c`},
	}
	for _, c := range cases {
		if got := globToRegex(c.glob); got != c.want {
			t.Errorf("globToRegex(%q) = %q; want %q", c.glob, got, c.want)
		}
	}
}

func TestIsIgnoredWithMatcher(t *testing.T) {
	// Build a matcher by hand from a set of patterns.
	// Patterns are already stripped of their leading "!" here, mirroring what
	// newIgnoreMatcher does before handing the pattern to the regex compiler.
	patterns := []struct {
		pat string
		neg bool
	}{
		{"*.log", false},
		{"node_modules", false},
		{"keep.log", true},
		{"/build", false},       // single-component anchored pattern
		{"/dist/cache", false},  // multi-component anchored pattern
	}

	m := &ignoreMatcher{dockerfileRel: "Dockerfile"}
	for _, p := range patterns {
		re, ok := dockerignorePatternToRegex(p.pat)
		if !ok {
			t.Fatalf("failed to compile pattern %q", p.pat)
		}
		m.rules = append(m.rules, ignoreRule{neg: p.neg, regex: re})
	}

	cases := []struct {
		rel  string
		want bool
	}{
		{"app.log", true},                 // matches *.log
		{"src/app.log", true},             // *.log matches anywhere
		{"keep.log", false},               // negated
		{"node_modules/foo/bar.js", true}, // basename dir match
		{"build/out.bin", true},           // anchored /build at root
		{"other/build/out.bin", false},    // anchored /build must not match nested
		{"dist/cache/x", true},            // anchored /dist/cache at root
		{"other/dist/cache/x", false},     // anchored: must not match nested
		{"main.go", false},                // no rule matches
		{"Dockerfile", false},             // always kept
		{".dockerignore", false},          // always kept
	}

	for _, c := range cases {
		if got := m.isIgnored(c.rel); got != c.want {
			t.Errorf("isIgnored(%q) = %v; want %v", c.rel, got, c.want)
		}
	}
}

func TestNewIgnoreMatcherMissingFile(t *testing.T) {
	// A context directory with no .dockerignore must yield an empty (nil rules) matcher.
	dir := t.TempDir()
	m, err := newIgnoreMatcher(dir, "Dockerfile")
	if err != nil {
		t.Fatalf("newIgnoreMatcher returned error: %v", err)
	}
	if len(m.rules) != 0 {
		t.Errorf("expected no rules, got %d", len(m.rules))
	}
	if m.isIgnored("anything.txt") {
		t.Errorf("nothing should be ignored when there is no .dockerignore")
	}
}

func TestNewIgnoreMatcherParsesFile(t *testing.T) {
	dir := t.TempDir()
	content := "# a comment\n\n*.tmp\n!important.tmp\n"
	if err := os.WriteFile(filepath.Join(dir, ".dockerignore"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := newIgnoreMatcher(dir, "Dockerfile")
	if err != nil {
		t.Fatalf("newIgnoreMatcher returned error: %v", err)
	}
	// Two effective rules: *.tmp and !important.tmp (comment + blank skipped).
	if len(m.rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(m.rules))
	}
	if !m.isIgnored("scratch.tmp") {
		t.Errorf("scratch.tmp should be ignored")
	}
	if m.isIgnored("important.tmp") {
		t.Errorf("important.tmp should be un-ignored by negation")
	}
}
