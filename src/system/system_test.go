// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/system/system_test.go

package system

import (
	"archive/tar"
	"bytes"
	"context"
	"dtools2/containers"
	"dtools2/env"
	"dtools2/images"
	"dtools2/internal/testutil"
	"dtools2/networks"
	"dtools2/rest"
	"dtools2/volumes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func resetGlobals(t *testing.T) {
	t.Helper()
	testutil.IsolateHome(t)
	quiet, ctx, jsonOut, regFile := rest.QuietOutput, rest.Context, JSONoutputfile, env.RegConfigFile
	force, rmVols, rmBL := ForceRemove, RemoveUnamedVolumes, RemoveBlacklisted
	cForce, cVols, cBL, cOnly := containers.ForceRemoveContainer, containers.RemoveUnamedVolumes, containers.RemoveBlacklisted, containers.OnlyRunningContainers
	iForce, iBL, vForce, vBL, vNamed, nBL := images.ForceRemove, images.RemoveBlacklisted, volumes.ForceRemove, volumes.RemoveBlackListed, volumes.RemoveNamedVolumes, networks.RemoveBlacklisted
	rest.QuietOutput, rest.Context = true, context.Background()
	t.Cleanup(func() {
		rest.QuietOutput, rest.Context, JSONoutputfile, env.RegConfigFile = quiet, ctx, jsonOut, regFile
		ForceRemove, RemoveUnamedVolumes, RemoveBlacklisted = force, rmVols, rmBL
		containers.ForceRemoveContainer, containers.RemoveUnamedVolumes, containers.RemoveBlacklisted, containers.OnlyRunningContainers = cForce, cVols, cBL, cOnly
		images.ForceRemove, images.RemoveBlacklisted, volumes.ForceRemove, volumes.RemoveBlackListed, volumes.RemoveNamedVolumes, networks.RemoveBlacklisted = iForce, iBL, vForce, vBL, vNamed, nBL
	})
}

type tarEntry struct {
	name, body, link string
	typ              byte
}

func makeTar(t *testing.T, entries ...tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		h := &tar.Header{Name: e.name, Typeflag: e.typ, Mode: 0o644, Size: int64(len(e.body)), Linkname: e.link}
		switch e.typ {
		case tar.TypeDir:
			h.Mode = 0o755
		case tar.TypeSymlink, tar.TypeLink:
			h.Size = 0
		}
		if err := tw.WriteHeader(h); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(e.body))
	}
	tw.Close()
	return buf.Bytes()
}

func readTar(t *testing.T, b []byte) map[string]string {
	t.Helper()
	out := map[string]string{}
	tr := tar.NewReader(bytes.NewReader(b))
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return out
		}
		if err != nil {
			t.Fatalf("reading uploaded tar: %v", err)
		}
		body, _ := io.ReadAll(tr)
		out[h.Name] = string(body)
	}
}

func statHeader(name string, isDir bool) string {
	mode := int64(0o644)
	if isDir {
		mode = int64(os.ModeDir | 0o755)
	}
	b, _ := json.Marshal(containerPathStat{Name: name, Mode: mode})
	return base64.StdEncoding.EncodeToString(b)
}

// cpDaemon serves container "web": /etc/app.conf is a file, /data a directory, anything else is missing
func cpDaemon(t *testing.T, archive []byte) *testutil.Daemon {
	t.Helper()
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Handle("HEAD /containers/web/archive", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("path") {
		case "/etc/app.conf":
			w.Header().Set("X-Docker-Container-Path-Stat", statHeader("app.conf", false))
		case "/data", "/tmp":
			w.Header().Set("X-Docker-Container-Path-Stat", statHeader("data", true))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	d.Handle("GET /containers/web/archive", func(w http.ResponseWriter, _ *http.Request) { w.Write(archive) })
	d.Reply("PUT /containers/web/archive", http.StatusOK, nil)
	return d
}

func TestCopyFromFileRename(t *testing.T) {
	d := cpDaemon(t, makeTar(t, tarEntry{name: "app.conf", body: "key=value", typ: tar.TypeReg}))
	dest := filepath.Join(t.TempDir(), "copy.conf")

	if err := CopyFile(d.Client(), "web:/etc/app.conf", dest); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	if b, _ := os.ReadFile(dest); string(b) != "key=value" {
		t.Errorf("dest content = %q", b)
	}
	if q := d.Calls(http.MethodGet, "/containers/web/archive")[0].Query; q.Get("path") != "/etc/app.conf" {
		t.Errorf("archive query = %v", q)
	}
}

func TestCopyFromFileIntoDir(t *testing.T) {
	d := cpDaemon(t, makeTar(t, tarEntry{name: "app.conf", body: "x", typ: tar.TypeReg}))
	dir := t.TempDir()
	if err := CopyFile(d.Client(), "web:/etc/app.conf", dir+"/"); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "app.conf")); string(b) != "x" {
		t.Errorf("dir/app.conf = %q", b)
	}
}

func TestCopyFromDir(t *testing.T) {
	archive := makeTar(t,
		tarEntry{name: "./", typ: tar.TypeDir},
		tarEntry{name: "sub/", typ: tar.TypeDir},
		tarEntry{name: "sub/f.txt", body: "hello", typ: tar.TypeReg},
		tarEntry{name: "link", link: "sub/f.txt", typ: tar.TypeSymlink},
		tarEntry{name: "hard", link: "sub/f.txt", typ: tar.TypeLink},
	)

	// Into a new directory: the daemon is asked for the contents only (path/.)
	d := cpDaemon(t, archive)
	dest := filepath.Join(t.TempDir(), "newdir")
	if err := CopyFile(d.Client(), "web:/data", dest); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	if q := d.Calls(http.MethodGet, "/containers/web/archive")[0].Query; q.Get("path") != "/data/." {
		t.Errorf("archive path = %q; want /data/. for a new destination", q.Get("path"))
	}
	for _, f := range []string{"sub/f.txt", "link", "hard"} {
		if b, err := os.ReadFile(filepath.Join(dest, f)); err != nil || string(b) != "hello" {
			t.Errorf("%s = %q, %v", f, b, err)
		}
	}

	// Into an existing directory: the directory itself is requested
	d = cpDaemon(t, archive)
	existing := t.TempDir()
	if err := CopyFile(d.Client(), "web:/data", existing); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	if q := d.Calls(http.MethodGet, "/containers/web/archive")[0].Query; q.Get("path") != "/data" {
		t.Errorf("archive path = %q; want /data for an existing destination", q.Get("path"))
	}
}

func TestCopyFromErrors(t *testing.T) {
	d := cpDaemon(t, nil)
	tmp := t.TempDir()
	file := filepath.Join(tmp, "file")
	os.WriteFile(file, nil, 0o644)

	cases := []struct {
		name, src, dest, want string
	}{
		{"missing path", "web:/nope", filepath.Join(tmp, "x"), "not found"},
		{"empty destination", "web:/etc/app.conf", " ", "destination path is empty"},
		{"dir onto file", "web:/data", file, "directory onto an existing file"},
		{"trailing slash on file", "web:/etc/app.conf", file + "/", "existing file"},
		{"bad source", ":/x", tmp, "Invalid source"},
	}
	for _, c := range cases {
		err := CopyFile(d.Client(), c.src, c.dest)
		if err == nil || !strings.Contains(err.Title+" "+err.Message, c.want) {
			t.Errorf("%s: CopyFile = %v; want it to mention %q", c.name, err, c.want)
		}
	}

	d.Reply("GET /containers/web/archive", http.StatusInternalServerError, map[string]string{"message": "disk on fire"})
	if err := CopyFile(d.Client(), "web:/etc/app.conf", filepath.Join(tmp, "y")); err == nil || !strings.Contains(err.Message, "disk on fire") {
		t.Errorf("CopyFile on HTTP 500 = %v; want the daemon message", err)
	}
}

func TestCopyTo(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("A"), 0o644)
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("B"), 0o644)
	srcName := filepath.Base(src)

	cases := []struct {
		name, from, to, wantDir string
		wantEntries             map[string]string
	}{
		{"file into dir", filepath.Join(src, "a.txt"), "web:/tmp", "/tmp", map[string]string{"a.txt": "A"}},
		{"file renamed", filepath.Join(src, "a.txt"), "web:/etc/new.conf", "/etc", map[string]string{"new.conf": "A"}},
		{"dir into dir", src, "web:/data/", "/data", map[string]string{srcName + "/": "", srcName + "/a.txt": "A", srcName + "/sub/": "", srcName + "/sub/b.txt": "B"}},
		{"dir contents", src + "/.", "web:/data", "/data", map[string]string{"a.txt": "A", "sub/": "", "sub/b.txt": "B"}},
		{"dir renamed", src, "web:/srv/app", "/srv", map[string]string{"app/": "", "app/a.txt": "A", "app/sub/": "", "app/sub/b.txt": "B"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := cpDaemon(t, nil)
			if err := CopyFile(d.Client(), c.from, c.to); err != nil {
				t.Fatalf("CopyFile: %v", err)
			}
			put := d.Calls(http.MethodPut, "/containers/web/archive")
			if len(put) != 1 || put[0].Query.Get("path") != c.wantDir {
				t.Fatalf("PUT calls %+v; want one to %s", put, c.wantDir)
			}
			if got := readTar(t, put[0].Body); !reflect.DeepEqual(got, c.wantEntries) {
				t.Errorf("uploaded entries = %v; want %v", got, c.wantEntries)
			}
		})
	}
}

func TestCopyToErrors(t *testing.T) {
	d := cpDaemon(t, nil)
	src := t.TempDir()
	if err := CopyFile(d.Client(), filepath.Join(src, "missing"), "web:/tmp"); err == nil {
		t.Error("a missing source: want an error")
	}
	if err := CopyFile(d.Client(), src, "web:/etc/app.conf"); err == nil {
		t.Error("a directory onto a container file: want an error")
	}
	if err := CopyFile(d.Client(), src, "web:"); err == nil {
		t.Error("an empty container path: want an error")
	}
	d.Reply("PUT /containers/web/archive", http.StatusForbidden, map[string]string{"message": "read-only file system"})
	if err := CopyFile(d.Client(), src, "web:/data/"); err == nil || !strings.Contains(err.Message, "read-only") {
		t.Errorf("CopyFile on HTTP 403 = %v; want the daemon message", err)
	}
}

// A tar entry must never be written outside the destination, including through a symlink
// planted by an earlier entry of the same archive
func TestExtractTarDirSymlinkEscape(t *testing.T) {
	base := t.TempDir()
	dest := filepath.Join(base, "dest")
	outside := filepath.Join(base, "outside")
	os.MkdirAll(outside, 0o755)

	archive := makeTar(t,
		tarEntry{name: "evil", link: outside, typ: tar.TypeSymlink},
		tarEntry{name: "evil/pwned", body: "owned", typ: tar.TypeReg},
	)
	_ = extractTarDir(bytes.NewReader(archive), dest)
	if _, err := os.Stat(filepath.Join(outside, "pwned")); err == nil {
		t.Error("extractTarDir wrote a file outside the destination through a symlink")
	}
}

func TestExtractTarDirTraversal(t *testing.T) {
	base := t.TempDir()
	dest := filepath.Join(base, "dest")
	archive := makeTar(t, tarEntry{name: "../escape.txt", body: "x", typ: tar.TypeReg})
	_ = extractTarDir(bytes.NewReader(archive), dest)
	if _, err := os.Stat(filepath.Join(base, "escape.txt")); err == nil {
		t.Error("a ../ entry escaped the destination")
	}

	archive = makeTar(t, tarEntry{name: "h", link: "../../etc/passwd", typ: tar.TypeLink})
	if err := extractTarDir(bytes.NewReader(archive), dest); err == nil {
		t.Error("a hardlink escaping the destination: want an error")
	}
}

func TestExtractSingleFile(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out", "file")
	if err := extractSingleFile(bytes.NewReader(makeTar(t, tarEntry{name: "d/", typ: tar.TypeDir}, tarEntry{name: "d/f", body: "v", typ: tar.TypeReg})), dest); err != nil {
		t.Fatalf("extractSingleFile: %v", err)
	}
	if b, _ := os.ReadFile(dest); string(b) != "v" {
		t.Errorf("content = %q", b)
	}
	if err := extractSingleFile(bytes.NewReader(makeTar(t, tarEntry{name: "d/", typ: tar.TypeDir})), dest); err == nil {
		t.Error("no regular file: want an error")
	}
}

func TestStatContainerPath(t *testing.T) {
	d := cpDaemon(t, nil)
	st, ok, err := statContainerPath(d.Client(), "web", "/data")
	if err != nil || !ok || os.FileMode(st.Mode)&os.ModeDir == 0 {
		t.Errorf("statContainerPath(/data) = %+v, %v, %v; want an existing directory", st, ok, err)
	}
	if _, ok, err := statContainerPath(d.Client(), "web", "/nope"); ok || err != nil {
		t.Errorf("statContainerPath(missing) = %v, %v; want not found, no error", ok, err)
	}

	d.Handle("HEAD /containers/web/archive", func(w http.ResponseWriter, _ *http.Request) {})
	if _, _, err := statContainerPath(d.Client(), "web", "/x"); err == nil {
		t.Error("a missing stat header: want an error")
	}
	d.Handle("HEAD /containers/web/archive", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Docker-Container-Path-Stat", "%%%")
	})
	if _, _, err := statContainerPath(d.Client(), "web", "/x"); err == nil {
		t.Error("an undecodable stat header: want an error")
	}
	d.Handle("HEAD /containers/web/archive", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) })
	if _, _, err := statContainerPath(d.Client(), "web", "/x"); err == nil {
		t.Error("HTTP 500: want an error")
	}
}

func TestDockerMsgAndHostExists(t *testing.T) {
	if got := dockerMsg(strings.NewReader(`{"message":" boom "}`)); got != ": boom" {
		t.Errorf("dockerMsg(json) = %q", got)
	}
	if got := dockerMsg(strings.NewReader("plain")); got != ": plain" {
		t.Errorf("dockerMsg(plain) = %q", got)
	}
	if got := dockerMsg(strings.NewReader("  ")); got != "" {
		t.Errorf("dockerMsg(empty) = %q", got)
	}

	dir := t.TempDir()
	if ok, isDir, err := hostExistsIsDir(dir + "/"); !ok || !isDir || err != nil {
		t.Errorf("hostExistsIsDir(dir/) = %v, %v, %v", ok, isDir, err)
	}
	if ok, _, err := hostExistsIsDir(filepath.Join(dir, "nope")); ok || err != nil {
		t.Errorf("hostExistsIsDir(missing) = %v, %v", ok, err)
	}
}

func TestMakeTarStreamCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err := makeTarStream(ctx, t.TempDir(), "x", true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(r); err == nil {
		t.Error("a cancelled context must fail the tar stream")
	}
}

func TestRmContainers(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /containers/json", http.StatusOK, []any{
		testutil.Container("aaa111", "running1", "running"),
		testutil.Container("bbb222", "exited1", "exited"),
		testutil.Container("ccc333", "created1", "created"),
		testutil.Container("ddd444", "paused1", "paused"),
	})
	d.Reply("DELETE /containers/{id}", http.StatusNoContent, nil)

	ForceRemove, RemoveUnamedVolumes = true, false
	if err := RmContainers(d.Client()); err != nil {
		t.Fatalf("RmContainers: %v", err)
	}
	var removed []string
	for _, r := range d.Requests() {
		if r.Method == http.MethodDelete {
			removed = append(removed, strings.TrimPrefix(r.Path, "/containers/"))
			if r.Query.Get("force") != "true" || r.Query.Get("v") != "false" {
				t.Errorf("DELETE query = %v; want force=true v=false", r.Query)
			}
		}
	}
	sort.Strings(removed)
	if !reflect.DeepEqual(removed, []string{"bbb222", "ccc333"}) {
		t.Errorf("removed %v; want only the exited and created containers", removed)
	}
}

func TestClean(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /images/json", http.StatusOK, []map[string]any{
		{"Id": "sha256:used", "RepoTags": []string{"nginx:latest"}, "Containers": 1},
		{"Id": "sha256:unused", "RepoTags": []string{"alpine:3.20", "alpine:latest"}, "Containers": 0},
		{"Id": "sha256:dangling00", "RepoTags": []string{"<none>:<none>"}, "Containers": 0},
	})
	d.Reply("GET /volumes", http.StatusOK, map[string]any{"Volumes": []any{}})
	d.Reply("GET /containers/json", http.StatusOK, []any{})
	d.Reply("GET /networks", http.StatusOK, []map[string]any{
		{"Name": "bridge", "Id": "n1"}, {"Name": "host", "Id": "n2"}, {"Name": "none", "Id": "n3"}, {"Name": "stale", "Id": "n4"},
	})
	d.Reply("DELETE /images/{name...}", http.StatusOK, []any{})
	d.Reply("DELETE /networks/{id}", http.StatusNoContent, nil)

	_ = testutil.CaptureStdout(t, func() {
		if err := Clean(d.Client()); err != nil {
			t.Fatalf("Clean: %v", err)
		}
	})
	var removed []string
	for _, r := range d.Requests() {
		if r.Method == http.MethodDelete {
			removed = append(removed, r.Path)
		}
	}
	sort.Strings(removed)
	want := []string{"/images/alpine:3.20", "/images/alpine:latest", "/images/dangling00", "/networks/n4"}
	if !reflect.DeepEqual(removed, want) {
		t.Errorf("removed %v; want %v (by tag, ID for dangling, no used image, no built-in network)", removed, want)
	}
}

func TestInfo(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /info", http.StatusOK, map[string]any{
		"Containers": 4, "ContainersRunning": 2, "Images": 7, "ServerVersion": "29.7.2", "KernelVersion": "7.2.11",
		"OperatingSystem": "openSUSE Tumbleweed", "NCPU": 16, "MemTotal": 34359738368, "Name": "host1",
		"Swarm": map[string]any{"LocalNodeState": "inactive"},
	})
	out := testutil.CaptureStdout(t, func() {
		if err := Info(d.Client()); err != nil {
			t.Errorf("Info: %v", err)
		}
	})
	for _, want := range []string{"29.7.2", "7.2.11", "openSUSE Tumbleweed", "16", "32.00 GiB", "host1"} {
		if !strings.Contains(out, want) {
			t.Errorf("Info output missing %q:\n%s", want, out)
		}
	}

	// An unexpected schema falls back to printing the raw JSON
	d.Handle("GET /info", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, `{"Containers":"not a number"}`) })
	out = testutil.CaptureStdout(t, func() { _ = Info(d.Client()) })
	if !strings.Contains(out, "not a number") {
		t.Errorf("fallback output = %q; want the raw payload", out)
	}

	d.Reply("GET /info", http.StatusOK, nil)
	if err := Info(d.Client()); err == nil {
		t.Error("an empty /info body: want an error")
	}
	d.Reply("GET /info", http.StatusInternalServerError, nil)
	if err := Info(d.Client()); err == nil {
		t.Error("Info on HTTP 500: want an error")
	}
}

func TestFormatBytesBinary(t *testing.T) {
	cases := map[int64]string{0: "0 B", 1000: "1000 B", 1024: "1.00 KiB", 1536: "1.50 KiB", 32 << 30: "32.00 GiB", 5 << 50: "5.00 PiB"}
	for in, want := range cases {
		if got := formatBytesBinary(in); got != want {
			t.Errorf("formatBytesBinary(%d) = %q; want %q", in, got, want)
		}
	}
	if formatBool(true) != "true" || formatBool(false) != "false" {
		t.Error("formatBool")
	}
	if !(PluginsInfo{}).Empty() || (PluginsInfo{Volume: []string{"local"}}).Empty() {
		t.Error("PluginsInfo.Empty")
	}
}

// registryWithDefault serves a registry and points the default-registry file at it
func registryWithDefault(t *testing.T) *httptest.Server {
	t.Helper()
	resetGlobals(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/_catalog":
			io.WriteString(w, `{"repositories":["team/app","team/db"]}`)
		case "/v2/team/app/tags/list":
			io.WriteString(w, `{"name":"team/app","tags":["v1","v2"]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	env.RegConfigFile = filepath.Join(t.TempDir(), "defaultRegistry.json")
	os.WriteFile(env.RegConfigFile, []byte(`{"RegistryName":"`+srv.URL+`"}`), 0o600)
	return srv
}

func TestGetCatalogAndTags(t *testing.T) {
	registryWithDefault(t)

	out := testutil.CaptureStdout(t, func() {
		if err := GetCatalog(); err != nil {
			t.Errorf("GetCatalog: %v", err)
		}
	})
	if !strings.Contains(out, "team/db") {
		t.Errorf("catalog output = %q", out)
	}

	JSONoutputfile = filepath.Join(t.TempDir(), "tags.json")
	if err := GetTags("team/app"); err != nil {
		t.Fatalf("GetTags: %v", err)
	}
	var tags TagsListResponse
	b, _ := os.ReadFile(JSONoutputfile)
	if err := json.Unmarshal(b, &tags); err != nil || !reflect.DeepEqual(tags.Tags, []string{"v1", "v2"}) {
		t.Errorf("tags file = %s, %v", b, err)
	}

	if err := GetTags("team/ghost"); err == nil {
		t.Error("GetTags of an unknown repo: want an error")
	}
	env.RegConfigFile = filepath.Join(t.TempDir(), "missing.json")
	if err := GetCatalog(); err == nil {
		t.Error("GetCatalog without a default registry: want an error")
	}
}
