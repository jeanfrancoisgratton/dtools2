// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 22:00
// Original filename: src/extras/run.go

package run

import (
	"bytes"
	"dtools2/extras"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"dtools2/rest"

	"github.com/docker/docker/pkg/stdcopy"
	ce "github.com/jeanfrancoisgratton/customError/v3"
	mobyterm "github.com/moby/term"
	xterm "golang.org/x/term"
)

// RunContainer emulates `docker run` (subset).
//
// Behaviour covered:
//   - create + start
//   - attach when not detached
//   - optional: -t, -i, -u, -w, -e, -p, -v, --name, --rm, --network, --entrypoint, --hostname
//   - auto-pull when image is missing
//
// Return values:
//   - exitCode: container process exit code (attached mode) or 0 (detached)
//   - containerID: non-empty in detached mode
//
// NOTE: This file relies on existing helpers in the same package (extras), already present in your tree:
//   - stringsTrim(string) string
//   - copyStdin(dst io.Writer, stop <-chan struct{})
func RunContainer(client *rest.Client, image string, cmd []string) (exitCode int, containerID string, errCode *ce.CustomError) {
	if image == "" {
		return 1, "", &ce.CustomError{Title: "Missing image", Message: "no image specified"}
	}

	// Docker CLI errors when -it is used but stdin isn't a TTY.
	if RunTTY && RunInteractive {
		if !xterm.IsTerminal(int(os.Stdin.Fd())) {
			return 1, "", &ce.CustomError{Title: "The input device is not a TTY", Message: "cannot allocate a TTY with non-terminal stdin"}
		}
	}

	// Create (with auto-pull on missing image).
	id, cerr := createContainerWithAutoPull(client, image, cmd)
	if cerr != nil {
		return 1, "", cerr
	}

	if RunDetach {
		if cerr := startContainer(client, id); cerr != nil {
			return 1, "", cerr
		}
		return 0, id, nil
	}

	// Attach BEFORE start (docker behaviour).
	hj, cerr := attachContainer(client, id)
	if cerr != nil {
		return 1, "", cerr
	}
	conn := hj.Conn
	reader := hj.Reader

	// Ensure connection is closed on context cancellation.
	done := make(chan struct{})
	go func() {
		select {
		case <-rest.Context.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	defer func() {
		close(done)
		_ = conn.Close()
	}()

	// Terminal raw mode + resize when allocating TTY.
	var restoreTerm func()
	if RunTTY && RunInteractive && xterm.IsTerminal(int(os.Stdin.Fd())) {
		oldState, e := xterm.MakeRaw(int(os.Stdin.Fd()))
		if e == nil {
			restoreTerm = func() { _ = xterm.Restore(int(os.Stdin.Fd()), oldState) }
		}
	}
	if restoreTerm != nil {
		defer restoreTerm()
	}

	if RunTTY {
		setupContainerResizeHandler(client, id)
	}

	// Start container.
	if cerr := startContainer(client, id); cerr != nil {
		return 1, "", cerr
	}

	// Wait in parallel for exit code.
	type wr struct {
		code int
		err  *ce.CustomError
	}
	waitCh := make(chan wr, 1)
	go func() {
		code, werr := waitContainerExit(client, id)
		waitCh <- wr{code: code, err: werr}
	}()

	// Optional signal proxying in non-TTY mode. In TTY mode, Ctrl+C is sent
	// as bytes through the pty (raw stdin), so we mostly avoid double-sending.
	stopSigProxy := make(chan struct{})
	if !RunTTY {
		go proxySignalsToContainer(client, id, stopSigProxy)
	}
	defer close(stopSigProxy)

	// Stream.
	var wg sync.WaitGroup
	stdinStop := make(chan struct{})

	if RunInteractive {
		wg.Add(1)
		go func() {
			defer wg.Done()
			extras.CopyStdin(conn, stdinStop)
			if cw, ok := conn.(interface{ CloseWrite() error }); ok {
				_ = cw.CloseWrite()
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if RunTTY {
			_, _ = io.Copy(os.Stdout, reader)
		} else {
			_, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
		}
		close(stdinStop)
	}()

	wg.Wait()

	r := <-waitCh
	if r.err != nil {
		return 1, "", r.err
	}

	return r.code, "", nil
}

func createContainerWithAutoPull(client *rest.Client, image string, cmd []string) (string, *ce.CustomError) {
	id, missing, cerr := createContainer(client, image, cmd)
	if cerr == nil {
		return id, nil
	}
	if !missing {
		return "", cerr
	}

	// Auto-pull image then retry.
	if err := pullImageViaDaemon(client, image); err != nil {
		return "", &ce.CustomError{Title: "Unable to pull image", Message: err.Error()}
	}
	id, _, cerr = createContainer(client, image, cmd)
	if cerr != nil {
		return "", cerr
	}
	return id, nil
}

// createContainer returns (id, imageMissing, customError).
func createContainer(client *rest.Client, image string, cmd []string) (string, bool, *ce.CustomError) {
	req := ContainerCreateRequest{
		Image: image,
		Cmd:   nil,
		// IO
		AttachStdin:  RunInteractive,
		AttachStdout: !RunDetach,
		AttachStderr: !RunDetach,
		OpenStdin:    RunInteractive,
		StdinOnce:    false,
		Tty:          RunTTY,

		User:       RunUser,
		Env:        RunEnv,
		WorkingDir: RunWorkdir,
		Hostname:   RunHostname,
	}
	if len(cmd) > 0 {
		req.Cmd = cmd
	}
	if RunEntrypoint != "" {
		// docker CLI treats --entrypoint as a single binary string; we do the same.
		req.Entrypoint = []string{RunEntrypoint}
	}

	// HostConfig
	hc := &HostConfig{}
	if RunRemove {
		hc.AutoRemove = true
	}
	if RunNetwork != "" {
		hc.NetworkMode = RunNetwork
	}
	req.HostConfig = hc

	if cerr := applyVolumes(&req, RunVolume); cerr != nil {
		return "", false, cerr
	}

	if cerr := applyPublish(&req, RunPublish); cerr != nil {
		return "", false, cerr
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return "", false, &ce.CustomError{Title: "Unable to marshal container create request", Message: err.Error()}
	}

	q := url.Values{}
	if RunName != "" {
		q.Set("name", RunName)
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	resp, err := client.Do(rest.Context, http.MethodPost, "/containers/create", q, bytes.NewReader(payload), headers)
	if err != nil {
		return "", false, &ce.CustomError{Title: "Unable to create container", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		msg := extras.StringsTrim(string(b))
		if msg == "" {
			msg = resp.Status
		}
		missing := resp.StatusCode == http.StatusNotFound && strings.Contains(strings.ToLower(msg), "no such image")
		return "", missing, &ce.CustomError{Title: "Container create failed", Message: fmt.Sprintf("POST /containers/create returned %s", msg)}
	}

	var out ContainerCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", false, &ce.CustomError{Title: "Unable to decode container create response", Message: err.Error()}
	}
	if out.ID == "" {
		return "", false, &ce.CustomError{Title: "Container create failed", Message: "daemon returned an empty container id"}
	}
	return out.ID, false, nil
}

func startContainer(client *rest.Client, id string) *ce.CustomError {
	path := "/containers/" + id + "/start"
	resp, err := client.Do(rest.Context, http.MethodPost, path, nil, nil, nil)
	if err != nil {
		return &ce.CustomError{Title: "Unable to start container", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		msg := extras.StringsTrim(string(b))
		if msg == "" {
			msg = resp.Status
		}
		return &ce.CustomError{Title: "Container start failed", Message: fmt.Sprintf("POST %s returned %s", path, msg)}
	}
	return nil
}

func attachContainer(client *rest.Client, id string) (*rest.HijackedConn, *ce.CustomError) {
	q := url.Values{}
	q.Set("stream", "1")
	q.Set("stdout", "1")
	q.Set("stderr", "1")
	if RunInteractive {
		q.Set("stdin", "1")
	}

	// No logs replay; `docker run` does not include previous logs.
	q.Set("logs", "0")

	headers := http.Header{}
	hj, err := client.Hijack(rest.Context, http.MethodPost, "/containers/"+id+"/attach", q, headers, nil, true)
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to attach to container", Message: err.Error()}
	}
	return hj, nil
}

func waitContainerExit(client *rest.Client, id string) (int, *ce.CustomError) {
	q := url.Values{}
	q.Set("condition", "not-running")
	path := "/containers/" + id + "/wait"

	resp, err := client.Do(rest.Context, http.MethodPost, path, q, nil, nil)
	if err != nil {
		return 1, &ce.CustomError{Title: "Unable to wait for container", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		msg := extras.StringsTrim(string(b))
		if msg == "" {
			msg = resp.Status
		}
		return 1, &ce.CustomError{Title: "Container wait failed", Message: fmt.Sprintf("POST %s returned %s", path, msg)}
	}

	var out ContainerWaitResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 1, &ce.CustomError{Title: "Unable to decode container wait response", Message: err.Error()}
	}
	if out.Error != nil && out.Error.Message != "" {
		return 1, &ce.CustomError{Title: "Container failed", Message: out.Error.Message}
	}

	return out.StatusCode, nil
}

func applyPublish(req *ContainerCreateRequest, pubs []string) *ce.CustomError {
	if len(pubs) == 0 {
		return nil
	}
	if req.ExposedPorts == nil {
		req.ExposedPorts = map[string]struct{}{}
	}
	if req.HostConfig == nil {
		req.HostConfig = &HostConfig{}
	}
	if req.HostConfig.PortBindings == nil {
		req.HostConfig.PortBindings = map[string][]PortBinding{}
	}

	for _, p := range pubs {
		portKey, bind, cerr := parsePublishSpec(p)
		if cerr != nil {
			return cerr
		}
		req.ExposedPorts[portKey] = struct{}{}
		req.HostConfig.PortBindings[portKey] = append(req.HostConfig.PortBindings[portKey], bind)
	}
	return nil
}

func applyVolumes(req *ContainerCreateRequest, vols []string) *ce.CustomError {
	if len(vols) == 0 {
		return nil
	}
	if req.HostConfig == nil {
		req.HostConfig = &HostConfig{}
	}
	for _, v := range vols {
		m, anonTarget, cerr := parseVolumeSpec(v)
		if cerr != nil {
			return cerr
		}
		if anonTarget != "" {
			if req.Volumes == nil {
				req.Volumes = map[string]struct{}{}
			}
			req.Volumes[anonTarget] = struct{}{}
			continue
		}
		req.HostConfig.Mounts = append(req.HostConfig.Mounts, m)
	}
	return nil
}

// parseVolumeSpec supports common docker forms:
//
//	-v /host/path:/container/path[:ro|rw]
//	-v volumeName:/container/path[:ro|rw]
//	-v /container/path                       (anonymous volume)
func parseVolumeSpec(spec string) (mount Mount, anonymousTarget string, errCode *ce.CustomError) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return Mount{}, "", &ce.CustomError{Title: "Invalid volume", Message: "empty -v value"}
	}

	parts := strings.Split(spec, ":")
	if len(parts) == 1 {
		// Anonymous volume: "-v /path".
		if strings.HasPrefix(parts[0], "/") {
			return Mount{}, parts[0], nil
		}
		return Mount{}, "", &ce.CustomError{Title: "Invalid volume", Message: fmt.Sprintf("unsupported -v format: %q", spec)}
	}
	if len(parts) > 3 {
		return Mount{}, "", &ce.CustomError{Title: "Invalid volume", Message: fmt.Sprintf("unsupported -v format: %q", spec)}
	}

	source := parts[0]
	target := parts[1]
	mode := ""
	if len(parts) == 3 {
		mode = parts[2]
	}
	if source == "" || target == "" {
		return Mount{}, "", &ce.CustomError{Title: "Invalid volume", Message: fmt.Sprintf("invalid -v format: %q", spec)}
	}
	if !strings.HasPrefix(target, "/") {
		return Mount{}, "", &ce.CustomError{Title: "Invalid volume", Message: fmt.Sprintf("container path must be absolute in -v %q", spec)}
	}

	readOnly := false
	if mode != "" {
		// Accept "ro" anywhere in the mode list (e.g. "ro,z").
		for _, m := range strings.Split(mode, ",") {
			if strings.TrimSpace(m) == "ro" {
				readOnly = true
			}
		}
	}

	mt := "volume"
	if looksLikeHostPath(source) {
		mt = "bind"
	}

	return Mount{Type: mt, Source: source, Target: target, ReadOnly: readOnly}, "", nil
}

func looksLikeHostPath(s string) bool {
	// Basic heuristic good enough for Linux:
	// - absolute paths
	// - relative paths (./foo, ../foo)
	// - ~ is not expanded, but if user passes it, treat as path
	return strings.HasPrefix(s, "/") || strings.HasPrefix(s, "./") || strings.HasPrefix(s, "../") || strings.HasPrefix(s, "~")
}

// parsePublishSpec supports common docker forms:
//
//	-p 8080:80
//	-p 127.0.0.1:8080:80
//	-p :80
//	-p 80
//
// and optional /udp or /tcp suffix on the container port, e.g. 8080:53/udp.
func parsePublishSpec(spec string) (portKey string, bind PortBinding, errCode *ce.CustomError) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", PortBinding{}, &ce.CustomError{Title: "Invalid publish", Message: "empty -p value"}
	}

	parts := strings.Split(spec, ":")
	if len(parts) > 3 {
		return "", PortBinding{}, &ce.CustomError{Title: "Invalid publish", Message: fmt.Sprintf("unsupported -p format: %q", spec)}
	}

	containerPart := parts[len(parts)-1]
	proto := "tcp"
	if strings.Contains(containerPart, "/") {
		sp := strings.SplitN(containerPart, "/", 2)
		containerPart = sp[0]
		if sp[1] != "" {
			proto = strings.ToLower(sp[1])
		}
	}

	if containerPart == "" {
		return "", PortBinding{}, &ce.CustomError{Title: "Invalid publish", Message: fmt.Sprintf("missing container port in -p %q", spec)}
	}
	if _, err := strconv.Atoi(containerPart); err != nil {
		return "", PortBinding{}, &ce.CustomError{Title: "Invalid publish", Message: fmt.Sprintf("invalid container port in -p %q", spec)}
	}

	bind = PortBinding{}
	switch len(parts) {
	case 1:
		// Only container port: random host port
		bind.HostPort = ""
	case 2:
		bind.HostPort = parts[0]
	case 3:
		bind.HostIP = parts[0]
		bind.HostPort = parts[1]
	}

	if bind.HostPort != "" {
		// allow empty host port for random assignment
		if _, err := strconv.Atoi(bind.HostPort); err != nil {
			return "", PortBinding{}, &ce.CustomError{Title: "Invalid publish", Message: fmt.Sprintf("invalid host port in -p %q", spec)}
		}
	}

	portKey = containerPart + "/" + proto
	return portKey, bind, nil
}

func setupContainerResizeHandler(client *rest.Client, id string) {
	resize := func() {
		ws, err := mobyterm.GetWinsize(os.Stdout.Fd())
		if err != nil {
			return
		}
		q := url.Values{}
		q.Set("h", fmt.Sprintf("%d", ws.Height))
		q.Set("w", fmt.Sprintf("%d", ws.Width))
		path := "/containers/" + id + "/resize"
		resp, err := client.Do(rest.Context, http.MethodPost, path, q, nil, nil)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}

	resize()

	ch := make(chan os.Signal, 2)
	signal.Notify(ch, syscall.SIGWINCH)

	go func() {
		defer signal.Stop(ch)
		for {
			select {
			case <-rest.Context.Done():
				return
			case <-ch:
				resize()
			}
		}
	}()
}

func proxySignalsToContainer(client *rest.Client, id string, stop <-chan struct{}) {
	ch := make(chan os.Signal, 16)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)
	defer signal.Stop(ch)

	for {
		select {
		case <-stop:
			return
		case <-rest.Context.Done():
			return
		case s := <-ch:
			sigName := signalName(s)
			if sigName == "" {
				continue
			}
			q := url.Values{}
			q.Set("signal", sigName)
			resp, err := client.Do(rest.Context, http.MethodPost, "/containers/"+id+"/kill", q, nil, nil)
			if err == nil && resp != nil {
				_ = resp.Body.Close()
			}
		}
	}
}

func signalName(s os.Signal) string {
	switch s {
	case syscall.SIGINT:
		return "SIGINT"
	case syscall.SIGTERM:
		return "SIGTERM"
	case syscall.SIGQUIT:
		return "SIGQUIT"
	case syscall.SIGHUP:
		return "SIGHUP"
	default:
		return ""
	}
}
