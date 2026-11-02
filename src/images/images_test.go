// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/images/images_test.go

package images

import (
	"bytes"
	"compress/gzip"
	"dtools2/extras"
	"dtools2/internal/testutil"
	"dtools2/rest"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

func resetGlobals(t *testing.T) {
	t.Helper()
	testutil.IsolateHome(t)
	force, rmBL := ForceRemove, RemoveBlacklisted
	quiet, outFmt := rest.QuietOutput, extras.OutputFormat
	rest.QuietOutput = true
	t.Cleanup(func() {
		ForceRemove, RemoveBlacklisted = force, rmBL
		rest.QuietOutput, extras.OutputFormat = quiet, outFmt
	})
}

func newDaemon(t *testing.T) *testutil.Daemon {
	t.Helper()
	resetGlobals(t)
	return testutil.NewDaemon(t)
}

func TestImagesList(t *testing.T) {
	d := newDaemon(t)
	d.Reply("GET /images/json", http.StatusOK, []map[string]any{
		{"Id": "sha256:0123456789abcdef", "RepoTags": []string{"alpine:3.20", "alpine:latest"}, "Size": 8000000, "Containers": 2},
		{"Id": "sha256:fedcba9876543210", "RepoTags": []string{"registry:5000/team/app:v1"}},
	})

	imgs, err := ImagesList(d.Client(), false)
	if err != nil {
		t.Fatalf("ImagesList: %v", err)
	}
	if len(imgs) != 3 {
		t.Fatalf("got %d entries; want one per tag (3)", len(imgs))
	}
	if imgs[0].RepoImgName != "alpine" || imgs[0].ImgTag != "3.20" || imgs[0].ID != "0123456789abcdef" || imgs[0].Containers != 2 {
		t.Errorf("first entry = %+v; want alpine:3.20 with the sha256: prefix dropped", imgs[0])
	}
	if imgs[2].RepoImgName != "registry:5000/team/app" || imgs[2].ImgTag != "v1" {
		t.Errorf("registry entry = %+v", imgs[2])
	}

	rest.QuietOutput = false
	out := testutil.CaptureStdout(t, func() { _, _ = ImagesList(d.Client(), true) })
	if !strings.Contains(out, "alpine") || !strings.Contains(out, "0123456789ab") {
		t.Errorf("table output %q: want names and short IDs", out)
	}

	extras.OutputFormat = "ImgTag"
	out = testutil.CaptureStdout(t, func() { _, _ = ImagesList(d.Client(), true) })
	if got := strings.Fields(out); !reflect.DeepEqual(got, []string{"3.20", "latest", "v1"}) {
		t.Errorf("--format ImgTag output = %v", got)
	}
}

func TestImagesListErrors(t *testing.T) {
	d := newDaemon(t)
	d.Reply("GET /images/json", http.StatusInternalServerError, map[string]string{"message": "boom"})
	if _, err := ImagesList(d.Client(), false); err == nil {
		t.Error("ImagesList on HTTP 500: want an error")
	}
	d.Handle("GET /images/json", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("{")) })
	if _, err := ImagesList(d.Client(), false); err == nil {
		t.Error("ImagesList on invalid JSON: want an error")
	}
}

func TestPull(t *testing.T) {
	d := newDaemon(t)
	d.Handle("POST /images/create", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("fromImage") == "ghost" {
			io.WriteString(w, `{"errorDetail":{"message":"manifest unknown"},"error":"manifest unknown"}`)
			return
		}
		io.WriteString(w, `{"status":"Pulling from library/alpine"}`+"\n"+`{"status":"Download complete"}`)
	})

	var out bytes.Buffer
	if err := PullRef(d.Client(), "alpine:3.20", &out); err != nil {
		t.Fatalf("PullRef: %v", err)
	}
	q := d.Calls(http.MethodPost, "/images/create")[0].Query
	if q.Get("fromImage") != "alpine" || q.Get("tag") != "3.20" {
		t.Errorf("pull query = %v; want fromImage=alpine tag=3.20", q)
	}
	if !strings.Contains(out.String(), "Download complete") {
		t.Errorf("progress output %q: want the daemon's messages rendered", out.String())
	}

	_ = PullRef(d.Client(), "busybox", io.Discard)
	if q := d.Calls(http.MethodPost, "/images/create")[1].Query; q.Get("tag") != "latest" {
		t.Errorf("untagged pull query = %v; want tag=latest", q)
	}

	if err := PullRef(d.Client(), "ghost", io.Discard); err == nil || !strings.Contains(err.Error(), "manifest unknown") {
		t.Errorf("PullRef(ghost) = %v; want the daemon's error", err)
	}
	if err := Pull(d.Client(), PullOptions{}, io.Discard); err == nil {
		t.Error("Pull without a reference: want an error")
	}
}

// writeDockerAuth stores a login for registry in the isolated $DOCKER_CONFIG/config.json
func writeDockerAuth(t *testing.T, registry string) {
	t.Helper()
	dir := os.Getenv("DOCKER_CONFIG")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := `{"auths":{"` + registry + `":{"auth":"dXNlcjpwYXNz"}}}` // user:pass
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestPullSendsRegistryAuth(t *testing.T) {
	d := newDaemon(t)
	writeDockerAuth(t, "registry.example.com")
	d.Reply("POST /images/create", http.StatusOK, map[string]string{"status": "ok"})
	if err := PullRef(d.Client(), "registry.example.com/team/app:v1", io.Discard); err != nil {
		t.Fatalf("PullRef: %v", err)
	}
	call := d.Calls(http.MethodPost, "/images/create")[0]
	if call.Query.Get("fromImage") != "registry.example.com/team/app" {
		t.Errorf("fromImage = %q", call.Query.Get("fromImage"))
	}
	if call.Header.Get("X-Registry-Auth") == "" {
		t.Error("a pull from a registry with a stored login must send X-Registry-Auth")
	}
}

// Public images on an explicit registry (ghcr.io, quay.io...) must be pullable without a login,
// as with `docker pull`
func TestPullAnonymousFromExplicitRegistry(t *testing.T) {
	d := newDaemon(t)
	d.Reply("POST /images/create", http.StatusOK, map[string]string{"status": "ok"})
	if err := PullRef(d.Client(), "quay.io/prometheus/node-exporter:v1.8.0", io.Discard); err != nil {
		t.Fatalf("anonymous PullRef from quay.io: %v", err)
	}
	if len(d.Calls(http.MethodPost, "/images/create")) != 1 {
		t.Error("the pull request was never sent")
	}
}

func TestImagePush(t *testing.T) {
	d := newDaemon(t)
	writeDockerAuth(t, "registry.example.com")
	d.Reply("POST /images/{name...}", http.StatusOK, map[string]string{"status": "pushed"})
	_ = testutil.CaptureStdout(t, func() {
		if err := ImagePush(d.Client(), "registry.example.com/team/app:v2"); err != nil {
			t.Errorf("ImagePush: %v", err)
		}
	})
	calls := d.Calls(http.MethodPost, "/images/registry.example.com/team/app/push")
	if len(calls) != 1 || calls[0].Query.Get("tag") != "v2" || calls[0].Header.Get("X-Registry-Auth") == "" {
		t.Errorf("push calls %+v; want one with tag=v2 and an auth header", calls)
	}

	d.Reply("POST /images/{name...}", http.StatusInternalServerError, map[string]string{"message": "denied"})
	_ = testutil.CaptureStdout(t, func() {
		if err := ImagePush(d.Client(), "app"); err == nil {
			t.Error("ImagePush on HTTP 500: want an error")
		}
	})
}

func TestTagImage(t *testing.T) {
	d := newDaemon(t)
	d.Reply("POST /images/{name...}", http.StatusCreated, nil)
	if err := TagImage(d.Client(), "alpine:latest", "registry:5000/mine/alpine:v1"); err != nil {
		t.Fatalf("TagImage: %v", err)
	}
	call := d.Calls(http.MethodPost, "/images/alpine:latest/tag")
	if len(call) != 1 || call[0].Query.Get("repo") != "registry:5000/mine/alpine" || call[0].Query.Get("tag") != "v1" {
		t.Errorf("tag calls %+v; want repo=registry:5000/mine/alpine tag=v1", call)
	}
	d.Reply("POST /images/{name...}", http.StatusNotFound, nil)
	if err := TagImage(d.Client(), "ghost", "x:y"); err == nil {
		t.Error("TagImage on HTTP 404: want an error")
	}
}

func TestRemoveImage(t *testing.T) {
	d := newDaemon(t)
	dir := filepath.Join(os.Getenv("HOME"), ".config", "JFG", "dtools")
	os.WriteFile(filepath.Join(dir, "blacklist.json"), []byte(`{"Images":["nginx:latest"]}`), 0o600)
	d.Reply("DELETE /images/{name...}", http.StatusOK, []any{})

	ForceRemove = true
	if err := RemoveImage(d.Client(), []string{"alpine:latest", "nginx:latest"}); err != nil {
		t.Fatalf("RemoveImage: %v", err)
	}
	del := d.Calls(http.MethodDelete, "/images/alpine:latest")
	if len(del) != 1 || del[0].Query.Get("force") != "true" {
		t.Errorf("DELETE calls %+v; want one with force=true", del)
	}
	if len(d.Calls(http.MethodDelete, "/images/nginx:latest")) != 0 {
		t.Error("a blacklisted image must not be removed without -B")
	}

	d.Reply("DELETE /images/{name...}", http.StatusConflict, map[string]string{"message": "image is in use"})
	if err := RemoveImage(d.Client(), []string{"alpine:latest"}); err == nil {
		t.Error("RemoveImage on HTTP 409: want an error")
	}
}

func TestRemoveImageBlacklistedWithFlag(t *testing.T) {
	for _, quiet := range []bool{false, true} {
		d := newDaemon(t)
		dir := filepath.Join(os.Getenv("HOME"), ".config", "JFG", "dtools")
		os.WriteFile(filepath.Join(dir, "blacklist.json"), []byte(`{"Images":["nginx:latest"]}`), 0o600)
		d.Reply("DELETE /images/{name...}", http.StatusOK, []any{})
		RemoveBlacklisted, rest.QuietOutput = true, quiet

		_ = testutil.CaptureStdout(t, func() { _ = RemoveImage(d.Client(), []string{"nginx:latest"}) })
		if len(d.Calls(http.MethodDelete, "/images/nginx:latest")) != 1 {
			t.Errorf("quiet=%v: -B should remove a blacklisted image", quiet)
		}
	}
}

func TestImageCommit(t *testing.T) {
	d := newDaemon(t)
	d.Reply("POST /commit", http.StatusCreated, map[string]string{"Id": "sha256:new"})
	err := ImageCommit(d.Client(), "web", "mine/web:v2", "jfg", "snapshot", []string{"ENV A=1", "  ", "EXPOSE 80"})
	if err != nil {
		t.Fatalf("ImageCommit: %v", err)
	}
	q := d.Calls(http.MethodPost, "/commit")[0].Query
	if q.Get("container") != "web" || q.Get("repo") != "mine/web" || q.Get("tag") != "v2" || q.Get("author") != "jfg" || q.Get("comment") != "snapshot" {
		t.Errorf("commit query = %v", q)
	}
	if !reflect.DeepEqual(q["changes"], []string{"ENV A=1", "EXPOSE 80"}) {
		t.Errorf("changes = %v; want blank entries dropped", q["changes"])
	}

	if err := ImageCommit(d.Client(), "web", "mine/web", "", "", nil); err == nil {
		t.Error("ImageCommit without a tag: want an error")
	}
	d.Reply("POST /commit", http.StatusNotFound, map[string]string{"message": "no such container"})
	if err := ImageCommit(d.Client(), "ghost", "a:b", "", "", nil); err == nil || !strings.Contains(err.Message, "no such container") {
		t.Errorf("ImageCommit on HTTP 404 = %v; want the daemon message", err)
	}
}

func TestInspectImage(t *testing.T) {
	d := newDaemon(t)
	d.Reply("GET /images/{name...}", http.StatusOK, map[string]any{"Id": "sha256:abc", "RepoTags": []string{"registry:5000/team/app:v1"}})

	var img ImageInspect
	var err *ce.CustomError
	out := testutil.CaptureStdout(t, func() { img, err = InspectImage(d.Client(), "registry:5000/team/app:v1") })
	if err != nil || img.ID != "sha256:abc" {
		t.Fatalf("InspectImage = %+v, %v", img, err)
	}
	// ':' and '/' must reach the daemon unescaped
	if len(d.Calls(http.MethodGet, "/images/registry:5000/team/app:v1/json")) != 1 {
		t.Errorf("requests %v; want the reference kept intact in the path", d.Sequence())
	}
	if !strings.Contains(out, `"Id":"sha256:abc"`) {
		t.Errorf("quiet output %q: want the raw JSON", out)
	}

	d.Reply("GET /images/{name...}", http.StatusNotFound, map[string]string{"message": "no such image"})
	if _, err := InspectImage(d.Client(), "ghost"); err == nil || !strings.Contains(err.Title, "404") {
		t.Errorf("InspectImage(ghost) = %v; want an HTTP 404 error", err)
	}
}

func TestImageSave(t *testing.T) {
	for _, name := range []string{"imgs.tar", "imgs.tar.gz", "imgs.tar.bz2"} {
		t.Run(name, func(t *testing.T) {
			d := newDaemon(t)
			d.Reply("GET /images/get", http.StatusOK, "tar-stream")
			out := filepath.Join(t.TempDir(), name)

			if err := ImageSave(d.Client(), []string{"alpine:latest", "", "busybox"}, out); err != nil {
				t.Fatalf("ImageSave: %v", err)
			}
			if q := d.Calls(http.MethodGet, "/images/get")[0].Query; !reflect.DeepEqual(q["names"], []string{"alpine:latest", "busybox"}) {
				t.Errorf("names = %v; want empty references dropped", q["names"])
			}
			r, err := openArchiveReader(out)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			if b, _ := io.ReadAll(r); !strings.Contains(string(b), "tar-stream") {
				t.Errorf("saved content = %q; want the daemon's stream", b)
			}
		})
	}
}

func TestImageSaveErrors(t *testing.T) {
	d := newDaemon(t)
	dir := t.TempDir()
	if err := ImageSave(d.Client(), []string{""}, filepath.Join(dir, "a.tar")); err == nil {
		t.Error("ImageSave without an image: want an error")
	}
	if err := ImageSave(d.Client(), []string{"a"}, filepath.Join(dir, "a.tar.xz")); err == nil {
		t.Error("ImageSave to .xz: want an error (unsupported)")
	}
	d.Reply("GET /images/get", http.StatusNotFound, map[string]string{"message": "no such image"})
	if err := ImageSave(d.Client(), []string{"ghost"}, filepath.Join(dir, "b.tar")); err == nil || !strings.Contains(err.Message, "no such image") {
		t.Errorf("ImageSave on HTTP 404 = %v; want the daemon message", err)
	}
}

// ImageLoad must upload the (decompressed) archive to POST /images/load
func TestImageLoad(t *testing.T) {
	d := newDaemon(t)
	d.Reply("POST /images/load", http.StatusOK, map[string]string{"stream": "Loaded image: alpine:latest\n"})

	path := filepath.Join(t.TempDir(), "imgs.tar.gz")
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte("image-tarball"))
	gz.Close()
	os.WriteFile(path, buf.Bytes(), 0o644)

	if err := ImageLoad(d.Client(), path); err != nil {
		t.Fatalf("ImageLoad: %v", err)
	}
	load := d.Calls(http.MethodPost, "/images/load")
	if len(load) != 1 {
		t.Fatalf("requests %v; want POST /images/load", d.Sequence())
	}
	if string(load[0].Body) != "image-tarball" || load[0].Header.Get("Content-Type") != "application/x-tar" {
		t.Errorf("load body %q, content-type %q; want the decompressed archive as application/x-tar", load[0].Body, load[0].Header.Get("Content-Type"))
	}
}

func TestImageLoadMissingFile(t *testing.T) {
	d := newDaemon(t)
	if err := ImageLoad(d.Client(), filepath.Join(t.TempDir(), "nope.tar")); err == nil {
		t.Error("ImageLoad of a missing file: want an error")
	}
}

func TestArchiveRoundTrip(t *testing.T) {
	for _, name := range []string{"a.tar", "a.tar.gz", "a.tgz", "a.tar.bz2", "a.tbz2"} {
		path := filepath.Join(t.TempDir(), name)
		w, cerr := openArchiveWriter(path)
		if cerr != nil {
			t.Fatalf("%s: openArchiveWriter: %v", name, cerr)
		}
		w.Write([]byte("payload"))
		if err := w.Close(); err != nil {
			t.Fatalf("%s: close: %v", name, err)
		}
		r, err := openArchiveReader(path)
		if err != nil {
			t.Fatalf("%s: openArchiveReader: %v", name, err)
		}
		b, _ := io.ReadAll(r)
		r.Close()
		if string(b) != "payload" {
			t.Errorf("%s: round trip = %q", name, b)
		}
	}
}

func TestDetectCompression(t *testing.T) {
	cases := map[string]archiveCompression{
		"a.tar": compNone, "A.TAR.GZ": compGzip, "a.tgz": compGzip, "a.gz": compGzip,
		"a.tar.bz2": compBzip2, "a.tbz": compBzip2, "a.tar.xz": compXz, "a.txz": compXz,
	}
	for name, want := range cases {
		if got := detectCompression(name); got != want {
			t.Errorf("detectCompression(%q) = %v; want %v", name, got, want)
		}
	}
}

func TestOpenArchiveReaderErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := openArchiveReader(filepath.Join(dir, "missing.tar")); err == nil {
		t.Error("missing file: want an error")
	}
	bad := filepath.Join(dir, "bad.tar.gz")
	os.WriteFile(bad, []byte("not gzip"), 0o644)
	if _, err := openArchiveReader(bad); err == nil {
		t.Error("corrupt gzip: want an error")
	}
	badxz := filepath.Join(dir, "bad.tar.xz")
	os.WriteFile(badxz, []byte("not xz"), 0o644)
	if _, err := openArchiveReader(badxz); err == nil {
		t.Error("corrupt xz: want an error")
	}
	if _, err := openArchiveWriter(filepath.Join(dir, "no", "such", "dir.tar")); err == nil {
		t.Error("unwritable path: want an error")
	}
}

func TestReadSmallBody(t *testing.T) {
	got, err := readSmallBody(strings.NewReader("  hello world  "), 8)
	// The first 8 bytes are "  hello ", trimmed to "hello"
	if err != nil || got != "hello" {
		t.Errorf("readSmallBody = %q, %v; want the first 8 bytes, trimmed", got, err)
	}
}
