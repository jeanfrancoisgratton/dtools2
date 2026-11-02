// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/run_build/run_build_test.go

package run_build

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"dtools2/internal/testutil"
	"dtools2/rest"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// resetRunFlags zeroes every run flag and restores the originals afterwards
func resetRunFlags(t *testing.T) {
	t.Helper()
	type snapshot struct {
		detach, interactive, tty, remove, privileged, readOnly bool
		name, user, workdir, network, entrypoint, hostname     string
		memory, restart, shm                                   string
		env, publish, volume, mount, ulimits, capAdd, capDrop  []string
		cpus                                                   float64
		cpuShares, pids                                        int64
		quiet                                                  bool
		ctx                                                    context.Context
	}
	s := snapshot{RunDetach, RunInteractive, RunTTY, RunRemove, RunPrivileged, RunReadOnly,
		RunName, RunUser, RunWorkdir, RunNetwork, RunEntrypoint, RunHostname, RunMemory, RunRestart, RunShmSize,
		RunEnv, RunPublish, RunVolume, RunMount, RunUlimits, RunCapAdd, RunCapDrop, RunCPUs, RunCPUShares, RunPidsLimit,
		rest.QuietOutput, rest.Context}
	restore := func(s snapshot) {
		RunDetach, RunInteractive, RunTTY, RunRemove, RunPrivileged, RunReadOnly = s.detach, s.interactive, s.tty, s.remove, s.privileged, s.readOnly
		RunName, RunUser, RunWorkdir, RunNetwork, RunEntrypoint, RunHostname = s.name, s.user, s.workdir, s.network, s.entrypoint, s.hostname
		RunMemory, RunRestart, RunShmSize = s.memory, s.restart, s.shm
		RunEnv, RunPublish, RunVolume, RunMount, RunUlimits, RunCapAdd, RunCapDrop = s.env, s.publish, s.volume, s.mount, s.ulimits, s.capAdd, s.capDrop
		RunCPUs, RunCPUShares, RunPidsLimit = s.cpus, s.cpuShares, s.pids
		rest.QuietOutput, rest.Context = s.quiet, s.ctx
	}
	restore(snapshot{quiet: true, ctx: context.Background()})
	t.Cleanup(func() { restore(s) })
}

func TestParsePublishSpec(t *testing.T) {
	cases := []struct {
		spec string
		key  string
		bind PortBinding
	}{
		{"80", "80/tcp", PortBinding{}},
		{":80", "80/tcp", PortBinding{}},
		{"8080:80", "80/tcp", PortBinding{HostPort: "8080"}},
		{"127.0.0.1:8080:80", "80/tcp", PortBinding{HostIP: "127.0.0.1", HostPort: "8080"}},
		{"127.0.0.1::80", "80/tcp", PortBinding{HostIP: "127.0.0.1"}},
		{"5353:53/UDP", "53/udp", PortBinding{HostPort: "5353"}},
		{"80/", "80/tcp", PortBinding{}},
	}
	for _, c := range cases {
		key, bind, err := parsePublishSpec(c.spec)
		if err != nil || key != c.key || bind != c.bind {
			t.Errorf("parsePublishSpec(%q) = %q, %+v, %v; want %q, %+v", c.spec, key, bind, err, c.key, c.bind)
		}
	}
	for _, bad := range []string{"", "8080:", "a:b:c:d", "8080:http", "abc:80"} {
		if _, _, err := parsePublishSpec(bad); err == nil {
			t.Errorf("parsePublishSpec(%q): want an error", bad)
		}
	}
}

// docker accepts IPv6 host addresses in brackets
func TestParsePublishSpecIPv6(t *testing.T) {
	key, bind, err := parsePublishSpec("[::1]:8080:80")
	if err != nil || key != "80/tcp" || bind.HostIP != "::1" || bind.HostPort != "8080" {
		t.Errorf("parsePublishSpec([::1]:8080:80) = %q, %+v, %v; want 80/tcp on ::1:8080", key, bind, err)
	}
}

func TestParseVolumeSpec(t *testing.T) {
	cases := []struct {
		spec string
		want Mount
		anon string
	}{
		{"/data", Mount{}, "/data"},
		{"/host:/ctr", Mount{Type: "bind", Source: "/host", Target: "/ctr"}, ""},
		{"/host:/ctr:ro", Mount{Type: "bind", Source: "/host", Target: "/ctr", ReadOnly: true}, ""},
		{"/host:/ctr:rw", Mount{Type: "bind", Source: "/host", Target: "/ctr"}, ""},
		{"/host:/ctr:z,ro", Mount{Type: "bind", Source: "/host", Target: "/ctr", ReadOnly: true}, ""},
		{"pgdata:/var/lib/postgresql", Mount{Type: "volume", Source: "pgdata", Target: "/var/lib/postgresql"}, ""},
	}
	for _, c := range cases {
		m, anon, err := parseVolumeSpec(c.spec)
		if err != nil || m != c.want || anon != c.anon {
			t.Errorf("parseVolumeSpec(%q) = %+v, %q, %v; want %+v, %q", c.spec, m, anon, err, c.want, c.anon)
		}
	}
	for _, bad := range []string{"", "relative", "a:b:c:d", ":/ctr", "/host:", "/host:relative"} {
		if _, _, err := parseVolumeSpec(bad); err == nil {
			t.Errorf("parseVolumeSpec(%q): want an error", bad)
		}
	}
}

// The daemon refuses relative bind sources; the docker CLI resolves them against the cwd
func TestParseVolumeSpecRelativePath(t *testing.T) {
	m, _, err := parseVolumeSpec("./data:/data")
	if err != nil {
		t.Fatalf("parseVolumeSpec: %v", err)
	}
	if !filepath.IsAbs(m.Source) {
		t.Errorf("bind source = %q; want it resolved to an absolute path", m.Source)
	}
}

func TestParseMountSpec(t *testing.T) {
	cases := []struct {
		spec string
		want Mount
	}{
		{"type=bind,src=/h,dst=/c", Mount{Type: "bind", Source: "/h", Target: "/c"}},
		{"type=volume,source=v,target=/c,readonly", Mount{Type: "volume", Source: "v", Target: "/c", ReadOnly: true}},
		{"src=/h,destination=/c,ro=true", Mount{Type: "bind", Source: "/h", Target: "/c", ReadOnly: true}},
		{"src=v,dst=/c", Mount{Type: "volume", Source: "v", Target: "/c"}},
		{"type=tmpfs,dst=/run", Mount{Type: "tmpfs", Target: "/run"}},
		{"dst=/anon", Mount{Type: "volume", Target: "/anon"}},
		{"type=bind,src=/h,dst=/c,ro,rw", Mount{Type: "bind", Source: "/h", Target: "/c"}},
		{"TYPE=Bind , SRC=/h , DST=/c , rw=false", Mount{Type: "bind", Source: "/h", Target: "/c", ReadOnly: true}},
	}
	for _, c := range cases {
		m, err := parseMountSpec(c.spec)
		if err != nil || m != c.want {
			t.Errorf("parseMountSpec(%q) = %+v, %v; want %+v", c.spec, m, err, c.want)
		}
	}
	bad := []string{"", "src=/h", "dst=relative", "type=nfs,dst=/c", "type=bind,dst=/c", "type=tmpfs,src=x,dst=/c", "bogus,dst=/c", "foo=bar,dst=/c", "dst=/c,ro=maybe", "dst=/c,rw=maybe"}
	for _, b := range bad {
		if _, err := parseMountSpec(b); err == nil {
			t.Errorf("parseMountSpec(%q): want an error", b)
		}
	}
}

func TestParseBoolish(t *testing.T) {
	for _, v := range []string{"", "1", "TRUE", "yes", "on", " t "} {
		if b, err := parseBoolish(v); err != nil || !b {
			t.Errorf("parseBoolish(%q) = %v, %v; want true", v, b, err)
		}
	}
	for _, v := range []string{"0", "false", "No", "off"} {
		if b, err := parseBoolish(v); err != nil || b {
			t.Errorf("parseBoolish(%q) = %v, %v; want false", v, b, err)
		}
	}
	if _, err := parseBoolish("perhaps"); err == nil {
		t.Error("parseBoolish(perhaps): want an error")
	}
}

func TestApplyHelpers(t *testing.T) {
	req := &ContainerCreateRequest{}
	if err := applyPublish(req, []string{"8080:80", "8443:80", "53/udp"}); err != nil {
		t.Fatal(err)
	}
	if len(req.HostConfig.PortBindings["80/tcp"]) != 2 || len(req.ExposedPorts) != 2 {
		t.Errorf("port bindings = %+v, exposed %v", req.HostConfig.PortBindings, req.ExposedPorts)
	}
	if err := applyVolumes(req, []string{"/anon", "v:/data"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := req.Volumes["/anon"]; !ok || len(req.HostConfig.Mounts) != 1 {
		t.Errorf("volumes = %v, mounts = %+v", req.Volumes, req.HostConfig.Mounts)
	}
	if err := applyMounts(req, []string{"type=tmpfs,dst=/run"}); err != nil || len(req.HostConfig.Mounts) != 2 {
		t.Errorf("mounts after applyMounts = %+v, %v", req.HostConfig.Mounts, err)
	}
	if err := applyPublish(req, []string{"bad:port"}); err == nil {
		t.Error("applyPublish(invalid): want an error")
	}
	if err := applyVolumes(req, []string{"rel"}); err == nil {
		t.Error("applyVolumes(invalid): want an error")
	}
	if err := applyMounts(req, []string{"src=/x"}); err == nil {
		t.Error("applyMounts(invalid): want an error")
	}
	empty := &ContainerCreateRequest{}
	_ = applyPublish(empty, nil)
	_ = applyVolumes(empty, nil)
	_ = applyMounts(empty, nil)
	if empty.HostConfig != nil {
		t.Error("no flags must leave HostConfig untouched")
	}
}

func TestGetRunFlagValues(t *testing.T) {
	resetRunFlags(t)
	RunUlimits, RunMemory, RunCPUs, RunCPUShares = []string{"nofile=1024:2048"}, "512m", 1.5, 512
	RunRestart, RunPrivileged, RunCapAdd, RunCapDrop = "on-failure:3", true, []string{"NET_ADMIN"}, []string{"MKNOD"}
	RunReadOnly, RunShmSize, RunPidsLimit = true, "64m", 100

	hc, err := getRunFlagValues()
	if err != nil {
		t.Fatalf("getRunFlagValues: %v", err)
	}
	want := &HostConfig{
		Ulimits: []UlimitFlag{{Name: "nofile", Soft: 1024, Hard: 2048}}, Memory: 512 << 20, NanoCPUs: 1_500_000_000, CpuShares: 512,
		RestartPolicy: RestartPolicy{Name: "on-failure", MaximumRetryCount: 3}, Privileged: true,
		CapAdd: []string{"NET_ADMIN"}, CapDrop: []string{"MKNOD"}, ReadonlyRootfs: true, ShmSize: 64 << 20, PidsLimit: 100,
	}
	if !reflect.DeepEqual(hc, want) {
		t.Errorf("HostConfig = %+v; want %+v", hc, want)
	}

	for _, set := range []func(){
		func() { RunMemory = "lots" },
		func() { RunShmSize = "x" },
		func() { RunRestart = "sometimes" },
		func() { RunUlimits = []string{"bogus"} },
	} {
		resetRunFlags(t)
		set()
		if _, err := getRunFlagValues(); err == nil {
			t.Error("an invalid run flag: want an error")
		}
	}
}

// runDaemon accepts any create; images listed in missing are reported absent until pulled
func runDaemon(t *testing.T, missing ...string) *testutil.Daemon {
	t.Helper()
	resetRunFlags(t)
	testutil.IsolateHome(t)
	d := testutil.NewDaemon(t)
	pulled := map[string]bool{}
	d.Handle("POST /containers/create", func(w http.ResponseWriter, r *http.Request) {
		var req ContainerCreateRequest
		json.NewDecoder(r.Body).Decode(&req)
		for _, m := range missing {
			if req.Image == m && !pulled[m] {
				testutil.JSON(w, http.StatusNotFound, map[string]string{"message": "No such image: " + m})
				return
			}
		}
		testutil.JSON(w, http.StatusCreated, map[string]any{"Id": "ctr123", "Warnings": []string{}})
	})
	d.Handle("POST /images/create", func(w http.ResponseWriter, r *http.Request) {
		pulled[r.URL.Query().Get("fromImage")+":"+r.URL.Query().Get("tag")] = true
		testutil.JSON(w, http.StatusOK, map[string]string{"status": "Downloaded"})
	})
	return d
}

func TestCreateContainerPayload(t *testing.T) {
	d := runDaemon(t)
	RunInteractive, RunTTY, RunRemove, RunName, RunUser = true, true, true, "web", "1000"
	RunEnv, RunWorkdir, RunHostname, RunNetwork, RunEntrypoint = []string{"A=1"}, "/app", "h1", "appnet", "/bin/sh"
	RunPublish, RunVolume = []string{"8080:80"}, []string{"pgdata:/data"}

	id, missing, err := createContainer(d.Client(), "alpine:latest", []string{"echo", "hi"})
	if err != nil || missing || id != "ctr123" {
		t.Fatalf("createContainer = %q, %v, %v", id, missing, err)
	}
	call := d.Calls(http.MethodPost, "/containers/create")[0]
	if call.Query.Get("name") != "web" {
		t.Errorf("name query = %q", call.Query.Get("name"))
	}
	var req ContainerCreateRequest
	json.Unmarshal(call.Body, &req)
	if req.Image != "alpine:latest" || !reflect.DeepEqual(req.Cmd, []string{"echo", "hi"}) || !reflect.DeepEqual(req.Entrypoint, []string{"/bin/sh"}) {
		t.Errorf("image/cmd/entrypoint = %q %v %v", req.Image, req.Cmd, req.Entrypoint)
	}
	if !req.AttachStdin || !req.OpenStdin || !req.Tty || !req.AttachStdout || req.User != "1000" || req.WorkingDir != "/app" || req.Hostname != "h1" {
		t.Errorf("IO/process fields = %+v", req)
	}
	hc := req.HostConfig
	if !hc.AutoRemove || hc.NetworkMode != "appnet" || len(hc.PortBindings["80/tcp"]) != 1 || len(hc.Mounts) != 1 {
		t.Errorf("HostConfig = %+v", hc)
	}

	RunDetach = true
	_, _, _ = createContainer(d.Client(), "alpine:latest", nil)
	var detached ContainerCreateRequest
	json.Unmarshal(d.Calls(http.MethodPost, "/containers/create")[1].Body, &detached)
	if detached.AttachStdout || detached.AttachStderr || detached.Cmd != nil {
		t.Errorf("detached create = %+v; want no stdout/stderr attach and no Cmd", detached)
	}
}

func TestCreateContainerWithAutoPull(t *testing.T) {
	d := runDaemon(t, "busybox:latest")
	id, err := createContainerWithAutoPull(d.Client(), "busybox:latest", nil)
	if err != nil || id != "ctr123" {
		t.Fatalf("createContainerWithAutoPull = %q, %v", id, err)
	}
	if got := d.Sequence(); !reflect.DeepEqual(got, []string{"POST /containers/create", "POST /images/create", "POST /containers/create"}) {
		t.Errorf("requests = %v; want create, pull, create", got)
	}
}

func TestCreateContainerErrors(t *testing.T) {
	d := runDaemon(t)
	d.Reply("POST /containers/create", http.StatusConflict, map[string]string{"message": "name already in use"})
	if _, err := createContainerWithAutoPull(d.Client(), "alpine", nil); err == nil || !strings.Contains(err.Message, "already in use") {
		t.Errorf("create on HTTP 409 = %v; want the daemon message, no pull", err)
	}
	if len(d.Calls(http.MethodPost, "/images/create")) != 0 {
		t.Error("a non-'no such image' error must not trigger a pull")
	}

	d.Reply("POST /containers/create", http.StatusCreated, map[string]string{"Id": ""})
	if _, _, err := createContainer(d.Client(), "alpine", nil); err == nil {
		t.Error("an empty container ID: want an error")
	}

	RunMemory = "lots"
	if _, _, err := createContainer(d.Client(), "alpine", nil); err == nil {
		t.Error("an invalid run flag: want an error before any request")
	}
}

func TestWaitContainerExit(t *testing.T) {
	d := runDaemon(t)
	d.Reply("POST /containers/ok/wait", http.StatusOK, map[string]any{"StatusCode": 3})
	d.Reply("POST /containers/bad/wait", http.StatusOK, map[string]any{"StatusCode": 0, "Error": map[string]string{"Message": "oom"}})
	d.Reply("POST /containers/gone/wait", http.StatusNotFound, map[string]string{"message": "no such container"})

	if code, err := waitContainerExit(d.Client(), "ok"); err != nil || code != 3 {
		t.Errorf("wait(ok) = %d, %v; want 3", code, err)
	}
	if q := d.Calls(http.MethodPost, "/containers/ok/wait")[0].Query; q.Get("condition") != "not-running" {
		t.Errorf("wait query = %v", q)
	}
	if _, err := waitContainerExit(d.Client(), "bad"); err == nil || err.Message != "oom" {
		t.Errorf("wait(bad) = %v; want the container error", err)
	}
	if _, err := waitContainerExit(d.Client(), "gone"); err == nil {
		t.Error("wait(gone): want an error")
	}
}

func TestStartContainerRestoresQuiet(t *testing.T) {
	d := runDaemon(t)
	d.Reply("POST /containers/{id}/start", http.StatusNoContent, nil)
	d.Reply("GET /containers/json", http.StatusOK, []any{testutil.Container("abcdef123456", "web", "exited")})

	rest.QuietOutput = true
	_ = testutil.CaptureStdout(t, func() {
		if err := startContainer(d.Client(), "abcdef123456"); err != nil {
			t.Errorf("startContainer: %v", err)
		}
	})
	if !rest.QuietOutput {
		t.Error("startContainer must restore quiet mode")
	}
}

func TestParseBuildArgs(t *testing.T) {
	t.Setenv("FROM_ENV", "envval")
	got, err := parseBuildArgs([]string{"A=1", "B=", "C=x=y", " ", "FROM_ENV", "UNSET_IN_ENV_XYZ"})
	if err != nil {
		t.Fatal(err)
	}
	flat := map[string]string{}
	for k, v := range got {
		flat[k] = *v
	}
	want := map[string]string{"A": "1", "B": "", "C": "x=y", "FROM_ENV": "envval", "UNSET_IN_ENV_XYZ": ""}
	if !reflect.DeepEqual(flat, want) {
		t.Errorf("parseBuildArgs = %v; want %v", flat, want)
	}
	if _, err := parseBuildArgs([]string{"=v"}); err == nil {
		t.Error("an empty key: want an error")
	}
}

func TestDockerfileRelative(t *testing.T) {
	saved := Dockerfile
	t.Cleanup(func() { Dockerfile = saved })
	ctxDir := t.TempDir()
	os.MkdirAll(filepath.Join(ctxDir, "docker"), 0o755)
	os.WriteFile(filepath.Join(ctxDir, "Dockerfile"), []byte("FROM scratch"), 0o644)
	os.WriteFile(filepath.Join(ctxDir, "docker", "prod.Dockerfile"), []byte("FROM scratch"), 0o644)
	outside := filepath.Join(t.TempDir(), "Dockerfile")
	os.WriteFile(outside, []byte("FROM scratch"), 0o644)

	cases := []struct {
		df, want string
		err      bool
	}{
		{"", "Dockerfile", false},
		{"Dockerfile", "Dockerfile", false},
		{"docker/prod.Dockerfile", "docker/prod.Dockerfile", false},
		{filepath.Join(ctxDir, "docker", "prod.Dockerfile"), "docker/prod.Dockerfile", false},
		{"missing", "", true},
		{"docker", "", true},
		{outside, "", true},
	}
	for _, c := range cases {
		Dockerfile = c.df
		got, err := dockerfileRelative(ctxDir)
		if (err != nil) != c.err || got != c.want {
			t.Errorf("dockerfileRelative(-f %q) = %q, %v; want %q, error=%v", c.df, got, err, c.want, c.err)
		}
	}
}

func readContextTar(t *testing.T, r io.Reader, gz bool) []string {
	t.Helper()
	if gz {
		zr, err := gzip.NewReader(r)
		if err != nil {
			t.Fatal(err)
		}
		r = zr
	}
	var names []string
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, h.Name)
	}
	sort.Strings(names)
	return names
}

func TestMakeContextTarStream(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"Dockerfile":                "FROM scratch",
		".dockerignore":             "node_modules\n*.log\n!keep.log\nsecrets/\n",
		"main.go":                   "package main",
		"app.log":                   "noise",
		"keep.log":                  "kept",
		"node_modules/pkg/index.js": "x",
		"secrets/token":             "s3cret",
		"src/lib.go":                "package src",
	}
	for name, body := range files {
		p := filepath.Join(dir, name)
		os.MkdirAll(filepath.Dir(p), 0o755)
		os.WriteFile(p, []byte(body), 0o644)
	}
	os.Symlink("main.go", filepath.Join(dir, "link.go"))

	for _, gz := range []bool{false, true} {
		r, err := makeContextTarStream(context.Background(), dir, "Dockerfile", gz)
		if err != nil {
			t.Fatal(err)
		}
		got := readContextTar(t, r, gz)
		want := []string{".dockerignore", "Dockerfile", "keep.log", "link.go", "main.go", "src/", "src/lib.go"}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("gzip=%v: context entries = %v; want %v", gz, got, want)
		}
	}

	if _, err := makeContextTarStream(context.Background(), filepath.Join(dir, "main.go"), "Dockerfile", false); err == nil {
		t.Error("a file as context: want an error")
	}
	if _, err := makeContextTarStream(context.Background(), filepath.Join(dir, "nope"), "Dockerfile", false); err == nil {
		t.Error("a missing context: want an error")
	}
}

func TestStreamLooksLikeJSON(t *testing.T) {
	cases := map[string]bool{`  {"stream":"x"}`: true, "Step 1/2": false, "": false, "\n\n": false}
	for in, want := range cases {
		if got := streamLooksLikeJSON(bufio.NewReader(strings.NewReader(in))); got != want {
			t.Errorf("streamLooksLikeJSON(%q) = %v; want %v", in, got, want)
		}
	}
}

func TestBuildRegistryConfigHeader(t *testing.T) {
	cfgDir := t.TempDir()
	t.Setenv("DOCKER_CONFIG", cfgDir)

	if h, err := buildRegistryConfigHeader(); err != nil || h != "" {
		t.Errorf("no config = %q, %v; want an empty header", h, err)
	}

	os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{"auths":{"reg:5000":{"auth":"dTpw"}," ":{"auth":"x"}}}`), 0o600)
	h, err := buildRegistryConfigHeader()
	if err != nil {
		t.Fatal(err)
	}
	b, err := base64.URLEncoding.DecodeString(h)
	if err != nil {
		t.Fatalf("header is not URL-safe base64: %v", err)
	}
	var m map[string]registryAuthConfig
	json.Unmarshal(b, &m)
	if len(m) != 1 || m["reg:5000"].Auth != "dTpw" || m["reg:5000"].ServerAddress != "reg:5000" {
		t.Errorf("registry config = %+v; want only reg:5000", m)
	}
}

func TestBuilderVersionFromDaemon(t *testing.T) {
	d := testutil.NewDaemon(t)
	d.Handle("HEAD /_ping", func(w http.ResponseWriter, _ *http.Request) { w.Header().Set("Builder-Version", "2") })
	if ok, err := DaemonRecommendsBuildKit(context.Background(), d.Client()); err != nil || !ok {
		t.Errorf("Builder-Version 2 = %v, %v; want BuildKit recommended", ok, err)
	}

	d.Handle("HEAD /_ping", func(w http.ResponseWriter, _ *http.Request) { w.Header().Set("Builder-Version", "1") })
	if ok, _ := DaemonRecommendsBuildKit(context.Background(), d.Client()); ok {
		t.Error("Builder-Version 1: BuildKit must not be recommended")
	}

	d.Handle("HEAD /_ping", func(w http.ResponseWriter, _ *http.Request) {})
	if v, ok, err := BuilderVersionFromDaemon(context.Background(), d.Client()); err != nil || ok || v != "" {
		t.Errorf("no header = %q, %v, %v; want not ok, no error", v, ok, err)
	}

	// HEAD unsupported: fall back to GET
	d2 := testutil.NewDaemon(t)
	d2.Reply("HEAD /_ping", http.StatusMethodNotAllowed, nil)
	d2.Handle("GET /_ping", func(w http.ResponseWriter, _ *http.Request) { w.Header().Set("Builder-Version", "2") })
	if v, ok, err := BuilderVersionFromDaemon(context.Background(), d2.Client()); err != nil || !ok || v != "2" {
		t.Errorf("GET fallback = %q, %v, %v; want 2", v, ok, err)
	}

	if _, _, err := pingBuilderVersion(context.Background(), nil, http.MethodGet); err == nil {
		t.Error("a nil client: want an error")
	}
}

func TestDecideBuildKit(t *testing.T) {
	d := testutil.NewDaemon(t)
	d.Handle("HEAD /_ping", func(w http.ResponseWriter, _ *http.Request) { w.Header().Set("Builder-Version", "1") })

	t.Setenv("DOCKER_BUILDKIT", "0")
	if use, forced := decideBuildKit(context.Background(), d.Client(), "auto"); use || !forced {
		t.Errorf("DOCKER_BUILDKIT=0 = %v, %v; want off, forced", use, forced)
	}
	t.Setenv("DOCKER_BUILDKIT", "1")
	if use, forced := decideBuildKit(context.Background(), d.Client(), "auto"); !use || !forced {
		t.Errorf("DOCKER_BUILDKIT=1 = %v, %v; want on, forced", use, forced)
	}
	os.Unsetenv("DOCKER_BUILDKIT")
	for _, mode := range []string{"auto", "tty"} {
		if use, forced := decideBuildKit(context.Background(), d.Client(), mode); use || forced {
			t.Errorf("mode %s, daemon builder 1 = %v, %v; want classic builder", mode, use, forced)
		}
	}
}

func TestSessionHelpers(t *testing.T) {
	if !headerTokenContains("keep-alive, Upgrade", "upgrade") || headerTokenContains("keep-alive", "upgrade") {
		t.Error("headerTokenContains")
	}
	k1, err := randomSharedKey(32)
	k2, _ := randomSharedKey(32)
	if err != nil || k1 == k2 || strings.ContainsAny(k1, "+/=") {
		t.Errorf("randomSharedKey = %q, %q, %v; want distinct raw base64url keys", k1, k2, err)
	}
	if s, _ := readSmall(bufio.NewReader(strings.NewReader("abc")), 10); s != "abc" {
		t.Errorf("readSmall short = %q", s)
	}
	if s, _ := readSmall(nil, 10); s != "" {
		t.Errorf("readSmall(nil) = %q", s)
	}
}
