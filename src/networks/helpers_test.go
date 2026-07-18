// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/networks/helpers_test.go

package networks

import (
	"dtools2/containers"
	"testing"
)

func TestComputeNetworkUsageEmpty(t *testing.T) {
	used, byName, byID := computeNetworkUsage(nil)
	if used {
		t.Errorf("expected no networks in use")
	}
	if len(byName) != 0 || len(byID) != 0 {
		t.Errorf("expected empty usage sets, got name=%v id=%v", byName, byID)
	}
}

func TestComputeNetworkUsage(t *testing.T) {
	cs := []containers.ContainerSummary{
		{
			NetworkSettings: &containers.ContainerNetworkSettings{
				Networks: map[string]containers.EndpointSummary{
					"bridge":  {NetworkID: "netid-bridge"},
					"backend": {NetworkID: "netid-backend"},
				},
			},
		},
		{
			// No network settings: must be skipped without panicking.
			NetworkSettings: nil,
		},
		{
			NetworkSettings: &containers.ContainerNetworkSettings{
				Networks: map[string]containers.EndpointSummary{
					"bridge": {NetworkID: "netid-bridge"}, // duplicate
				},
			},
		},
	}

	used, byName, byID := computeNetworkUsage(cs)
	if !used {
		t.Fatalf("expected networks to be in use")
	}
	for _, name := range []string{"bridge", "backend"} {
		if _, ok := byName[name]; !ok {
			t.Errorf("expected network %q to be marked used", name)
		}
	}
	for _, id := range []string{"netid-bridge", "netid-backend"} {
		if _, ok := byID[id]; !ok {
			t.Errorf("expected network id %q to be marked used", id)
		}
	}
	if len(byName) != 2 {
		t.Errorf("expected 2 unique network names, got %d", len(byName))
	}
}
