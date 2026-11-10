// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/containers/port_test.go

package containers

import (
	"dtools2/extras"
	"dtools2/internal/testutil"
	"dtools2/rest"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func newPortDaemon(t *testing.T) *testutil.Daemon {
	t.Helper()
	resetGlobals(t)
	savedJSON := extras.OutputJSON
	t.Cleanup(func() { extras.OutputJSON = savedJSON })

	d := testutil.NewDaemon(t)
	d.Reply("GET /containers/web/json", http.StatusOK, map[string]any{
		"Id": "abc", "Name": "/web",
		"NetworkSettings": map[string]any{"Ports": map[string]any{
			"8080/tcp": []map[string]string{{"HostIp": "0.0.0.0", "HostPort": "18080"}, {"HostIp": "::", "HostPort": "18080"}},
			"53/udp":   []map[string]string{{"HostIp": "127.0.0.1", "HostPort": "5353"}},
			"9000/tcp": nil, // exposed, not published
		}},
	})
	return d
}

func TestPortContainerAll(t *testing.T) {
	d := newPortDaemon(t)
	got, cerr := PortContainer(d.Client(), "web", "")
	if cerr != nil {
		t.Fatalf("PortContainer: %v", cerr)
	}
	want := []PortMapping{
		{53, "udp", "127.0.0.1", "5353"},
		{8080, "tcp", "0.0.0.0", "18080"},
		{8080, "tcp", "::", "18080"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v; want %+v", got, want)
	}
}

func TestPortContainerFilter(t *testing.T) {
	d := newPortDaemon(t)
	for _, f := range []string{"8080", "8080/tcp", "8080/TCP"} {
		got, cerr := PortContainer(d.Client(), "web", f)
		if cerr != nil || len(got) != 2 {
			t.Errorf("filter %q = %+v, %v; want 2 mappings", f, got, cerr)
		}
	}
	if _, cerr := PortContainer(d.Client(), "web", "8080/udp"); cerr == nil || !strings.Contains(cerr.Message, "no public port '8080/udp'") {
		t.Errorf("unpublished filter = %v; want a 'no public port' error", cerr)
	}
	if _, cerr := PortContainer(d.Client(), "web", "9000"); cerr == nil {
		t.Error("exposed-only port should not be reported as published")
	}
	for _, f := range []string{"abc", "0", "70000/tcp"} {
		if _, cerr := PortContainer(d.Client(), "web", f); cerr == nil {
			t.Errorf("filter %q should be rejected", f)
		}
	}
}

func TestPortContainerOutput(t *testing.T) {
	d := newPortDaemon(t)
	rest.QuietOutput = false

	out := testutil.CaptureStdout(t, func() { PortContainer(d.Client(), "web", "53/udp") })
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "CONTAINER PORT") || strings.Join(strings.Fields(lines[1]), " ") != "53/udp 127.0.0.1 5353" {
		t.Errorf("table output = %q", out)
	}

	extras.OutputJSON = true
	out = testutil.CaptureStdout(t, func() { PortContainer(d.Client(), "web", "53/udp") })
	var got []PortMapping
	if err := json.Unmarshal([]byte(out), &got); err != nil || len(got) != 1 || got[0].HostPort != "5353" {
		t.Errorf("JSON output = %q (%v)", out, err)
	}
}
