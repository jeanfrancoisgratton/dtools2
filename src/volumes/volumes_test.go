// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/volumes/volumes_test.go

package volumes

import (
	"dtools2/extras"
	"dtools2/internal/testutil"
	"dtools2/rest"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// resetGlobals sets quiet mode and restores every package-level flag the functions under test touch
func resetGlobals(t *testing.T) {
	t.Helper()
	rmBL, force, named, driver, img, noClean := RemoveBlackListed, ForceRemove, RemoveNamedVolumes, CreateVolDriver, BackupImage, NoCleanup
	quiet, outJSON, outFile, outFmt := rest.QuietOutput, extras.OutputJSON, extras.OutputFile, extras.OutputFormat
	rest.QuietOutput = true
	t.Cleanup(func() {
		RemoveBlackListed, ForceRemove, RemoveNamedVolumes, CreateVolDriver, BackupImage, NoCleanup = rmBL, force, named, driver, img, noClean
		rest.QuietOutput, extras.OutputJSON, extras.OutputFile, extras.OutputFormat = quiet, outJSON, outFile, outFmt
	})
}

func writeBlacklist(t *testing.T, content string) {
	t.Helper()
	dir := testutil.IsolateHome(t)
	if err := os.WriteFile(filepath.Join(dir, "blacklist.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFetchVolumeListShapes(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		want   int
		err    bool
	}{
		{"object", http.StatusOK, `{"Volumes":[{"Name":"a"},{"Name":"b"}],"Warnings":[]}`, 2, false},
		{"array", http.StatusOK, `[{"Name":"a"}]`, 1, false},
		{"no content", http.StatusNoContent, ``, 0, false},
		{"empty body", http.StatusOK, `  `, 0, false},
		{"unsupported shape", http.StatusOK, `"nope"`, 0, true},
		{"bad json", http.StatusOK, `{"Volumes":[`, 0, true},
		{"http error", http.StatusInternalServerError, `{"message":"boom"}`, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resetGlobals(t)
			d := testutil.NewDaemon(t)
			d.Handle("GET /volumes", func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(c.status)
				w.Write([]byte(c.body))
			})
			vols, err := fetchVolumeList(d.Client())
			if (err != nil) != c.err {
				t.Fatalf("err = %v; want error=%v", err, c.err)
			}
			if len(vols) != c.want {
				t.Errorf("got %d volumes; want %d", len(vols), c.want)
			}
		})
	}
}

func TestListVolumesRefCount(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /volumes", http.StatusOK, map[string]any{"Volumes": []map[string]any{{"Name": "data"}, {"Name": "orphan"}}})
	d.Reply("GET /containers/json", http.StatusOK, []any{
		testutil.Container("aaa111", "web", "running", "data"),
		testutil.Container("bbb222", "api", "exited", "data"),
	})

	vols, err := ListVolumes(d.Client(), false)
	if err != nil {
		t.Fatalf("ListVolumes: %v", err)
	}
	byName := map[string]Volume{}
	for _, v := range vols {
		byName[v.Name] = v
	}
	if byName["data"].RefCount != 2 || byName["data"].UsedByStr != "api (exited)\nweb (running)" {
		t.Errorf("data = %+v; want RefCount 2 used by api and web", byName["data"])
	}
	if byName["orphan"].RefCount != 0 {
		t.Errorf("orphan RefCount = %d; want 0", byName["orphan"].RefCount)
	}
	// Stopped containers still reference volumes, so the list must include them
	if q := d.Calls(http.MethodGet, "/containers/json")[0].Query; q.Get("all") != "true" {
		t.Errorf("containers query = %v; want all=true", q)
	}
}

func TestListVolumesDisplay(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /volumes", http.StatusOK, map[string]any{"Volumes": []map[string]any{{"Name": "data", "Driver": "local"}}})
	d.Reply("GET /containers/json", http.StatusOK, []any{})

	rest.QuietOutput = false
	out := testutil.CaptureStdout(t, func() { _, _ = ListVolumes(d.Client(), true) })
	if !strings.Contains(out, "data") || !strings.Contains(out, "RefCount") {
		t.Errorf("table output %q: want the header and the volume", out)
	}

	extras.OutputFormat = "Name"
	out = testutil.CaptureStdout(t, func() { _, _ = ListVolumes(d.Client(), true) })
	if strings.TrimSpace(out) != "data" {
		t.Errorf("--format Name output = %q; want data", out)
	}
}

func TestCreateVolume(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Handle("POST /volumes/create", func(w http.ResponseWriter, r *http.Request) {
		var o VolumeCreateOptions
		_ = json.NewDecoder(r.Body).Decode(&o)
		if o.Name == "exists" {
			testutil.JSON(w, http.StatusNotModified, nil)
			return
		}
		if o.Name == "bad" {
			testutil.JSON(w, http.StatusInternalServerError, map[string]string{"message": "boom"})
			return
		}
		testutil.JSON(w, http.StatusCreated, map[string]string{"Name": o.Name})
	})

	CreateVolDriver = "nfs"
	if err := CreateVolume(d.Client(), "data"); err != nil {
		t.Fatalf("CreateVolume: %v", err)
	}
	var sent VolumeCreateOptions
	_ = json.Unmarshal(d.Calls(http.MethodPost, "/volumes/create")[0].Body, &sent)
	if sent.Name != "data" || sent.Driver != "nfs" {
		t.Errorf("payload = %+v; want Name=data Driver=nfs", sent)
	}

	rest.QuietOutput = false
	out := testutil.CaptureStdout(t, func() {
		if err := CreateVolume(d.Client(), "exists"); err != nil {
			t.Errorf("CreateVolume on 304: %v", err)
		}
	})
	if !strings.Contains(out, "already exists") {
		t.Errorf("output %q: want 'already exists'", out)
	}
	if err := CreateVolume(d.Client(), "bad"); err == nil {
		t.Error("CreateVolume on HTTP 500: want an error")
	}
}

func TestRemoveVolumes(t *testing.T) {
	resetGlobals(t)
	writeBlacklist(t, `{"Volumes":["keep"]}`)
	d := testutil.NewDaemon(t)
	d.Reply("DELETE /volumes/{name}", http.StatusNoContent, nil)

	ForceRemove = true
	if err := RemoveVolumes(d.Client(), []string{"data", "keep"}); err != nil {
		t.Fatalf("RemoveVolumes: %v", err)
	}
	del := d.Calls(http.MethodDelete, "/volumes/data")
	if len(del) != 1 || del[0].Query.Get("force") != "true" {
		t.Errorf("DELETE calls %+v; want one with force=true", del)
	}
	if len(d.Calls(http.MethodDelete, "/volumes/keep")) != 0 {
		t.Error("a blacklisted volume must not be removed without -B")
	}
}

func TestRemoveVolumesBlacklistedWithFlag(t *testing.T) {
	for _, quiet := range []bool{false, true} {
		resetGlobals(t)
		writeBlacklist(t, `{"Volumes":["keep"]}`)
		d := testutil.NewDaemon(t)
		d.Reply("DELETE /volumes/{name}", http.StatusNoContent, nil)

		RemoveBlackListed = true
		rest.QuietOutput = quiet
		_ = testutil.CaptureStdout(t, func() {
			if err := RemoveVolumes(d.Client(), []string{"keep"}); err != nil {
				t.Errorf("RemoveVolumes -B: %v", err)
			}
		})
		if len(d.Calls(http.MethodDelete, "/volumes/keep")) != 1 {
			t.Errorf("quiet=%v: -B should remove a blacklisted volume", quiet)
		}
	}
}

func TestRemoveVolumesError(t *testing.T) {
	resetGlobals(t)
	testutil.IsolateHome(t)
	d := testutil.NewDaemon(t)
	d.Reply("DELETE /volumes/{name}", http.StatusConflict, map[string]string{"message": "volume is in use"})
	if err := RemoveVolumes(d.Client(), []string{"data"}); err == nil {
		t.Error("RemoveVolumes on HTTP 409: want an error")
	}
}

func pruneDaemon(t *testing.T) *testutil.Daemon {
	d := testutil.NewDaemon(t)
	d.Reply("GET /volumes", http.StatusOK, map[string]any{"Volumes": []map[string]any{
		{"Name": "anon", "Driver": "local", "Scope": "local", "Labels": map[string]string{"com.docker.volume.anonymous": ""}},
		{"Name": "named", "Driver": "local", "Scope": "local"},
		{"Name": "used", "Driver": "local", "Scope": "local"},
		{"Name": "remote", "Driver": "nfs", "Scope": "local"},
		{"Name": "keep", "Driver": "local", "Scope": "local"},
	}})
	d.Reply("GET /containers/json", http.StatusOK, []any{testutil.Container("aaa111", "web", "exited", "used")})
	d.Reply("DELETE /volumes/{name}", http.StatusNoContent, nil)
	return d
}

func deleted(d *testutil.Daemon) []string {
	var out []string
	for _, r := range d.Requests() {
		if r.Method == http.MethodDelete {
			out = append(out, strings.TrimPrefix(r.Path, "/volumes/"))
		}
	}
	return out
}

func TestPruneVolumes(t *testing.T) {
	cases := []struct {
		named bool
		want  string
	}{
		{true, "anon,named"},
		{false, "anon"},
	}
	for _, c := range cases {
		resetGlobals(t)
		writeBlacklist(t, `{"Volumes":["keep"]}`)
		d := pruneDaemon(t)
		RemoveNamedVolumes = c.named
		if err := PruneVolumes(d.Client()); err != nil {
			t.Fatalf("PruneVolumes: %v", err)
		}
		if got := strings.Join(deleted(d), ","); got != c.want {
			t.Errorf("named=%v: pruned %q; want %q (in-use, non-local and blacklisted volumes are kept)", c.named, got, c.want)
		}
	}
}

func TestPruneVolumesBlacklistedWithFlag(t *testing.T) {
	resetGlobals(t)
	writeBlacklist(t, `{"Volumes":["keep"]}`)
	d := pruneDaemon(t)
	RemoveNamedVolumes, RemoveBlackListed = true, true
	if err := PruneVolumes(d.Client()); err != nil {
		t.Fatalf("PruneVolumes: %v", err)
	}
	if got := strings.Join(deleted(d), ","); got != "anon,named,keep" {
		t.Errorf("prune -B pruned %q; want anon,named,keep", got)
	}
}

func TestPruneVolumesRestoresQuietOnError(t *testing.T) {
	resetGlobals(t)
	testutil.IsolateHome(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /volumes", http.StatusOK, map[string]any{"Volumes": []map[string]any{{"Name": "anon", "Driver": "local", "Scope": "local", "Labels": map[string]string{"com.docker.volume.anonymous": ""}}}})
	d.Reply("GET /containers/json", http.StatusOK, []any{})
	d.Reply("DELETE /volumes/{name}", http.StatusInternalServerError, map[string]string{"message": "boom"})

	rest.QuietOutput = false
	if err := PruneVolumes(d.Client()); err == nil {
		t.Fatal("PruneVolumes on a failed removal: want an error")
	}
	if rest.QuietOutput {
		t.Error("PruneVolumes left quiet mode switched on after a failure")
	}
}

func TestInspectVolume(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /volumes/data", http.StatusOK, map[string]any{"Name": "data", "Driver": "local"})
	d.Reply("GET /volumes/ghost", http.StatusNotFound, map[string]string{"message": "no such volume"})

	var v Volume
	var err *ce.CustomError
	out := testutil.CaptureStdout(t, func() { v, err = InspectVolume(d.Client(), "data") })
	if err != nil || v.Name != "data" {
		t.Fatalf("InspectVolume = %+v, %v; want data", v, err)
	}
	if !strings.Contains(out, `"Name":"data"`) {
		t.Errorf("quiet output %q: want the raw JSON", out)
	}
	if _, err := InspectVolume(d.Client(), "ghost"); err == nil || !strings.Contains(err.Title, "404") {
		t.Errorf("InspectVolume(ghost) = %v; want an HTTP 404 error", err)
	}
}
