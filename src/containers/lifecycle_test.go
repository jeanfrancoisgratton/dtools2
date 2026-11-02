// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original filename: src/containers/lifecycle_test.go

package containers

import (
	"dtools2/internal/testutil"
	"dtools2/rest"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// fakeContainer is a container in the stateful fake daemon below
type fakeContainer struct {
	id, name, state string
	// killPending mimics the daemon replying to /kill before the container has exited:
	// it keeps reporting "running" until somebody waits on it
	killPending bool
}

// fakeEngine is a minimal stateful container engine on top of testutil.Daemon
type fakeEngine struct {
	*testutil.Daemon
	mu         sync.Mutex
	containers []*fakeContainer
}

func newFakeEngine(t *testing.T, cs ...*fakeContainer) *fakeEngine {
	t.Helper()
	resetGlobals(t)
	e := &fakeEngine{Daemon: testutil.NewDaemon(t), containers: cs}

	e.Handle("GET /containers/json", func(w http.ResponseWriter, r *http.Request) {
		e.mu.Lock()
		defer e.mu.Unlock()
		all := r.URL.Query().Get("all") == "true"
		out := []map[string]any{}
		for _, c := range e.containers {
			if !all && c.state != "running" {
				continue
			}
			out = append(out, testutil.Container(c.id, c.name, c.state))
		}
		testutil.JSON(w, http.StatusOK, out)
	})
	e.Handle("POST /containers/{id}/stop", e.transition("running", "exited", false))
	e.Handle("POST /containers/{id}/kill", e.transition("running", "exited", true))
	e.Handle("POST /containers/{id}/start", e.transition("exited", "running", false))
	e.Handle("POST /containers/{id}/wait", func(w http.ResponseWriter, r *http.Request) {
		c := e.find(r.PathValue("id"))
		if c == nil {
			testutil.JSON(w, http.StatusNotFound, map[string]string{"message": "no such container"})
			return
		}
		e.mu.Lock()
		if c.killPending {
			c.killPending, c.state = false, "exited"
		}
		e.mu.Unlock()
		testutil.JSON(w, http.StatusOK, map[string]int{"StatusCode": 137})
	})
	return e
}

// find looks a container up by ID or name, like the daemon does
func (e *fakeEngine) find(ref string) *fakeContainer {
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, c := range e.containers {
		if c.id == ref || c.name == ref {
			return c
		}
	}
	return nil
}

func (e *fakeEngine) transition(from, to string, deferred bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := e.find(r.PathValue("id"))
		if c == nil {
			testutil.JSON(w, http.StatusNotFound, map[string]string{"message": "no such container"})
			return
		}
		e.mu.Lock()
		defer e.mu.Unlock()
		if c.state != from {
			testutil.JSON(w, http.StatusNotModified, nil)
			return
		}
		if deferred {
			c.killPending = true
		} else {
			c.state = to
		}
		testutil.JSON(w, http.StatusNoContent, nil)
	}
}

func (e *fakeEngine) state(name string) string {
	c := e.find(name)
	e.mu.Lock()
	defer e.mu.Unlock()
	if c.killPending {
		return "running"
	}
	return c.state
}

// resetGlobals restores the package-level flags mutated by the functions under test
func resetGlobals(t *testing.T) {
	t.Helper()
	saved := struct {
		running, extended, size, rmBL, rmVols, kill, force bool
		timeout                                            int
		quiet                                              bool
	}{OnlyRunningContainers, ExtendedContainerInfo, DisplaySizeValues, RemoveBlacklisted, RemoveUnamedVolumes, KillSwitch, ForceRemoveContainer, StopTimeout, rest.QuietOutput}
	rest.QuietOutput = true
	t.Cleanup(func() {
		OnlyRunningContainers, ExtendedContainerInfo, DisplaySizeValues = saved.running, saved.extended, saved.size
		RemoveBlacklisted, RemoveUnamedVolumes, KillSwitch, ForceRemoveContainer = saved.rmBL, saved.rmVols, saved.kill, saved.force
		StopTimeout, rest.QuietOutput = saved.timeout, saved.quiet
	})
}

func TestListContainersQuery(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "running"}, &fakeContainer{id: "bbb222", name: "db", state: "exited"})

	OnlyRunningContainers = true
	cs, err := ListContainers(e.Client(), false)
	if err != nil {
		t.Fatalf("ListContainers: %v", err)
	}
	if len(cs) != 1 || cs[0].Names[0] != "/web" {
		t.Errorf("running-only list = %+v; want only web", cs)
	}

	OnlyRunningContainers = false
	cs, _ = ListContainers(e.Client(), false)
	if len(cs) != 2 {
		t.Errorf("full list has %d containers; want 2", len(cs))
	}

	q := e.Calls(http.MethodGet, "/containers/json")[0].Query
	if q.Get("all") != "false" || q.Get("size") != "false" {
		t.Errorf("query = %v; want all=false size=false", q)
	}
}

func TestListContainersErrors(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /containers/json", http.StatusInternalServerError, map[string]string{"message": "boom"})
	if _, err := ListContainers(d.Client(), false); err == nil {
		t.Error("ListContainers on HTTP 500: want an error")
	}

	d2 := testutil.NewDaemon(t)
	d2.Handle("GET /containers/json", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("{not json")) })
	if _, err := ListContainers(d2.Client(), false); err == nil {
		t.Error("ListContainers on invalid JSON: want an error")
	}
}

func TestName2IDAndID2Name(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "running"})

	if id, err := Name2ID(e.Client(), "web"); err != nil || id != "aaa111" {
		t.Errorf("Name2ID(web) = %q, %v; want aaa111", id, err)
	}
	if _, err := Name2ID(e.Client(), "nope"); err == nil {
		t.Error("Name2ID(unknown): want an error")
	}
	if name, err := ID2Name(e.Client(), "aaa111"); err != nil || name != "web" {
		t.Errorf("ID2Name(aaa111) = %q, %v; want web", name, err)
	}
	if _, err := ID2Name(e.Client(), "zzz999"); err == nil {
		t.Error("ID2Name(unknown): want an error")
	}

	empty := newFakeEngine(t)
	if _, err := Name2ID(empty.Client(), "web"); err == nil {
		t.Error("Name2ID with no containers: want an error")
	}
}

func TestStartContainers(t *testing.T) {
	e := newFakeEngine(t,
		&fakeContainer{id: "aaa111", name: "web", state: "exited"},
		&fakeContainer{id: "bbb222", name: "db", state: "running"},
	)
	rest.QuietOutput = false

	var err *ce.CustomError
	out := testutil.CaptureStdout(t, func() { err = StartContainers(e.Client(), []string{"web", "db"}) })
	if err != nil {
		t.Fatalf("StartContainers: %v", err)
	}
	if e.state("web") != "running" {
		t.Error("web was not started")
	}
	if len(e.Calls(http.MethodPost, "/containers/bbb222/start")) != 0 {
		t.Error("db is already running and should not have been started")
	}
	if !strings.Contains(out, "db is already running") {
		t.Errorf("output %q: want a warning about db already running", out)
	}
}

func TestStartContainersQuietSuppressesWarning(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "bbb222", name: "db", state: "running"})
	out := testutil.CaptureStdout(t, func() { _ = StartContainers(e.Client(), []string{"db"}) })
	if out != "" {
		t.Errorf("quiet output = %q; want nothing", out)
	}
}

func TestStartContainersUnknown(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "exited"})
	err := StartContainers(e.Client(), []string{"web", "ghost", "phantom"})
	if err == nil {
		t.Fatal("StartContainers with unknown names: want an error")
	}
	if !strings.Contains(err.Message, "ghost") || !strings.Contains(err.Message, "phantom") {
		t.Errorf("error %q: want both unknown names listed", err.Message)
	}
	if e.state("web") != "running" {
		t.Error("the known container should still have been started")
	}
}

func TestStartContainersByID(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "abcdef123456", name: "web", state: "exited"})
	if err := StartContainers(e.Client(), []string{"abcdef123456"}); err != nil {
		t.Fatalf("StartContainers(id): %v", err)
	}
	if len(e.Calls(http.MethodPost, "/containers/abcdef123456/start")) != 1 {
		t.Errorf("requests %v: want a start by ID", e.Sequence())
	}
}

func TestStartContainersEmpty(t *testing.T) {
	e := newFakeEngine(t)
	if err := StartContainers(e.Client(), nil); err != nil {
		t.Errorf("StartContainers(nil) = %v; want nil", err)
	}
	if len(e.Requests()) != 0 {
		t.Errorf("StartContainers(nil) sent %v; want no request", e.Sequence())
	}
}

func TestStartAllOnlyTargetsStopped(t *testing.T) {
	e := newFakeEngine(t,
		&fakeContainer{id: "aaa111", name: "web", state: "exited"},
		&fakeContainer{id: "bbb222", name: "db", state: "running"},
	)
	rest.QuietOutput = false
	out := testutil.CaptureStdout(t, func() {
		if err := StartAllContainers(e.Client()); err != nil {
			t.Errorf("StartAllContainers: %v", err)
		}
	})
	if e.state("web") != "running" {
		t.Error("web was not started")
	}
	if strings.Contains(out, "already running") {
		t.Errorf("startall output %q: running containers should not be warned about", out)
	}
}

func TestStopContainersSequential(t *testing.T) {
	e := newFakeEngine(t,
		&fakeContainer{id: "aaa111", name: "web", state: "running"},
		&fakeContainer{id: "bbb222", name: "db", state: "exited"},
	)
	StopTimeout = 7
	if err := StopContainers(e.Client(), []string{"web", "db"}); err != nil {
		t.Fatalf("StopContainers: %v", err)
	}
	calls := e.Calls(http.MethodPost, "/containers/aaa111/stop")
	if len(calls) != 1 || calls[0].Query.Get("t") != "7" {
		t.Errorf("stop calls %+v; want one with t=7", calls)
	}
	if len(e.Calls(http.MethodPost, "/containers/bbb222/stop")) != 0 {
		t.Error("an exited container should not be stopped")
	}
}

func TestStopContainersConcurrent(t *testing.T) {
	e := newFakeEngine(t,
		&fakeContainer{id: "aaa111", name: "web", state: "running"},
		&fakeContainer{id: "bbb222", name: "db", state: "running"},
	)
	StopTimeout = 0
	if err := StopContainers(e.Client(), []string{"web", "db"}); err != nil {
		t.Fatalf("StopContainers: %v", err)
	}
	if e.state("web") != "exited" || e.state("db") != "exited" {
		t.Error("both containers should be stopped")
	}
}

func TestStopContainersNothingToStop(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "exited"})
	rest.QuietOutput = false
	out := testutil.CaptureStdout(t, func() { _ = StopContainers(e.Client(), []string{"web"}) })
	if !strings.Contains(out, "No containers to stop") {
		t.Errorf("output %q: want 'No containers to stop'", out)
	}
}

func TestStopErrorStatus(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /containers/json", http.StatusOK, []any{testutil.Container("aaa111", "web", "running")})
	d.Reply("POST /containers/{id}/stop", http.StatusInternalServerError, map[string]string{"message": "boom"})
	if err := StopContainers(d.Client(), []string{"web"}); err == nil {
		t.Error("StopContainers on HTTP 500: want an error")
	}
}

func TestKillContainers(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "running"})
	if err := KillContainers(e.Client(), []string{"web"}); err != nil {
		t.Fatalf("KillContainers: %v", err)
	}
	if len(e.Calls(http.MethodPost, "/containers/aaa111/kill")) != 1 {
		t.Errorf("requests %v: want a POST /kill", e.Sequence())
	}
}

func TestKillAllContainers(t *testing.T) {
	e := newFakeEngine(t,
		&fakeContainer{id: "aaa111", name: "web", state: "running"},
		&fakeContainer{id: "bbb222", name: "db", state: "exited"},
	)
	if err := KillAllContainers(e.Client()); err != nil {
		t.Fatalf("KillAllContainers: %v", err)
	}
	if len(e.Calls(http.MethodPost, "/containers/aaa111/kill")) != 1 || len(e.Calls(http.MethodPost, "/containers/bbb222/kill")) != 0 {
		t.Errorf("requests %v: want only the running container killed", e.Sequence())
	}
}

// Regression test: the daemon replies to /kill before the container has exited,
// so restart -k used to find it still running and skip the start
func TestRestartWithKillWaitsForExit(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "running"})
	KillSwitch = true
	if err := RestartContainers(e.Client(), []string{"web"}); err != nil {
		t.Fatalf("RestartContainers: %v", err)
	}
	if len(e.Calls(http.MethodPost, "/containers/aaa111/start")) != 1 {
		t.Fatalf("requests %v: the killed container was never started", e.Sequence())
	}
	wait := e.Calls(http.MethodPost, "/containers/web/wait")
	if len(wait) != 1 || wait[0].Query.Get("condition") != "not-running" {
		t.Errorf("wait calls %+v; want one with condition=not-running", wait)
	}
	if e.state("web") != "running" {
		t.Errorf("web state = %s; want running", e.state("web"))
	}
}

func TestRestartWithStop(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "running"})
	if err := RestartContainers(e.Client(), []string{"web"}); err != nil {
		t.Fatalf("RestartContainers: %v", err)
	}
	if len(e.Calls(http.MethodPost, "/containers/aaa111/stop")) != 1 || len(e.Calls(http.MethodPost, "/containers/aaa111/start")) != 1 {
		t.Errorf("requests %v: want a stop then a start", e.Sequence())
	}
}

func TestRestartAllOnlyRestartsRunning(t *testing.T) {
	for _, kill := range []bool{false, true} {
		e := newFakeEngine(t,
			&fakeContainer{id: "aaa111", name: "web", state: "running"},
			&fakeContainer{id: "bbb222", name: "db", state: "exited"},
		)
		KillSwitch = kill
		if err := RestartAllContainers(e.Client()); err != nil {
			t.Fatalf("RestartAllContainers(kill=%v): %v", kill, err)
		}
		if e.state("web") != "running" {
			t.Errorf("kill=%v: web was not restarted", kill)
		}
		if e.state("db") != "exited" {
			t.Errorf("kill=%v: db was stopped beforehand and must stay stopped", kill)
		}
	}
}

func TestRestartAllNothingRunning(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "bbb222", name: "db", state: "exited"})
	rest.QuietOutput = false
	out := testutil.CaptureStdout(t, func() { _ = RestartAllContainers(e.Client()) })
	if !strings.Contains(out, "RESTARTALL is thus un-needed") {
		t.Errorf("output %q: want the 'nothing running' note", out)
	}
	if len(e.Calls(http.MethodPost, "/containers/bbb222/start")) != 0 {
		t.Error("a stopped container must not be started")
	}
}

func TestWaitNotRunningError(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("POST /containers/{id}/wait", http.StatusNotFound, map[string]string{"message": "no such container"})
	if err := waitNotRunning(d.Client(), []string{"ghost"}); err == nil {
		t.Error("waitNotRunning on HTTP 404: want an error")
	}
}

func TestPauseUnpause(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "running"})
	e.Reply("POST /containers/{id}/pause", http.StatusNoContent, nil)
	e.Reply("POST /containers/{id}/unpause", http.StatusNoContent, nil)

	if err := PauseContainer(e.Client(), []string{"web"}); err != nil {
		t.Errorf("PauseContainer: %v", err)
	}
	if err := UnpauseContainer(e.Client(), []string{"web"}); err != nil {
		t.Errorf("UnpauseContainer: %v", err)
	}
	if len(e.Calls(http.MethodPost, "/containers/aaa111/pause")) != 1 || len(e.Calls(http.MethodPost, "/containers/aaa111/unpause")) != 1 {
		t.Errorf("requests %v: want POST /containers/aaa111/pause and /unpause", e.Sequence())
	}
}

func TestRenameContainer(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "running"})
	e.Handle("POST /containers/{id}/rename", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") == "taken" {
			testutil.JSON(w, http.StatusConflict, map[string]string{"message": "name in use"})
			return
		}
		testutil.JSON(w, http.StatusNoContent, nil)
	})

	if err := RenameContainer(e.Client(), "web", "web2"); err != nil {
		t.Errorf("RenameContainer: %v", err)
	}
	if got := e.Calls(http.MethodPost, "/containers/aaa111/rename")[0].Query.Get("name"); got != "web2" {
		t.Errorf("rename name = %q; want web2", got)
	}
	if err := RenameContainer(e.Client(), "web", "taken"); err == nil {
		t.Error("RenameContainer to a taken name: want an error")
	}
	if err := RenameContainer(e.Client(), "ghost", "x"); err == nil {
		t.Error("RenameContainer of an unknown container: want an error")
	}
}

func TestRemoveContainer(t *testing.T) {
	cfgDir := testutil.IsolateHome(t)
	if err := os.WriteFile(filepath.Join(cfgDir, "blacklist.json"), []byte(`{"Containers":["db"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	e := newFakeEngine(t,
		&fakeContainer{id: "aaa111", name: "web", state: "exited"},
		&fakeContainer{id: "bbb222", name: "db", state: "exited"},
	)
	e.Reply("DELETE /containers/{id}", http.StatusNoContent, nil)

	ForceRemoveContainer, RemoveUnamedVolumes = true, false
	if err := RemoveContainer(e.Client(), []string{"web", "db"}); err != nil {
		t.Fatalf("RemoveContainer: %v", err)
	}
	del := e.Calls(http.MethodDelete, "/containers/aaa111")
	if len(del) != 1 || del[0].Query.Get("force") != "true" || del[0].Query.Get("v") != "false" {
		t.Errorf("DELETE calls %+v; want one with force=true v=false", del)
	}
	if len(e.Calls(http.MethodDelete, "/containers/bbb222")) != 0 {
		t.Error("a blacklisted container must not be removed without -B")
	}

	RemoveBlacklisted = true
	if err := RemoveContainer(e.Client(), []string{"db"}); err != nil {
		t.Fatalf("RemoveContainer -B: %v", err)
	}
	if len(e.Calls(http.MethodDelete, "/containers/bbb222")) != 1 {
		t.Error("-B should remove a blacklisted container")
	}
}

func TestRemoveContainerConflict(t *testing.T) {
	testutil.IsolateHome(t)
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "running"})
	e.Reply("DELETE /containers/{id}", http.StatusConflict, map[string]string{"message": "container is running"})
	err := RemoveContainer(e.Client(), []string{"web"})
	if err == nil || !strings.Contains(err.Title, "409") {
		t.Errorf("RemoveContainer on 409 = %v; want a conflict error", err)
	}
}

func TestInspectContainer(t *testing.T) {
	resetGlobals(t)
	d := testutil.NewDaemon(t)
	d.Reply("GET /containers/{id}/json", http.StatusOK, map[string]any{"Id": "aaa111", "Name": "/web"})

	var err *ce.CustomError
	out := testutil.CaptureStdout(t, func() { _, err = InspectContainer(d.Client(), "web") })
	if err != nil {
		t.Fatalf("InspectContainer: %v", err)
	}
	// In quiet mode, the raw JSON is printed as-is
	if !strings.Contains(out, `"Id":"aaa111"`) {
		t.Errorf("quiet output %q: want the raw JSON", out)
	}
	if q := d.Calls(http.MethodGet, "/containers/web/json")[0].Query; q.Get("size") != "true" {
		t.Errorf("query = %v; want size=true", q)
	}

	d.Reply("GET /containers/ghost/json", http.StatusNotFound, map[string]string{"message": "no such container"})
	if _, err := InspectContainer(d.Client(), "ghost"); err == nil || !strings.Contains(err.Title, "404") {
		t.Errorf("InspectContainer(ghost) = %v; want an HTTP 404 error", err)
	}
}

func TestInfoContainersRestoresFlags(t *testing.T) {
	e := newFakeEngine(t, &fakeContainer{id: "aaa111", name: "web", state: "running"})
	OnlyRunningContainers, ExtendedContainerInfo, DisplaySizeValues = true, false, false
	_ = testutil.CaptureStdout(t, func() {
		if err := InfoContainers(e.Client(), "web"); err != nil {
			t.Errorf("InfoContainers: %v", err)
		}
	})
	got := []bool{OnlyRunningContainers, ExtendedContainerInfo, DisplaySizeValues}
	if want := []bool{true, false, false}; !reflect.DeepEqual(got, want) {
		t.Errorf("flags after InfoContainers = %v; want %v (restored)", got, want)
	}
	if q := e.Calls(http.MethodGet, "/containers/json")[0].Query; q.Get("size") != "true" {
		t.Errorf("InfoContainers query = %v; want size=true", q)
	}
}
