// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/extras/extras_test.go

package extras

import (
	"bytes"
	"dtools2/internal/testutil"
	"dtools2/rest"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"text/tabwriter"
)

type fmtInner struct {
	Driver string
}

type fmtItem struct {
	Name    string            `json:"Name"`
	ID      string            `json:"Id"`
	Count   int               `json:"Count"`
	Size    uint64            `json:"Size"`
	Ratio   float64           `json:"Ratio"`
	Up      bool              `json:"Up"`
	Tags    []string          `json:"Tags"`
	Ports   []int             `json:"Ports"`
	Labels  map[string]string `json:"Labels"`
	IPAM    fmtInner          `json:"IPAM"`
	Ptr     *fmtInner         `json:"Ptr"`
	private string
}

// ContainerSummary mimics containers.ContainerSummary, whose Names get a special treatment
type ContainerSummary struct {
	Names []string
}

func TestParseFormatKeys(t *testing.T) {
	cases := map[string][]string{
		"":                    nil,
		"Name":                {"Name"},
		"{{.Name}}":           {"Name"},
		"{{ .Name }}":         {"Name"},
		".Name, Id ,,":        {"Name", "Id"},
		"{{.Name}},{{.Id}}":   {"Name", "Id"},
		"IPAM.Driver":         {"IPAM.Driver"},
		"{{.Name | upper}}":   {"Name"},
		"{{}}":                nil,
		"..Name":              {"Name"},
		"{{.Labels.com.foo}}": {"Labels.com.foo"},
	}
	for in, want := range cases {
		if got := parseFormatKeys(in); !reflect.DeepEqual(got, want) {
			t.Errorf("parseFormatKeys(%q) = %#v; want %#v", in, got, want)
		}
	}
}

func TestExtractFormatRows(t *testing.T) {
	items := []fmtItem{{
		Name: "web", ID: "abc", Count: 3, Size: 42, Ratio: 0.5, Up: true,
		Tags: []string{"a", "b"}, Ports: []int{80, 443}, Labels: map[string]string{"env": "prod"},
		IPAM: fmtInner{Driver: "default"}, private: "hidden",
	}}
	cases := map[string]string{
		"Name":        "web",
		"id":          "abc", // matched through the json tag, case-insensitively
		"Count":       "3",
		"Size":        "42",
		"Ratio":       "0.5",
		"Up":          "true",
		"Tags":        "a,b",
		"Ports":       "[80,443]",
		"Labels.env":  "prod",
		"Labels":      `{"env":"prod"}`,
		"IPAM.Driver": "default",
		"IPAM":        `{"Driver":"default"}`,
		"Ptr.Driver":  "", // nil pointer
		"Missing":     "", // unknown field: empty, like docker
		"private":     "", // unexported fields are not reachable
	}
	for format, want := range cases {
		rows, err := ExtractFormatRows(items, format)
		if err != nil {
			t.Fatalf("ExtractFormatRows(%q): %v", format, err)
		}
		if len(rows) != 1 || len(rows[0]) != 1 || rows[0][0] != want {
			t.Errorf("ExtractFormatRows(%q) = %v; want [[%q]]", format, rows, want)
		}
	}

	rows, _ := ExtractFormatRows(&items, "Name,Count")
	if !reflect.DeepEqual(rows, [][]string{{"web", "3"}}) {
		t.Errorf("multi-field rows from a pointer = %v", rows)
	}

	rows, _ = ExtractFormatRows([]*fmtItem{nil, {Name: "x"}}, "Name")
	if !reflect.DeepEqual(rows, [][]string{{"x"}}) {
		t.Errorf("nil elements must be skipped: %v", rows)
	}

	rows, _ = ExtractFormatRows([]ContainerSummary{{Names: []string{"/web", "/alias"}}, {}}, "Name")
	if !reflect.DeepEqual(rows, [][]string{{"web"}, {""}}) {
		t.Errorf("ContainerSummary names = %v; want the first name without its slash", rows)
	}
}

func TestExtractFormatRowsErrors(t *testing.T) {
	if rows, err := ExtractFormatRows([]fmtItem{{}}, "  "); rows != nil || err != nil {
		t.Errorf("empty format = %v, %v; want nil, nil", rows, err)
	}
	var nilSlice *[]fmtItem
	bad := []any{nil, nilSlice, fmtItem{}, []int{1}}
	for _, b := range bad {
		if _, err := ExtractFormatRows(b, "Name"); err == nil {
			t.Errorf("ExtractFormatRows(%T): want an error", b)
		}
	}
}

func TestPrintFormatRows(t *testing.T) {
	out := testutil.CaptureStdout(t, func() {
		_ = PrintFormatRows([][]string{{"a", "b"}, {"c", ""}})
	})
	if out != "a\tb\nc\t\n" {
		t.Errorf("PrintFormatRows output = %q", out)
	}
}

func TestMarshalAndPrintJSON(t *testing.T) {
	b, err := MarshalJSON(map[string]int{"a": 1})
	if err != nil || string(b) != "{\n  \"a\": 1\n}\n" {
		t.Errorf("MarshalJSON = %q, %v; want indented JSON ending with a newline", b, err)
	}
	if _, err := MarshalJSON(make(chan int)); err == nil {
		t.Error("MarshalJSON(chan): want an error")
	}
	out := testutil.CaptureStdout(t, func() { _ = PrintJSONBytes(b) })
	if out != string(b) {
		t.Errorf("PrintJSONBytes output = %q", out)
	}
}

func TestSend2File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.json")
	os.WriteFile(path, []byte("previous content"), 0o600)

	b, err := Send2File([]string{"x"}, path)
	if err != nil {
		t.Fatalf("Send2File: %v", err)
	}
	disk, _ := os.ReadFile(path)
	if !bytes.Equal(b, disk) || !strings.Contains(string(disk), `"x"`) {
		t.Errorf("file = %q; want the returned JSON %q", disk, b)
	}
	if fi, _ := os.Stat(path); fi.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v; want 0644", fi.Mode().Perm())
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("dir holds %d entries; the temp file must be cleaned up", len(entries))
	}

	if _, err := Send2File(make(chan int), path); err == nil {
		t.Error("Send2File(unmarshalable): want an error")
	}
	if err := WriteFileAtomic("  ", nil); err == nil {
		t.Error("WriteFileAtomic(empty name): want an error")
	}
	if err := WriteFileAtomic(filepath.Join(dir, "no", "such", "dir.json"), nil); err == nil {
		t.Error("WriteFileAtomic in a missing directory: want an error")
	}
}

func TestFormatSize(t *testing.T) {
	cases := map[int64]string{
		0:               "0 B",
		512:             "512 B",
		1024:            "1.0 KB",
		1536:            "1.5 KB",
		5 * 1024 * 1024: "5.0 MB",
		3 << 30:         "3.0 GB",
		2 << 40:         "2.0 TB",
		int64(7) << 50:  "7.0 PB",
	}
	for in, want := range cases {
		if got := FormatSize(in); got != want {
			t.Errorf("FormatSize(%d) = %q; want %q", in, got, want)
		}
	}
}

func TestFormatTimestamp(t *testing.T) {
	cases := map[string]string{
		"":                               "N/A",
		"0001-01-01T00:00:00Z":           "N/A",
		"2026-01-02T15:04:05Z":           "2026.01.02 15:04:05",
		"2026-01-02T15:04:05.123456789Z": "2026.01.02 15:04:05",
		"yesterday":                      "yesterday",
	}
	for in, want := range cases {
		if got := FormatTimestamp(in); got != want {
			t.Errorf("FormatTimestamp(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestFormatState(t *testing.T) {
	for _, s := range []string{"paused", "exited", "dead", "created", "weird"} {
		if got := FormatState(s, false); !strings.Contains(got, s) {
			t.Errorf("FormatState(%q) = %q; want it to contain the state", s, got)
		}
	}
	if got := FormatState("exited", true); !strings.Contains(got, "running") {
		t.Errorf("FormatState(running=true) = %q", got)
	}
}

func render(fn func(w *tabwriter.Writer)) string {
	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 1, 4, 2, ' ', 0)
	fn(w)
	w.Flush()
	return buf.String()
}

func TestFormatInspectHelpers(t *testing.T) {
	out := render(func(w *tabwriter.Writer) {
		FormatInspectSection(w, "info")
		FormatInspectField(w, "Name", "web")
		FormatInspectField(w, "Empty", "")
		FormatInspectFieldInt(w, "Count", 3)
		FormatInspectFieldInt64(w, "Big", 1<<40)
		FormatInspectFieldBool(w, "Up", true)
		FormatInspectList(w, "Tags", []string{"a", "b"})
		FormatInspectList(w, "NoTags", nil)
		FormatInspectMap(w, "Labels", map[string]string{"env": "prod"})
		FormatInspectMap(w, "NoLabels", nil)
	})
	for _, want := range []string{"INFO", "web", "3", "1099511627776", "true", "- a", "- b", "env: prod"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	for _, absent := range []string{"Empty", "NoTags", "NoLabels"} {
		if strings.Contains(out, absent) {
			t.Errorf("output should omit %q:\n%s", absent, out)
		}
	}
}

func TestPrintInspectData(t *testing.T) {
	vol := render(func(w *tabwriter.Writer) {
		PrintVolumeInspectData(w, map[string]any{
			"Name": "data", "Driver": "local", "Mountpoint": "/var/lib/docker/volumes/data", "Scope": "local",
			"Options": map[string]string{"o": "bind"}, "Labels": map[string]string{"team": "ops"},
			"UsageData": map[string]int64{"Size": 2048, "RefCount": 1},
		})
	})
	for _, want := range []string{"data", "/var/lib/docker/volumes/data", "bind", "ops", "2.0 KB"} {
		if !strings.Contains(vol, want) {
			t.Errorf("volume output missing %q:\n%s", want, vol)
		}
	}

	net := render(func(w *tabwriter.Writer) {
		PrintNetworkInspectData(w, map[string]any{"Name": "appnet", "Id": "net1", "Driver": "bridge"})
	})
	if !strings.Contains(net, "appnet") || !strings.Contains(net, "net1") {
		t.Errorf("network output:\n%s", net)
	}

	img := render(func(w *tabwriter.Writer) {
		PrintImageInspectData(w, map[string]any{"Id": "sha256:abc", "RepoTags": []string{"alpine:latest"}, "Size": 8 * 1024 * 1024})
	})
	if !strings.Contains(img, "alpine:latest") {
		t.Errorf("image output:\n%s", img)
	}

	ctr := render(func(w *tabwriter.Writer) {
		PrintContainerInspectData(w, "abc123", "/web", "2026-01-02T15:04:05Z",
			map[string]any{"Status": "running", "Running": true}, "alpine:latest", "linux", "overlay2", 10, 20,
			map[string]any{"Env": []string{"A=1"}}, map[string]any{}, map[string]any{}, []any{})
	})
	for _, want := range []string{"abc123", "web", "2026.01.02 15:04:05", "alpine:latest", "overlay2"} {
		if !strings.Contains(ctr, want) {
			t.Errorf("container output missing %q:\n%s", want, ctr)
		}
	}

	// nil payloads print nothing
	if out := render(func(w *tabwriter.Writer) {
		PrintVolumeInspectData(w, nil)
		PrintNetworkInspectData(w, nil)
		PrintImageInspectData(w, nil)
	}); out != "" {
		t.Errorf("nil payloads printed %q", out)
	}
}

func muxFrame(stream byte, payload string) []byte {
	hdr := make([]byte, 8)
	hdr[0] = stream
	binary.BigEndian.PutUint32(hdr[4:], uint32(len(payload)))
	return append(hdr, payload...)
}

func TestLooksLikeStdCopyMux(t *testing.T) {
	cases := []struct {
		name string
		hdr  []byte
		want bool
	}{
		{"stdout frame", muxFrame(1, "hello")[:8], true},
		{"stderr frame", muxFrame(2, "x")[:8], true},
		{"zero-size frame", []byte{1, 0, 0, 0, 0, 0, 0, 0}, true},
		{"plain text", []byte("hello wo"), false},
		{"bad padding", []byte{1, 1, 0, 0, 0, 0, 0, 5}, false},
		{"too short", []byte{1, 0, 0}, false},
		{"huge frame", []byte{1, 0, 0, 0, 0xff, 0xff, 0xff, 0xff}, false},
	}
	for _, c := range cases {
		if got := looksLikeStdCopyMux(c.hdr); got != c.want {
			t.Errorf("%s: looksLikeStdCopyMux = %v; want %v", c.name, got, c.want)
		}
	}
}

func resetLogFlags(t *testing.T) {
	t.Helper()
	follow, ts, tail := LogFollow, LogTimestamps, LogTail
	t.Cleanup(func() { LogFollow, LogTimestamps, LogTail = follow, ts, tail })
}

func TestLogs(t *testing.T) {
	resetLogFlags(t)
	d := testutil.NewDaemon(t)
	d.Handle("GET /containers/{id}/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("id") == "tty" {
			w.Write([]byte("raw tty output\n"))
			return
		}
		w.Write(muxFrame(1, "from stdout\n"))
		w.Write(muxFrame(2, "from stderr\n"))
	})

	LogFollow, LogTimestamps, LogTail = true, true, 50
	out := testutil.CaptureStdout(t, func() {
		if err := Logs(d.Client(), "web"); err != nil {
			t.Errorf("Logs: %v", err)
		}
	})
	if out != "from stdout\n" {
		t.Errorf("stdout = %q; want only the demultiplexed stdout frame", out)
	}
	q := d.Calls(http.MethodGet, "/containers/web/logs")[0].Query
	if q.Get("follow") != "true" || q.Get("timestamps") != "true" || q.Get("tail") != "50" || q.Get("stdout") != "true" || q.Get("stderr") != "true" {
		t.Errorf("logs query = %v", q)
	}

	LogFollow, LogTimestamps, LogTail = false, false, -1
	out = testutil.CaptureStdout(t, func() { _ = Logs(d.Client(), "tty") })
	if out != "raw tty output\n" {
		t.Errorf("tty stdout = %q; want the raw stream", out)
	}
	q = d.Calls(http.MethodGet, "/containers/tty/logs")[0].Query
	if q.Get("tail") != "all" || q.Has("follow") || q.Has("timestamps") {
		t.Errorf("default logs query = %v; want tail=all and no follow/timestamps", q)
	}
}

func TestLogsError(t *testing.T) {
	resetLogFlags(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /containers/{id}/logs", http.StatusNotFound, map[string]string{"message": "no such container"})
	if err := Logs(d.Client(), "ghost"); err == nil || !strings.Contains(err.Message, "no such container") {
		t.Errorf("Logs(ghost) = %v; want the daemon message", err)
	}
	d.Reply("GET /containers/{id}/logs", http.StatusInternalServerError, nil)
	if err := Logs(d.Client(), "ghost"); err == nil || !strings.Contains(err.Message, "500") {
		t.Errorf("Logs on an empty 500 = %v; want the status", err)
	}
}

func resetExecFlags(t *testing.T) {
	t.Helper()
	i, tty, u := Interactive, AllocateTTY, User
	t.Cleanup(func() { Interactive, AllocateTTY, User = i, tty, u })
}

func TestCreateExec(t *testing.T) {
	resetExecFlags(t)
	d := testutil.NewDaemon(t)
	d.Reply("POST /containers/{id}/exec", http.StatusCreated, map[string]string{"Id": "exec1"})

	Interactive, AllocateTTY, User = true, false, "nobody"
	id, err := createExec(d.Client(), "web", []string{"ls", "-l"})
	if err != nil || id != "exec1" {
		t.Fatalf("createExec = %q, %v", id, err)
	}
	var req ExecCreateRequest
	json.Unmarshal(d.Calls(http.MethodPost, "/containers/web/exec")[0].Body, &req)
	want := ExecCreateRequest{AttachStdin: true, AttachStdout: true, AttachStderr: true, Cmd: []string{"ls", "-l"}, User: "nobody"}
	if !reflect.DeepEqual(req, want) {
		t.Errorf("exec request = %+v; want %+v", req, want)
	}

	d.Reply("POST /containers/{id}/exec", http.StatusCreated, map[string]string{"Id": ""})
	if _, err := createExec(d.Client(), "web", []string{"ls"}); err == nil {
		t.Error("an empty exec ID: want an error")
	}
	d.Reply("POST /containers/{id}/exec", http.StatusConflict, map[string]string{"message": "container is not running"})
	if _, err := createExec(d.Client(), "web", []string{"ls"}); err == nil || !strings.Contains(err.Message, "not running") {
		t.Errorf("createExec on HTTP 409 = %v; want the daemon message", err)
	}
}

func TestInspectExitCode(t *testing.T) {
	d := testutil.NewDaemon(t)
	d.Reply("GET /exec/exec1/json", http.StatusOK, map[string]any{"Running": false, "ExitCode": 42})
	if code, err := inspectExitCode(d.Client(), "exec1"); err != nil || code != 42 {
		t.Errorf("inspectExitCode = %d, %v; want 42", code, err)
	}
	if code, err := inspectExitCode(d.Client(), "ghost"); err == nil || code != 1 {
		t.Errorf("inspectExitCode(unknown) = %d, %v; want 1 and an error", code, err)
	}
}

func TestRunValidation(t *testing.T) {
	resetExecFlags(t)
	d := testutil.NewDaemon(t)
	if code, err := Run(d.Client(), "web", nil); err == nil || code != 1 {
		t.Errorf("Run without a command = %d, %v; want 1 and an error", code, err)
	}
	// Under `go test`, stdin is not a terminal: -it must be refused like the docker CLI does
	Interactive, AllocateTTY = true, true
	if _, err := Run(d.Client(), "web", []string{"sh"}); err == nil || !strings.Contains(err.Title, "TTY") {
		t.Errorf("Run -it without a TTY = %v; want a TTY error", err)
	}
	if len(d.Requests()) != 0 {
		t.Errorf("requests %v; want none", d.Sequence())
	}
}

func TestErrorsIsWouldBlock(t *testing.T) {
	if !errorsIsWouldBlock(syscall.EAGAIN) || errorsIsWouldBlock(syscall.EPERM) {
		t.Error("errorsIsWouldBlock must match EAGAIN only")
	}
}

func TestGetDefaultRegistry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "defaultRegistry.json")
	os.WriteFile(path, []byte(`{"RegistryName":"nexus:9820"}`), 0o600)
	if got, err := GetDefaultRegistry(path); err != nil || got != "nexus:9820" {
		t.Errorf("GetDefaultRegistry = %q, %v", got, err)
	}
}

func TestCopyStdinFromPipe(t *testing.T) {
	r, w, _ := os.Pipe()
	orig := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = orig }()

	w.Write([]byte("piped input"))
	w.Close()
	var buf bytes.Buffer
	ctx := rest.Context
	defer func() { rest.Context = ctx }()
	CopyStdin(&buf, make(chan struct{}))
	if buf.String() != "piped input" {
		t.Errorf("CopyStdin copied %q", buf.String())
	}
}
