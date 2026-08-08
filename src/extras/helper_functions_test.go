// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/extras/helper_functions_test.go

package extras

import (
	"reflect"
	"testing"
)

func TestSplitURI(t *testing.T) {
	cases := []struct {
		name     string
		ref      string
		wantRepo string
		wantTag  string
	}{
		{"no tag", "alpine", "alpine", "latest"},
		{"explicit tag", "alpine:3.19", "alpine", "3.19"},
		{"registry port no tag", "registry:5000/repo/image", "registry:5000/repo/image", "latest"},
		{"registry port with tag", "registry:5000/repo/image:1.2.3", "registry:5000/repo/image", "1.2.3"},
		{"namespaced no tag", "library/nginx", "library/nginx", "latest"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, tag := SplitURI(c.ref)
			if repo != c.wantRepo || tag != c.wantTag {
				t.Errorf("SplitURI(%q) = (%q, %q); want (%q, %q)", c.ref, repo, tag, c.wantRepo, c.wantTag)
			}
		})
	}
}

func TestStringsTrim(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"  hello  ", "hello"},
		{"line1\nline2", "line1 line2"},
		{"a\r\nb", "a  b"},
		{"\t spaced \t", "spaced"},
		{"", ""},
	}

	for _, c := range cases {
		if got := StringsTrim(c.in); got != c.want {
			t.Errorf("StringsTrim(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestUnique(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"nil", nil, nil},
		{"empty", []string{}, nil},
		{"dedup and sort", []string{"b", "a", "b", "c", "a"}, []string{"a", "b", "c"}},
		{"already unique", []string{"x", "y"}, []string{"x", "y"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Unique(c.in); !reflect.DeepEqual(got, c.want) {
				t.Errorf("Unique(%v) = %v; want %v", c.in, got, c.want)
			}
		})
	}
}

// Unique must not mutate its input slice.
func TestUniqueDoesNotMutateInput(t *testing.T) {
	in := []string{"c", "a", "b", "a"}
	_ = Unique(in)
	want := []string{"c", "a", "b", "a"}
	if !reflect.DeepEqual(in, want) {
		t.Errorf("Unique mutated input: got %v; want %v", in, want)
	}
}
