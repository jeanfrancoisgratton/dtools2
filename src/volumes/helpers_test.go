// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/volumes/helpers_test.go

package volumes

import (
	"dtools2/containers"
	"reflect"
	"testing"
)

func TestFormatCreated(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"2026-01-02T15:04:05Z", "2026.01.02 15:04:05"},
		{"2026-01-02T15:04:05.123456789Z", "2026.01.02 15:04:05"},
		// Unparseable input is returned unchanged.
		{"not-a-date", "not-a-date"},
	}
	for _, c := range cases {
		if got := formatCreated(c.in); got != c.want {
			t.Errorf("formatCreated(%q) = %q; want %q", c.in, got, c.want)
		}
	}
}

func TestContainerDisplayName(t *testing.T) {
	cases := []struct {
		name string
		c    containers.ContainerSummary
		want string
	}{
		{
			"named running",
			containers.ContainerSummary{Names: []string{"/web"}, State: "running"},
			"web (running)",
		},
		{
			"named no state",
			containers.ContainerSummary{Names: []string{"/db"}},
			"db",
		},
		{
			"id fallback",
			containers.ContainerSummary{ID: "abcdef0123456789", State: "exited"},
			"abcdef012345 (exited)",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := containerDisplayName(c.c); got != c.want {
				t.Errorf("containerDisplayName = %q; want %q", got, c.want)
			}
		})
	}
}

func TestComputeVolumeUsage(t *testing.T) {
	cs := []containers.ContainerSummary{
		{
			Names: []string{"/web"}, State: "running",
			Mounts: []containers.MountsStruct{
				{Type: "volume", Name: "data"},
				{Type: "bind", Source: "/host/x"}, // ignored: not a volume
			},
		},
		{
			Names: []string{"/api"}, State: "running",
			Mounts: []containers.MountsStruct{
				{Type: "volume", Name: "data"},
				{Type: "volume", Name: ""}, // ignored: anonymous
			},
		},
	}

	got := computeVolumeUsage(cs)
	want := map[string][]string{
		"data": {"api (running)", "web (running)"}, // sorted
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("computeVolumeUsage = %v; want %v", got, want)
	}
}
