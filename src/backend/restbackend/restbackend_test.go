// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/backend/restbackend/restbackend_test.go

package restbackend

import (
	"context"
	"dtools2/backend"
	"dtools2/internal/testutil"
	"dtools2/rest"
	"net/http"
	"path/filepath"
	"testing"
)

// newBackend returns a Backend wired to a fake daemon advertising API 1.50
func newBackend(t *testing.T) (backend.Backend, *testutil.Daemon) {
	t.Helper()
	testutil.IsolateHome(t)
	quiet := rest.QuietOutput
	rest.QuietOutput = true
	t.Cleanup(func() { rest.QuietOutput = quiet })

	d := testutil.NewDaemon(t)
	d.Reply("GET /version", http.StatusOK, map[string]string{"ApiVersion": "1.50"})
	b, err := New(context.Background(), rest.Config{Host: d.URL()})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return b, d
}

func TestNewNegotiatesVersion(t *testing.T) {
	b, d := newBackend(t)
	defer b.Close()
	if b.Name() != "docker/podman" {
		t.Errorf("Name() = %q", b.Name())
	}
	_ = b.System().Info()
	if seq := d.Sequence(); len(seq) < 2 || seq[1] != "GET /v1.50/info" {
		t.Errorf("requests = %v; want the negotiated /v1.50 prefix", seq)
	}
}

func TestNewPinnedVersionSkipsNegotiation(t *testing.T) {
	d := testutil.NewDaemon(t)
	b, err := New(context.Background(), rest.Config{Host: d.URL(), APIVersion: "1.43"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(d.Calls(http.MethodGet, "/version")) != 0 {
		t.Error("a pinned API version must not be negotiated")
	}
	_ = b.System().Info()
	if seq := d.Sequence(); len(seq) != 1 || seq[0] != "GET /v1.43/info" {
		t.Errorf("requests = %v; want GET /v1.43/info", seq)
	}
}

func TestNewErrors(t *testing.T) {
	d := testutil.NewDaemon(t)
	d.Reply("GET /version", http.StatusInternalServerError, nil)
	if _, err := New(context.Background(), rest.Config{Host: d.URL()}); err == nil {
		t.Error("a failed negotiation: want an error")
	}
	if _, err := New(context.Background(), rest.Config{Host: "ftp://nope"}); err == nil {
		t.Error("an invalid host: want an error")
	}
}

func TestCapabilities(t *testing.T) {
	b, _ := newBackend(t)
	for name, ok := range map[string]bool{
		"networks": func() bool { _, ok := b.Networks(); return ok }(),
		"volumes":  func() bool { _, ok := b.Volumes(); return ok }(),
		"runner":   func() bool { _, ok := b.Runner(); return ok }(),
		"builder":  func() bool { _, ok := b.Builder(); return ok }(),
		"copier":   func() bool { _, ok := b.FileCopier(); return ok }(),
	} {
		if !ok {
			t.Errorf("docker/podman must support %s", name)
		}
	}
}

// Every adapter method must reach the daemon through the matching package function;
// the fake daemon answers 404 to most calls, so only the first request matters here
func TestAdapterForwarding(t *testing.T) {
	tmp := t.TempDir()
	nets := func(b backend.Backend) backend.NetworkService { s, _ := b.Networks(); return s }
	vols := func(b backend.Backend) backend.VolumeService { s, _ := b.Volumes(); return s }

	cases := []struct {
		name  string
		call  func(b backend.Backend)
		first string
	}{
		{"containers list", func(b backend.Backend) { b.Containers().List(false) }, "GET /v1.50/containers/json"},
		{"containers info", func(b backend.Backend) { b.Containers().Info("web") }, "GET /v1.50/containers/json"},
		{"containers inspect", func(b backend.Backend) { b.Containers().Inspect("web") }, "GET /v1.50/containers/web/json"},
		{"containers remove", func(b backend.Backend) { b.Containers().Remove([]string{"web"}) }, "GET /v1.50/containers/json"},
		{"containers pause", func(b backend.Backend) { b.Containers().Pause([]string{"web"}) }, "GET /v1.50/containers/json"},
		{"containers unpause", func(b backend.Backend) { b.Containers().Unpause([]string{"web"}) }, "GET /v1.50/containers/json"},
		{"containers start", func(b backend.Backend) { b.Containers().Start([]string{"web"}) }, "GET /v1.50/containers/json"},
		{"containers startall", func(b backend.Backend) { b.Containers().StartAll() }, "GET /v1.50/containers/json"},
		{"containers stop", func(b backend.Backend) { b.Containers().Stop([]string{"web"}) }, "GET /v1.50/containers/json"},
		{"containers stopall", func(b backend.Backend) { b.Containers().StopAll() }, "GET /v1.50/containers/json"},
		{"containers rename", func(b backend.Backend) { b.Containers().Rename("a", "b") }, "GET /v1.50/containers/json"},
		{"containers kill", func(b backend.Backend) { b.Containers().Kill([]string{"web"}) }, "GET /v1.50/containers/json"},
		{"containers killall", func(b backend.Backend) { b.Containers().KillAll() }, "GET /v1.50/containers/json"},
		{"containers restart", func(b backend.Backend) { b.Containers().Restart([]string{"web"}) }, "GET /v1.50/containers/json"},
		{"containers restartall", func(b backend.Backend) { b.Containers().RestartAll() }, "GET /v1.50/containers/json"},
		{"images pull", func(b backend.Backend) { b.Images().Pull("alpine") }, "POST /v1.50/images/create"},
		{"images push", func(b backend.Backend) { testutil.CaptureStdout(t, func() { b.Images().Push("alpine") }) }, "POST /v1.50/images/alpine/push"},
		{"images list", func(b backend.Backend) { b.Images().List(false) }, "GET /v1.50/images/json"},
		{"images tag", func(b backend.Backend) { b.Images().Tag("a:1", "b:2") }, "POST /v1.50/images/a:1/tag"},
		{"images remove", func(b backend.Backend) { b.Images().Remove([]string{"a:1"}) }, "DELETE /v1.50/images/a:1"},
		{"images save", func(b backend.Backend) { b.Images().Save([]string{"a"}, filepath.Join(tmp, "a.tar")) }, "GET /v1.50/images/get"},
		{"images commit", func(b backend.Backend) { b.Images().Commit("web", "r:t", "", "", nil) }, "POST /v1.50/commit"},
		{"images inspect", func(b backend.Backend) { b.Images().Inspect("a:1") }, "GET /v1.50/images/a:1/json"},
		{"networks list", func(b backend.Backend) { nets(b).List(false) }, "GET /v1.50/networks"},
		{"networks add", func(b backend.Backend) { nets(b).Add("n") }, "POST /v1.50/networks/create"},
		{"networks remove", func(b backend.Backend) { nets(b).Remove([]string{"n"}) }, "GET /v1.50/networks"},
		{"networks attach", func(b backend.Backend) { nets(b).Attach("n", "c") }, "GET /v1.50/networks"},
		{"networks detach", func(b backend.Backend) { nets(b).Detach("n", "c") }, "GET /v1.50/networks"},
		{"networks inspect", func(b backend.Backend) { nets(b).Inspect("n") }, "GET /v1.50/networks/n"},
		{"volumes list", func(b backend.Backend) { vols(b).List(false) }, "GET /v1.50/volumes"},
		{"volumes remove", func(b backend.Backend) { vols(b).Remove([]string{"v"}) }, "DELETE /v1.50/volumes/v"},
		{"volumes prune", func(b backend.Backend) { vols(b).Prune() }, "GET /v1.50/volumes"},
		{"volumes create", func(b backend.Backend) { vols(b).Create("v") }, "POST /v1.50/volumes/create"},
		{"volumes inspect", func(b backend.Backend) { vols(b).Inspect("v") }, "GET /v1.50/volumes/v"},
		{"volumes backup", func(b backend.Backend) { vols(b).Backup("v", filepath.Join(tmp, "b.tar")) }, "GET /v1.50/containers/json"},
		// A missing archive is detected before any request
		{"volumes restore", func(b backend.Backend) { vols(b).Restore("v", filepath.Join(tmp, "missing.tar")) }, ""},
		{"system rms", func(b backend.Backend) { b.System().RmContainers() }, "GET /v1.50/containers/json"},
		{"system clean", func(b backend.Backend) { b.System().Clean() }, "GET /v1.50/images/json"},
		{"system info", func(b backend.Backend) { b.System().Info() }, "GET /v1.50/info"},
		{"cp", func(b backend.Backend) { fc, _ := b.FileCopier(); fc.Copy("web:/etc/hosts", tmp) }, "HEAD /v1.50/containers/web/archive"},
		{"logs", func(b backend.Backend) { r, _ := b.Runner(); r.Logs("web") }, "GET /v1.50/containers/web/logs"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b, d := newBackend(t)
			skip := len(d.Requests()) // the /version negotiation
			c.call(b)
			got := ""
			if seq := d.Sequence(); len(seq) > skip {
				got = seq[skip]
			}
			if got != c.first {
				t.Errorf("first request = %q; want %q", got, c.first)
			}
		})
	}
}
