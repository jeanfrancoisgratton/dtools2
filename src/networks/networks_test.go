// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/networks/networks_test.go

package networks

import (
	"dtools2/containers"
	"dtools2/extras"
	"dtools2/internal/testutil"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

func resetGlobals(t *testing.T) {
	t.Helper()
	driver, internal, attachable, ipv6, rmBL, force := NetworkDriverName, NetworkInternalUse, NetworkAttachable, NetworkEnableIPv6, RemoveBlacklisted, ForceNetworkDetach
	quiet, onlyRunning, outFmt := rest.QuietOutput, containers.OnlyRunningContainers, extras.OutputFormat
	rest.QuietOutput = true
	t.Cleanup(func() {
		NetworkDriverName, NetworkInternalUse, NetworkAttachable, NetworkEnableIPv6, RemoveBlacklisted, ForceNetworkDetach = driver, internal, attachable, ipv6, rmBL, force
		rest.QuietOutput, containers.OnlyRunningContainers, extras.OutputFormat = quiet, onlyRunning, outFmt
	})
}

// netDaemon serves two networks: "bridge" (built-in) and "appnet"
func netDaemon(t *testing.T) *testutil.Daemon {
	t.Helper()
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /networks", http.StatusOK, []map[string]any{
		{"Name": "bridge", "Id": "net000000000001", "Driver": "bridge", "Scope": "local"},
		{"Name": "appnet", "Id": "net000000000002", "Driver": "bridge", "Scope": "local"},
	})
	return d
}

// noPanic turns a panic into a test failure, so one crashing case does not abort the whole package run
func noPanic(t *testing.T, what string) {
	if r := recover(); r != nil {
		t.Errorf("%s panicked: %v", what, r)
	}
}

func TestFetchNetworkList(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   int
		err    bool
	}{
		{http.StatusOK, `[{"Name":"a"},{"Name":"b"}]`, 2, false},
		{http.StatusNoContent, ``, 0, false},
		{http.StatusOK, `{`, 0, true},
		{http.StatusInternalServerError, `{}`, 0, true},
	}
	for _, c := range cases {
		resetGlobals(t)
		d := testutil.NewDaemon(t)
		d.Handle("GET /networks", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(c.status)
			w.Write([]byte(c.body))
		})
		ns, err := fetchNetworkList(d.Client())
		if (err != nil) != c.err || len(ns) != c.want {
			t.Errorf("status %d body %q: got %d networks, err %v; want %d, error=%v", c.status, c.body, len(ns), err, c.want, c.err)
		}
	}
}

func TestName2IDAndID2Name(t *testing.T) {
	d := netDaemon(t)
	if id, err := Name2ID(d.Client(), "appnet"); err != nil || id != "net000000000002" {
		t.Errorf("Name2ID(appnet) = %q, %v", id, err)
	}
	if _, err := Name2ID(d.Client(), "ghost"); err == nil || !strings.Contains(err.Message, "ghost") {
		t.Errorf("Name2ID(ghost) = %v; want a not-found error naming it", err)
	}
	if name, err := ID2Name(d.Client(), "net000000000001"); err != nil || name != "bridge" {
		t.Errorf("ID2Name = %q, %v", name, err)
	}
	if _, err := ID2Name(d.Client(), "nope"); err == nil {
		t.Error("ID2Name(unknown): want an error")
	}

	empty := testutil.NewDaemon(t)
	empty.Reply("GET /networks", http.StatusOK, []any{})
	if _, err := Name2ID(empty.Client(), "x"); err == nil {
		t.Error("Name2ID with no networks: want an error")
	}
}

func TestNetworkListInUse(t *testing.T) {
	d := netDaemon(t)
	c := testutil.Container("aaa111", "web", "exited")
	c["NetworkSettings"] = map[string]any{"Networks": map[string]any{"appnet": map[string]string{"NetworkID": "net000000000002"}}}
	d.Reply("GET /containers/json", http.StatusOK, []any{c})

	ns, err := NetworkList(d.Client(), false)
	if err != nil {
		t.Fatalf("NetworkList: %v", err)
	}
	inUse := map[string]bool{}
	for _, n := range ns {
		inUse[n.Name] = n.InUse
	}
	if inUse["bridge"] || !inUse["appnet"] {
		t.Errorf("InUse = %v; want only appnet in use (stopped containers count)", inUse)
	}

	rest.QuietOutput = false
	out := testutil.CaptureStdout(t, func() { _, _ = NetworkList(d.Client(), true) })
	if !strings.Contains(out, "appnet") || !strings.Contains(out, "net000000000") {
		t.Errorf("table output %q: want names and short IDs", out)
	}

	extras.OutputFormat = "Name"
	out = testutil.CaptureStdout(t, func() { _, _ = NetworkList(d.Client(), true) })
	if got := strings.Fields(out); len(got) != 2 {
		t.Errorf("--format Name output %q; want 2 names", out)
	}
}

func TestAddNetwork(t *testing.T) {
	d := netDaemon(t)
	d.Handle("POST /networks/create", func(w http.ResponseWriter, r *http.Request) {
		var req NetworkCreateRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req.Name == "dup" {
			testutil.JSON(w, http.StatusConflict, map[string]string{"message": "network with name dup already exists"})
			return
		}
		if req.Name == "raw" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("plain failure"))
			return
		}
		testutil.JSON(w, http.StatusCreated, map[string]string{"Id": "newnet"})
	})

	NetworkDriverName, NetworkInternalUse, NetworkAttachable, NetworkEnableIPv6 = "macvlan", true, true, true
	if err := AddNetwork(d.Client(), "lan"); err != nil {
		t.Fatalf("AddNetwork: %v", err)
	}
	var sent NetworkCreateRequest
	_ = json.Unmarshal(d.Calls(http.MethodPost, "/networks/create")[0].Body, &sent)
	want := NetworkCreateRequest{Name: "lan", CheckDuplicate: true, Driver: "macvlan", Internal: true, Attachable: true, EnableIPv6: true}
	if fmt.Sprint(sent) != fmt.Sprint(want) {
		t.Errorf("payload = %+v; want %+v", sent, want)
	}

	if err := AddNetwork(d.Client(), "dup"); err == nil || !strings.Contains(err.Message, "already exists") {
		t.Errorf("AddNetwork(dup) = %v; want the daemon's message", err)
	}
	if err := AddNetwork(d.Client(), "raw"); err == nil || !strings.Contains(err.Message, "plain failure") {
		t.Errorf("AddNetwork(raw) = %v; want the raw body in the message", err)
	}
}

func TestAttachDetach(t *testing.T) {
	d := netDaemon(t)
	d.Reply("POST /networks/{id}/connect", http.StatusOK, nil)
	d.Reply("POST /networks/{id}/disconnect", http.StatusOK, nil)

	if err := AttachNetwork(d.Client(), "appnet", "web"); err != nil {
		t.Fatalf("AttachNetwork: %v", err)
	}
	var conn NetworkConnectRequest
	_ = json.Unmarshal(d.Calls(http.MethodPost, "/networks/net000000000002/connect")[0].Body, &conn)
	if conn.Container != "web" {
		t.Errorf("connect payload = %+v; want container web", conn)
	}

	ForceNetworkDetach = true
	if err := DetachNetwork(d.Client(), "appnet", "web"); err != nil {
		t.Fatalf("DetachNetwork: %v", err)
	}
	var disc NetworkDisconnectRequest
	_ = json.Unmarshal(d.Calls(http.MethodPost, "/networks/net000000000002/disconnect")[0].Body, &disc)
	if disc.Container != "web" || !disc.Force {
		t.Errorf("disconnect payload = %+v; want container web, force", disc)
	}

	if err := AttachNetwork(d.Client(), "ghost", "web"); err == nil {
		t.Error("AttachNetwork to an unknown network: want an error")
	}
	d.Reply("POST /networks/{id}/connect", http.StatusNotFound, map[string]string{"message": "no such container"})
	if err := AttachNetwork(d.Client(), "appnet", "ghost"); err == nil {
		t.Error("AttachNetwork on HTTP 404: want an error")
	}
}

func TestRemoveNetwork(t *testing.T) {
	dir := testutil.IsolateHome(t)
	os.WriteFile(filepath.Join(dir, "blacklist.json"), []byte(`{"Networks":["bridge"]}`), 0o600)
	d := netDaemon(t)
	d.Reply("DELETE /networks/{id}", http.StatusNoContent, nil)

	if err := RemoveNetwork(d.Client(), []string{"appnet", "bridge"}); err != nil {
		t.Fatalf("RemoveNetwork: %v", err)
	}
	if len(d.Calls(http.MethodDelete, "/networks/net000000000002")) != 1 {
		t.Error("appnet should have been removed by ID")
	}
	if len(d.Calls(http.MethodDelete, "/networks/net000000000001")) != 0 {
		t.Error("a blacklisted network must not be removed without -B")
	}
}

func TestRemoveNetworkBlacklistedWithFlag(t *testing.T) {
	for _, quiet := range []bool{false, true} {
		dir := testutil.IsolateHome(t)
		os.WriteFile(filepath.Join(dir, "blacklist.json"), []byte(`{"Networks":["appnet"]}`), 0o600)
		d := netDaemon(t)
		d.Reply("DELETE /networks/{id}", http.StatusNoContent, nil)
		RemoveBlacklisted, rest.QuietOutput = true, quiet

		_ = testutil.CaptureStdout(t, func() { _ = RemoveNetwork(d.Client(), []string{"appnet"}) })
		if len(d.Calls(http.MethodDelete, "/networks/net000000000002")) != 1 {
			t.Errorf("quiet=%v: -B should remove a blacklisted network", quiet)
		}
	}
}

func TestRemoveNetworkErrors(t *testing.T) {
	cases := []struct {
		status int
		want   string
	}{
		{http.StatusForbidden, "not allowed to be removed"},
		{http.StatusNotFound, "is not found"},
		// e.g. "network has active endpoints"
		{http.StatusConflict, "409"},
		{http.StatusInternalServerError, "500"},
	}
	for _, c := range cases {
		t.Run(fmt.Sprint(c.status), func(t *testing.T) {
			testutil.IsolateHome(t)
			d := netDaemon(t)
			d.Reply("DELETE /networks/{id}", c.status, map[string]string{"message": "daemon says no"})

			defer noPanic(t, fmt.Sprintf("RemoveNetwork on HTTP %d", c.status))
			err := RemoveNetwork(d.Client(), []string{"appnet"})
			if err == nil || !strings.Contains(err.Title+" "+err.Message, c.want) {
				t.Errorf("RemoveNetwork on HTTP %d = %v; want it to mention %q", c.status, err, c.want)
			}
		})
	}
}

func TestInspectNetwork(t *testing.T) {
	d := netDaemon(t)
	d.Reply("GET /networks/appnet", http.StatusOK, map[string]any{"Name": "appnet", "Id": "net000000000002"})
	d.Reply("GET /networks/ghost", http.StatusNotFound, map[string]string{"message": "not found"})

	var n NetworkInspect
	var err *ce.CustomError
	out := testutil.CaptureStdout(t, func() { n, err = InspectNetwork(d.Client(), "appnet") })
	if err != nil || n.Name != "appnet" {
		t.Fatalf("InspectNetwork = %+v, %v", n, err)
	}
	if !strings.Contains(out, `"Name":"appnet"`) {
		t.Errorf("quiet output %q: want the raw JSON", out)
	}
	if _, err := InspectNetwork(d.Client(), "ghost"); err == nil || !strings.Contains(err.Title, "404") {
		t.Errorf("InspectNetwork(ghost) = %v; want an HTTP 404 error", err)
	}
}
