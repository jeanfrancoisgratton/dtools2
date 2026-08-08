// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/blacklist/helpers_test.go

package blacklist

import "testing"

func TestGetSlice(t *testing.T) {
	rb := &ResourceBlacklist{
		Volumes:    []string{"v1"},
		Networks:   []string{"n1"},
		Images:     []string{"i1"},
		Containers: []string{"c1"},
	}

	cases := []struct {
		resourceType string
		wantFirst    string
	}{
		{"volume", "v1"},
		{"volumes", "v1"},
		{"NETWORK", "n1"},
		{"Networks", "n1"},
		{"image", "i1"},
		{"images", "i1"},
		{"container", "c1"},
		{"containers", "c1"},
	}

	for _, c := range cases {
		t.Run(c.resourceType, func(t *testing.T) {
			slicePtr, err := getSlice(rb, c.resourceType)
			if err != nil {
				t.Fatalf("getSlice(%q) returned error: %v", c.resourceType, err)
			}
			if got := (*slicePtr)[0]; got != c.wantFirst {
				t.Errorf("getSlice(%q)[0] = %q; want %q", c.resourceType, got, c.wantFirst)
			}
		})
	}
}

func TestGetSliceUnknown(t *testing.T) {
	rb := &ResourceBlacklist{}
	if _, err := getSlice(rb, "widget"); err == nil {
		t.Errorf("getSlice with unknown type should return an error")
	}
}

// The returned pointer must alias the underlying slice so mutations persist.
func TestGetSliceIsMutable(t *testing.T) {
	rb := &ResourceBlacklist{}
	slicePtr, err := getSlice(rb, "images")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	*slicePtr = append(*slicePtr, "alpine:latest")
	if len(rb.Images) != 1 || rb.Images[0] != "alpine:latest" {
		t.Errorf("mutation through pointer did not affect the blacklist: %+v", rb.Images)
	}
}
