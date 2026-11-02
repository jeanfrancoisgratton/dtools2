// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/cmd/cmd_test.go

package cmd

import (
	"dtools2/internal/testutil"
	"dtools2/rest"
	"dtools2/volumes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// execute runs dtools with args from a clean backend state and returns its stdout and error;
// a command that reported a failure (what makes Execute() exit 1) yields errCommandFailed
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	savedURI, savedRuntime, savedQuiet := rest.ConnectURI, Runtime, rest.QuietOutput
	t.Cleanup(func() {
		activeBackend = nil
		rest.ConnectURI, Runtime, rest.QuietOutput = savedURI, savedRuntime, savedQuiet
		rootCmd.SetArgs(nil)
		// Cobra keeps flag values across Execute() calls: an earlier --help would stick
		resetHelp := func(c *cobra.Command) {
			if f := c.Flags().Lookup("help"); f != nil {
				_ = f.Value.Set("false")
				f.Changed = false
			}
		}
		resetHelp(rootCmd)
		walk(rootCmd, nil, func(c *cobra.Command, _ []string) { resetHelp(c) })
	})
	activeBackend, commandFailed = nil, false
	rootCmd.SetArgs(args)
	var err error
	out := testutil.CaptureStdout(t, func() { err = rootCmd.Execute() })
	if err == nil && commandFailed {
		err = errCommandFailed
	}
	return out, err
}

var errCommandFailed = errors.New("command reported a failure")

// walk calls fn for every command of the tree, with the args that reach it from the root
func walk(c *cobra.Command, path []string, fn func(*cobra.Command, []string)) {
	for _, sub := range c.Commands() {
		p := append(append([]string{}, path...), sub.Name())
		fn(sub, p)
		walk(sub, p, fn)
	}
}

// Cobra only detects a local flag shorthand colliding with a persistent one when the command
// is parsed, and then panics: render the help of every command to catch that early
func TestEveryCommandParses(t *testing.T) {
	testutil.IsolateHome(t)
	walk(rootCmd, nil, func(c *cobra.Command, path []string) {
		t.Run(strings.Join(path, " "), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("dtools %s --help panicked: %v", strings.Join(path, " "), r)
				}
			}()
			out, err := execute(t, append(path, "--help")...)
			if err != nil || !strings.Contains(out, "Usage:") {
				t.Errorf("--help = %v, output %q", err, out)
			}
		})
	})
}

func TestVolumeCommandTree(t *testing.T) {
	want := map[string]bool{"lsv": false, "rmv": false, "prune": false, "create": false, "inspect": false, "backup": false, "restore": false}
	for _, c := range volumeCmd.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("volume %s is not registered", name)
		}
	}
	if !volumeCmd.HasAlias("vol") {
		t.Error("volume must have the 'vol' alias")
	}
	for _, c := range []*cobra.Command{volumeBackupCmd, volumeRestoreCmd} {
		for _, f := range []string{"image", "no-cleanup"} {
			if c.Flags().Lookup(f) == nil {
				t.Errorf("%s is missing --%s", c.Name(), f)
			}
		}
		if err := c.Args(c, []string{"only-one"}); err == nil {
			t.Errorf("%s must require exactly 2 arguments", c.Name())
		}
	}
}

func TestPersistentPreRunValidation(t *testing.T) {
	testutil.IsolateHome(t)
	out, _ := execute(t, "--runtime", "lxc", "lsc")
	if !strings.Contains(out, `Unknown --runtime "lxc"`) || activeBackend != nil {
		t.Errorf("unknown runtime output %q; want an error and no backend", out)
	}
	out, _ = execute(t, "--runtime", "containerd", "-H", "tcp://h:2375", "lsc")
	if !strings.Contains(out, "not supported with --runtime=containerd") {
		t.Errorf("containerd with -H output %q; want a rejection", out)
	}
}

func fakeDaemon(t *testing.T) *testutil.Daemon {
	t.Helper()
	testutil.IsolateHome(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /version", http.StatusOK, map[string]string{"ApiVersion": "1.50"})
	return d
}

func TestBackendActive(t *testing.T) {
	d := fakeDaemon(t)
	out, err := execute(t, "-H", d.URL(), "--runtime", "docker", "backend", "active")
	if err != nil || !strings.Contains(out, "docker/podman") {
		t.Errorf("backend active = %q, %v", out, err)
	}
}

// End to end through cobra: flags, backend init, VolumeService and the backup itself
func TestVolumeBackupCommand(t *testing.T) {
	d := fakeDaemon(t)
	d.Reply("GET /v1.50/containers/json", http.StatusOK, []any{})
	d.Reply("GET /v1.50/images/json", http.StatusOK, []map[string]any{{"RepoTags": []string{"busybox:1.36"}}})
	d.Reply("POST /v1.50/containers/create", http.StatusCreated, map[string]string{"Id": "temp1"})
	d.Reply("POST /v1.50/containers/{id}/start", http.StatusNoContent, nil)
	d.Handle("GET /v1.50/containers/{id}/archive", func(w http.ResponseWriter, _ *http.Request) { w.Write(make([]byte, 1024)) })
	saved := volumes.NoCleanup
	t.Cleanup(func() { volumes.NoCleanup, volumes.BackupImage = saved, "alpine:latest" })

	archive := filepath.Join(t.TempDir(), "pg")
	_, err := execute(t, "-H", d.URL(), "--runtime", "docker", "-q", "vol", "backup", "-i", "busybox:1.36", "-n", "pgdata", archive)
	if err != nil {
		t.Fatalf("vol backup: %v", err)
	}
	if fi, err := os.Stat(archive + ".tar"); err != nil || fi.Size() != 1024 {
		t.Errorf("archive %s.tar: %v; want the 1024-byte stream", archive, err)
	}
	if len(d.Calls(http.MethodDelete, "/v1.50/containers/temp1")) != 0 {
		t.Error("-n: the temp container must be kept")
	}
	if !strings.Contains(string(d.Calls(http.MethodPost, "/v1.50/containers/create")[0].Body), "busybox:1.36") {
		t.Error("-i: the temp container must use the requested image")
	}
}

// Like docker, a failing command must make dtools exit non-zero
func TestFailingCommandReturnsError(t *testing.T) {
	d := fakeDaemon(t)
	d.Reply("GET /v1.50/containers/{id}/json", http.StatusNotFound, map[string]string{"message": "No such container: ghost"})
	out, err := execute(t, "-H", d.URL(), "--runtime", "docker", "container", "inspect", "ghost")
	if !strings.Contains(out, "404") {
		t.Fatalf("output %q; want the inspect failure reported", out)
	}
	if err == nil {
		t.Error("inspect of a missing container failed, but dtools would exit 0")
	}

	d.Reply("GET /v1.50/containers/{id}/json", http.StatusOK, map[string]any{"Id": "abc", "Name": "/web"})
	if out, err := execute(t, "-H", d.URL(), "--runtime", "docker", "-q", "container", "inspect", "web"); err != nil {
		t.Errorf("a successful inspect must not be flagged as failed: %v (output %q)", err, out)
	}
}

// Commands that never talk to the daemon must not try to reach it
func TestLocalCommandsDoNotProbeTheDaemon(t *testing.T) {
	d := fakeDaemon(t)
	for _, args := range [][]string{{"version"}, {"blacklist", "lsb"}} {
		before := len(d.Requests())
		out, _ := execute(t, append([]string{"-H", d.URL(), "--runtime", "docker"}, args...)...)
		if n := len(d.Requests()) - before; n != 0 {
			t.Errorf("dtools %s sent %d request(s) to the daemon; want none (output %q)", strings.Join(args, " "), n, out)
		}
	}
}

// Two different root commands sharing a name (or alias) make cobra pick one arbitrarily:
// it sorts root commands with an unstable sort before matching
func TestRootNamesAreUnique(t *testing.T) {
	owner := map[string]*cobra.Command{}
	for _, c := range rootCmd.Commands() {
		for _, n := range append([]string{c.Name()}, c.Aliases...) {
			if prev, ok := owner[n]; ok && prev != c {
				t.Errorf("root-level name %q is used by both %q and %q", n, prev.Use, c.Use)
			}
			owner[n] = c
		}
	}
}

func TestInspectShortcuts(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"inspect", "web"}, "GET /v1.50/containers/web/json"},
		{[]string{"inspc", "web"}, "GET /v1.50/containers/web/json"},
		{[]string{"inspv", "data"}, "GET /v1.50/volumes/data"},
		{[]string{"insv", "data"}, "GET /v1.50/volumes/data"},
		{[]string{"volume", "inspect", "data"}, "GET /v1.50/volumes/data"},
		{[]string{"inspn", "appnet"}, "GET /v1.50/networks/appnet"},
		{[]string{"network", "inspect", "appnet"}, "GET /v1.50/networks/appnet"},
		{[]string{"inspi", "alpine:latest"}, "GET /v1.50/images/alpine:latest/json"},
		{[]string{"image", "inspect", "alpine:latest"}, "GET /v1.50/images/alpine:latest/json"},
	}
	for _, c := range cases {
		t.Run(strings.Join(c.args, " "), func(t *testing.T) {
			d := fakeDaemon(t)
			_, _ = execute(t, append([]string{"-H", d.URL(), "--runtime", "docker", "-q"}, c.args...)...)
			seq := d.Sequence()
			if len(seq) < 2 || seq[1] != c.want {
				t.Errorf("requests = %v; want %q after the version negotiation", seq, c.want)
			}
		})
	}
}
