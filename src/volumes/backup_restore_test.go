// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/volumes/backup_restore_test.go

package volumes

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"dtools2/internal/testutil"
	"dtools2/rest"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/ulikunitz/xz"
)

// volumeTar returns a tar stream shaped like GET /containers/{id}/archive?path=/data
func volumeTar(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	files := []struct{ name, body string }{{"data/", ""}, {"data/.dot", "hidden"}, {"data/sub/f.txt", "hello"}}
	for _, f := range files {
		h := &tar.Header{Name: f.name, Mode: 0o644, Size: int64(len(f.body)), Typeflag: tar.TypeReg}
		if strings.HasSuffix(f.name, "/") {
			h.Typeflag, h.Mode = tar.TypeDir, 0o755
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(f.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// volEngine is a fake daemon covering every endpoint backup/restore use, healthy by default.
// Individual tests override routes to inject failures.
type volEngine struct {
	*testutil.Daemon
	tarball  []byte
	attached []any
}

func newVolEngine(t *testing.T, attached ...any) *volEngine {
	t.Helper()
	resetGlobals(t)
	testutil.IsolateHome(t)
	e := &volEngine{Daemon: testutil.NewDaemon(t), tarball: volumeTar(t), attached: attached}
	if e.attached == nil {
		e.attached = []any{}
	}
	e.Reply("GET /containers/json", http.StatusOK, e.attached)
	e.Reply("GET /images/json", http.StatusOK, []map[string]any{{"RepoTags": []string{"alpine:latest"}}})
	e.Reply("POST /images/create", http.StatusOK, map[string]string{"status": "pulled"})
	e.Reply("POST /containers/create", http.StatusCreated, map[string]string{"Id": "temp1234567890"})
	e.Reply("POST /containers/{id}/start", http.StatusNoContent, nil)
	e.Reply("POST /containers/{id}/stop", http.StatusNoContent, nil)
	e.Reply("POST /containers/{id}/wait", http.StatusOK, map[string]int{"StatusCode": 0})
	e.Reply("DELETE /containers/{id}", http.StatusNoContent, nil)
	e.Reply("DELETE /volumes/{name}", http.StatusNoContent, nil)
	e.Reply("POST /volumes/create", http.StatusCreated, map[string]string{"Name": "data"})
	e.Reply("PUT /containers/{id}/archive", http.StatusOK, nil)
	e.Handle("GET /containers/{id}/archive", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/x-tar")
		w.Write(e.tarball)
	})
	return e
}

func (e *volEngine) count(method, path string) int { return len(e.Calls(method, path)) }

// createdCmds returns the Cmd of every container created, in order
func (e *volEngine) createdCmds(t *testing.T) [][]string {
	t.Helper()
	var out [][]string
	for _, r := range e.Calls(http.MethodPost, "/containers/create") {
		var p struct {
			Cmd        []string
			HostConfig struct{ Binds []string }
		}
		if err := json.Unmarshal(r.Body, &p); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(p.HostConfig.Binds, []string{"data:/data"}) {
			t.Errorf("container binds = %v; want [data:/data]", p.HostConfig.Binds)
		}
		out = append(out, p.Cmd)
	}
	return out
}

func decompress(t *testing.T, path string) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var r io.Reader = f
	switch {
	case isGzipArchive(path):
		gz, err := gzip.NewReader(f)
		if err != nil {
			t.Fatalf("%s is not gzip: %v", path, err)
		}
		r = gz
	case isXzArchive(path):
		xr, err := xz.NewReader(f)
		if err != nil {
			t.Fatalf("%s is not xz: %v", path, err)
		}
		r = xr
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return b
}

func TestNormalizeArchiveName(t *testing.T) {
	cases := map[string]string{
		"backup":        "backup.tar",
		"backup.tar":    "backup.tar",
		"backup.tar.gz": "backup.tar.gz",
		"backup.tgz":    "backup.tgz",
		"backup.tar.xz": "backup.tar.xz",
		"backup.zip":    "backup.zip.tar",
	}
	for in, want := range cases {
		if got := normalizeArchiveName(in); got != want {
			t.Errorf("normalizeArchiveName(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestArchiveCompressionDetection(t *testing.T) {
	cases := []struct {
		name     string
		gzip, xz bool
	}{
		{"a.tar", false, false},
		{"a.tar.gz", true, false},
		{"a.tgz", true, false},
		{"a.gz", true, false},
		{"a.tar.xz", false, true},
		{"a.txz", false, true},
	}
	for _, c := range cases {
		if isGzipArchive(c.name) != c.gzip || isXzArchive(c.name) != c.xz {
			t.Errorf("%s: gzip=%v xz=%v; want gzip=%v xz=%v", c.name, isGzipArchive(c.name), isXzArchive(c.name), c.gzip, c.xz)
		}
	}
}

func TestCheckStatus(t *testing.T) {
	cases := []struct {
		status int
		body   string
		ok     []int
		want   string
	}{
		{204, "", []int{204, 304}, ""},
		{304, "", []int{204, 304}, ""},
		{409, `{"message":"volume is in use"}`, []int{204}, "HTTP 409 Conflict: volume is in use"},
		{500, "plain text", []int{200}, "HTTP 500 Internal Server Error: plain text"},
		{404, "", []int{200}, "HTTP 404 Not Found"},
	}
	for _, c := range cases {
		resp := &http.Response{
			StatusCode: c.status,
			Status:     strconv.Itoa(c.status) + " " + http.StatusText(c.status),
			Body:       io.NopCloser(strings.NewReader(c.body)),
		}
		err := checkStatus(resp, c.ok...)
		got := ""
		if err != nil {
			got = err.Error()
		}
		if got != c.want {
			t.Errorf("checkStatus(%d, %q) = %q; want %q", c.status, c.body, got, c.want)
		}
	}
}

func TestShortID(t *testing.T) {
	if got := shortID("0123456789abcdef"); got != "0123456789ab" {
		t.Errorf("shortID(long) = %q", got)
	}
	if got := shortID("abc"); got != "abc" {
		t.Errorf("shortID(short) = %q", got)
	}
}

func TestEnsureImageExists(t *testing.T) {
	e := newVolEngine(t)
	if err := ensureImageExists(e.Client(), "alpine:latest"); err != nil {
		t.Fatalf("ensureImageExists(present): %v", err)
	}
	if e.count(http.MethodPost, "/images/create") != 0 {
		t.Error("a present image must not be pulled")
	}

	if err := ensureImageExists(e.Client(), "busybox:1.36"); err != nil {
		t.Fatalf("ensureImageExists(absent): %v", err)
	}
	pulls := e.Calls(http.MethodPost, "/images/create")
	if len(pulls) != 1 || pulls[0].Query.Get("fromImage") != "busybox" || pulls[0].Query.Get("tag") != "1.36" {
		t.Errorf("pull calls %+v; want fromImage=busybox tag=1.36", pulls)
	}
}

func TestGetContainersUsingVolume(t *testing.T) {
	e := newVolEngine(t,
		testutil.Container("aaa111", "web", "running", "data"),
		testutil.Container("bbb222", "old", "exited", "data"),
		testutil.Container("ccc333", "other", "running", "elsewhere"),
	)
	ids, err := getContainersUsingVolume(e.Client(), "data")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"aaa111"}) {
		t.Errorf("got %v; want only the running container using the volume", ids)
	}
}

func TestBackupVolume(t *testing.T) {
	for _, name := range []string{"backup.tar", "backup.tar.gz", "backup.tar.xz"} {
		t.Run(name, func(t *testing.T) {
			e := newVolEngine(t, testutil.Container("aaa111", "web", "running", "data"))
			archive := filepath.Join(t.TempDir(), name)

			if err := BackupVolume(e.Client(), "data", archive); err != nil {
				t.Fatalf("BackupVolume: %v", err)
			}
			if got := decompress(t, archive); !bytes.Equal(got, e.tarball) {
				t.Errorf("archive content differs from the daemon's tar stream (%d vs %d bytes)", len(got), len(e.tarball))
			}
			if q := e.Calls(http.MethodGet, "/containers/temp1234567890/archive")[0].Query; q.Get("path") != "/data" {
				t.Errorf("archive query = %v; want path=/data", q)
			}
			want := []string{
				"GET /containers/json",
				"POST /containers/aaa111/stop",
				"GET /images/json",
				"POST /containers/create",
				"POST /containers/temp1234567890/start",
				"GET /containers/temp1234567890/archive",
				"POST /containers/temp1234567890/stop",
				"DELETE /containers/temp1234567890",
				"POST /containers/aaa111/start",
			}
			if got := e.Sequence(); !reflect.DeepEqual(got, want) {
				t.Errorf("requests:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
			if cmds := e.createdCmds(t); len(cmds) != 1 || !slices.Contains(cmds[0], "sleep infinity") {
				t.Errorf("temp container cmd = %v; want sleep infinity", cmds)
			}
		})
	}
}

func TestBackupVolumeAppendsTarSuffix(t *testing.T) {
	e := newVolEngine(t)
	base := filepath.Join(t.TempDir(), "backup")
	if err := BackupVolume(e.Client(), "data", base); err != nil {
		t.Fatalf("BackupVolume: %v", err)
	}
	if _, err := os.Stat(base + ".tar"); err != nil {
		t.Errorf("want %s.tar to be created: %v", base, err)
	}
}

func TestBackupVolumeNoCleanup(t *testing.T) {
	e := newVolEngine(t)
	NoCleanup = true
	if err := BackupVolume(e.Client(), "data", filepath.Join(t.TempDir(), "b.tar")); err != nil {
		t.Fatalf("BackupVolume: %v", err)
	}
	if e.count(http.MethodDelete, "/containers/temp1234567890") != 0 {
		t.Error("--no-cleanup: the temp container must be kept")
	}
}

func TestBackupVolumeArchiveFailure(t *testing.T) {
	e := newVolEngine(t, testutil.Container("aaa111", "web", "running", "data"))
	e.Reply("GET /containers/{id}/archive", http.StatusInternalServerError, map[string]string{"message": "boom"})
	archive := filepath.Join(t.TempDir(), "b.tar.gz")

	err := BackupVolume(e.Client(), "data", archive)
	if err == nil || !strings.Contains(err.Message, "boom") {
		t.Fatalf("BackupVolume = %v; want an error carrying the daemon message", err)
	}
	if _, serr := os.Stat(archive); !errors.Is(serr, os.ErrNotExist) {
		t.Error("a partial archive must not be left behind")
	}
	if e.count(http.MethodDelete, "/containers/temp1234567890") != 1 {
		t.Error("the temp container must be removed on failure")
	}
	if e.count(http.MethodPost, "/containers/aaa111/start") != 1 {
		t.Error("stopped containers must be restarted after a failed backup")
	}
}

func TestBackupVolumeCreateFailure(t *testing.T) {
	e := newVolEngine(t, testutil.Container("aaa111", "web", "running", "data"))
	e.Reply("POST /containers/create", http.StatusConflict, map[string]string{"message": "name in use"})
	if err := BackupVolume(e.Client(), "data", filepath.Join(t.TempDir(), "b.tar")); err == nil {
		t.Fatal("BackupVolume with a failing container create: want an error")
	}
	if e.count(http.MethodPost, "/containers/aaa111/start") != 1 {
		t.Error("stopped containers must be restarted after a failed backup")
	}
}

func TestBackupVolumePartialStopFailure(t *testing.T) {
	e := newVolEngine(t,
		testutil.Container("aaa111", "web", "running", "data"),
		testutil.Container("bbb222", "api", "running", "data"),
	)
	e.Reply("POST /containers/bbb222/stop", http.StatusInternalServerError, map[string]string{"message": "boom"})
	if err := BackupVolume(e.Client(), "data", filepath.Join(t.TempDir(), "b.tar")); err == nil {
		t.Fatal("BackupVolume with a failing stop: want an error")
	}
	if e.count(http.MethodPost, "/containers/aaa111/start") != 1 {
		t.Error("the container stopped before the failure must be restarted")
	}
	if e.count(http.MethodPost, "/containers/create") != 0 {
		t.Error("no temp container should be created after a failed stop")
	}
}

// writeArchive writes the volume tarball to dir/name, compressed as its suffix says
func writeArchive(t *testing.T, e *volEngine, name string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	var w io.WriteCloser = f
	switch {
	case isGzipArchive(name):
		w = gzip.NewWriter(f)
	case isXzArchive(name):
		if w, err = xz.NewWriter(f); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.Write(e.tarball); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if w != io.WriteCloser(f) {
		f.Close()
	}
	return path
}

func TestRestoreVolume(t *testing.T) {
	for _, name := range []string{"r.tar", "r.tar.gz", "r.tar.xz"} {
		t.Run(name, func(t *testing.T) {
			e := newVolEngine(t, testutil.Container("aaa111", "web", "running", "data"))
			// The volume is in use by the (stopped) container, so the daemon refuses its deletion
			e.Reply("DELETE /volumes/{name}", http.StatusConflict, map[string]string{"message": "volume is in use"})
			archive := writeArchive(t, e, name)

			if err := RestoreVolume(e.Client(), "data", archive); err != nil {
				t.Fatalf("RestoreVolume: %v", err)
			}
			put := e.Calls(http.MethodPut, "/containers/temp1234567890/archive")
			if len(put) != 1 {
				t.Fatalf("requests %v: want one PUT archive", e.Sequence())
			}
			if put[0].Query.Get("path") != "/" || put[0].Header.Get("Content-Type") != "application/x-tar" {
				t.Errorf("PUT query %v, content-type %q; want path=/ and application/x-tar", put[0].Query, put[0].Header.Get("Content-Type"))
			}
			if !bytes.Equal(put[0].Body, e.tarball) {
				t.Errorf("uploaded %d bytes; want the decompressed %d-byte tar stream", len(put[0].Body), len(e.tarball))
			}
			cmds := e.createdCmds(t)
			if len(cmds) != 2 || !strings.Contains(strings.Join(cmds[0], " "), "find /data -mindepth 1 -delete") {
				t.Errorf("created containers %v; want a wipe container then the temp container", cmds)
			}
			if e.count(http.MethodPost, "/volumes/create") != 0 {
				t.Error("a wiped volume must not be re-created")
			}
			if e.count(http.MethodPost, "/containers/aaa111/start") != 1 {
				t.Error("the attached container must be restarted")
			}
		})
	}
}

func TestRestoreVolumeDeletesAndRecreates(t *testing.T) {
	e := newVolEngine(t)
	if err := RestoreVolume(e.Client(), "data", writeArchive(t, e, "r.tar")); err != nil {
		t.Fatalf("RestoreVolume: %v", err)
	}
	want := []string{
		"GET /images/json",
		"GET /containers/json",
		"DELETE /volumes/data",
		"POST /volumes/create",
		"POST /containers/create",
		"POST /containers/temp1234567890/start",
		"PUT /containers/temp1234567890/archive",
		"POST /containers/temp1234567890/stop",
		"DELETE /containers/temp1234567890",
	}
	if got := e.Sequence(); !reflect.DeepEqual(got, want) {
		t.Errorf("requests:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestRestoreVolumeMissingArchive(t *testing.T) {
	e := newVolEngine(t, testutil.Container("aaa111", "web", "running", "data"))
	if err := RestoreVolume(e.Client(), "data", filepath.Join(t.TempDir(), "nope.tar.gz")); err == nil {
		t.Fatal("RestoreVolume with a missing archive: want an error")
	}
	// Nothing at all may happen: no stop, and above all no volume deletion
	if n := len(e.Requests()); n != 0 {
		t.Errorf("requests %v: want none before the archive is known to be readable", e.Sequence())
	}
}

func TestRestoreVolumeFailureAfterReset(t *testing.T) {
	e := newVolEngine(t, testutil.Container("aaa111", "web", "running", "data"))
	e.Reply("PUT /containers/{id}/archive", http.StatusInternalServerError, map[string]string{"message": "disk full"})
	rest.QuietOutput = false

	var out string
	err := func() error {
		var cerr error
		out = testutil.CaptureStdout(t, func() {
			if e := RestoreVolume(e.Client(), "data", writeArchive(t, e, "r.tar")); e != nil {
				cerr = e
			}
		})
		return cerr
	}()
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("RestoreVolume = %v; want an error carrying the daemon message", err)
	}
	if e.count(http.MethodPost, "/containers/aaa111/start") != 0 {
		t.Error("containers must be left stopped once the volume has been reset")
	}
	if !strings.Contains(out, "left stopped") || !strings.Contains(out, "aaa111") {
		t.Errorf("output %q: want a warning listing the containers left stopped", out)
	}
	if e.count(http.MethodDelete, "/containers/temp1234567890") != 1 {
		t.Error("the temp container must be removed on failure")
	}
}

func TestRestoreVolumeFailureBeforeReset(t *testing.T) {
	e := newVolEngine(t,
		testutil.Container("aaa111", "web", "running", "data"),
		testutil.Container("bbb222", "api", "running", "data"),
	)
	e.Reply("POST /containers/bbb222/stop", http.StatusInternalServerError, map[string]string{"message": "boom"})
	if err := RestoreVolume(e.Client(), "data", writeArchive(t, e, "r.tar")); err == nil {
		t.Fatal("RestoreVolume with a failing stop: want an error")
	}
	if e.count(http.MethodPost, "/containers/aaa111/start") != 1 {
		t.Error("nothing was altered yet: the stopped container must be restarted")
	}
	if e.count(http.MethodDelete, "/volumes/data") != 0 {
		t.Error("the volume must not be touched after a failed stop")
	}
}

func TestRestoreVolumeDeleteError(t *testing.T) {
	e := newVolEngine(t)
	e.Reply("DELETE /volumes/{name}", http.StatusInternalServerError, map[string]string{"message": "boom"})
	if err := RestoreVolume(e.Client(), "data", writeArchive(t, e, "r.tar")); err == nil {
		t.Fatal("RestoreVolume with a failing volume deletion: want an error")
	}
	if e.count(http.MethodPost, "/containers/create") != 0 {
		t.Error("no temp container should be created after a failed volume deletion")
	}
}

func TestRestoreVolumeWipeFailure(t *testing.T) {
	e := newVolEngine(t)
	e.Reply("DELETE /volumes/{name}", http.StatusConflict, map[string]string{"message": "volume is in use"})
	e.Reply("POST /containers/{id}/wait", http.StatusOK, map[string]int{"StatusCode": 1})
	err := RestoreVolume(e.Client(), "data", writeArchive(t, e, "r.tar"))
	if err == nil || !strings.Contains(err.Message, "exited with code 1") {
		t.Fatalf("RestoreVolume = %v; want a wipe failure", err)
	}
	if e.count(http.MethodDelete, "/containers/temp1234567890") != 1 {
		t.Error("the wipe container must always be removed")
	}
	if e.count(http.MethodPut, "/containers/temp1234567890/archive") != 0 {
		t.Error("nothing should be uploaded after a failed wipe")
	}
}

func TestRestoreVolumeCorruptArchive(t *testing.T) {
	e := newVolEngine(t)
	path := filepath.Join(t.TempDir(), "r.tar.gz")
	if err := os.WriteFile(path, []byte("this is not gzip"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RestoreVolume(e.Client(), "data", path); err == nil || !strings.Contains(err.Title, "gzip") {
		t.Fatalf("RestoreVolume = %v; want a gzip reader error", err)
	}
}

func TestRestoreVolumeBlacklisted(t *testing.T) {
	e := newVolEngine(t)
	writeBlacklist(t, `{"Volumes":["data"]}`)
	rest.QuietOutput = false
	out := testutil.CaptureStdout(t, func() {
		if err := RestoreVolume(e.Client(), "data", writeArchive(t, e, "r.tar")); err != nil {
			t.Errorf("RestoreVolume: %v", err)
		}
	})
	if !strings.Contains(out, "blacklisted") {
		t.Errorf("output %q: want a blacklist warning", out)
	}
	if e.count(http.MethodPut, "/containers/temp1234567890/archive") != 1 {
		t.Error("a blacklisted volume must still be restored")
	}
}

func TestRestoreVolumeCustomImage(t *testing.T) {
	e := newVolEngine(t)
	BackupImage = "busybox:1.36"
	if err := RestoreVolume(e.Client(), "data", writeArchive(t, e, "r.tar")); err != nil {
		t.Fatalf("RestoreVolume: %v", err)
	}
	if e.count(http.MethodPost, "/images/create") != 1 {
		t.Error("an absent custom image must be pulled")
	}
	var p struct{ Image string }
	_ = json.Unmarshal(e.Calls(http.MethodPost, "/containers/create")[0].Body, &p)
	if p.Image != "busybox:1.36" {
		t.Errorf("temp container image = %q; want busybox:1.36", p.Image)
	}
}
