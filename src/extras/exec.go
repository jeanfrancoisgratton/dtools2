// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 19:59
// Original filename: src/extras/exec.go

package extras

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"dtools2/rest"

	"github.com/docker/docker/pkg/stdcopy"
	ce "github.com/jeanfrancoisgratton/customError/v3"
	mobyterm "github.com/moby/term"
	xterm "golang.org/x/term"
)

// Flags (wired from Cobra).
var (
	Interactive bool
	AllocateTTY bool
	User        string
)

// Run emulates `docker exec`.
//
// It returns the remote command's exit code (like docker exec does).
// If a transport/protocol error occurs, a CustomError is returned and the caller
// should exit with code 1.
func Run(client *rest.Client, container string, cmd []string) (int, *ce.CustomError) {
	if len(cmd) == 0 {
		return 1, &ce.CustomError{Title: "Missing command", Message: "no command specified"}
	}

	// Docker CLI errors when -it is used but stdin isn't a TTY.
	if AllocateTTY && Interactive {
		if !xterm.IsTerminal(int(os.Stdin.Fd())) {
			return 1, &ce.CustomError{Title: "The input device is not a TTY", Message: "cannot allocate a TTY with non-terminal stdin"}
		}
	}

	execID, cerr := createExec(client, container, cmd)
	if cerr != nil {
		return 1, cerr
	}

	// Start the exec session (hijacked connection).
	if cerr := startAndStream(client, execID); cerr != nil {
		return 1, cerr
	}

	exitCode, cerr := inspectExitCode(client, execID)
	if cerr != nil {
		return 1, cerr
	}
	return exitCode, nil
}

func createExec(client *rest.Client, container string, cmd []string) (string, *ce.CustomError) {
	req := ExecCreateRequest{
		AttachStdin:  Interactive,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          AllocateTTY,
		Cmd:          cmd,
		User:         User,
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return "", &ce.CustomError{Title: "Unable to marshal exec request", Message: err.Error()}
	}

	path := "/containers/" + container + "/exec"
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	resp, err := client.Do(rest.Context, http.MethodPost, path, nil, bytes.NewReader(payload), headers)
	if err != nil {
		return "", &ce.CustomError{Title: "Unable to create exec instance", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		msg := StringsTrim(string(b))
		if msg == "" {
			msg = resp.Status
		}
		return "", &ce.CustomError{Title: "Exec create failed", Message: fmt.Sprintf("POST %s returned %s", path, msg)}
	}

	var out ExecCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", &ce.CustomError{Title: "Unable to decode exec create response", Message: err.Error()}
	}
	if out.ID == "" {
		return "", &ce.CustomError{Title: "Exec create failed", Message: "daemon returned an empty exec id"}
	}
	return out.ID, nil
}

func startAndStream(client *rest.Client, execID string) *ce.CustomError {
	startReq := ExecStartRequest{Detach: false, Tty: AllocateTTY}
	payload, err := json.Marshal(startReq)
	if err != nil {
		return &ce.CustomError{Title: "Unable to marshal exec start request", Message: err.Error()}
	}

	path := "/exec/" + execID + "/start"
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	hj, err := client.Hijack(rest.Context, http.MethodPost, path, nil, headers, payload, true)
	if err != nil {
		return &ce.CustomError{Title: "Unable to start exec session", Message: err.Error()}
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
	if AllocateTTY && Interactive && xterm.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err := xterm.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			restoreTerm = func() { _ = xterm.Restore(int(os.Stdin.Fd()), oldState) }
		}
	}
	if restoreTerm != nil {
		defer restoreTerm()
	}

	if AllocateTTY {
		setupResizeHandler(client, execID)
	}

	// Stream.
	var wg sync.WaitGroup

	stdinStop := make(chan struct{})

	// stdin
	if Interactive {
		wg.Add(1)
		go func() {
			defer wg.Done()
			CopyStdin(conn, stdinStop)
			if cw, ok := conn.(interface{ CloseWrite() error }); ok {
				_ = cw.CloseWrite()
			}
		}()
	}

	// stdout/stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		if AllocateTTY {
			_, _ = io.Copy(os.Stdout, reader)
		} else {
			_, _ = stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
		}
		close(stdinStop) // stop stdin reader (important for "-i" + short-lived commands)
	}()

	wg.Wait()
	return nil
}

func errorsIsWouldBlock(err error) bool {
	// Go maps EAGAIN/EWOULDBLOCK to syscall.EAGAIN (and sometimes syscall.EWOULDBLOCK).
	return errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EWOULDBLOCK)
}

func inspectExitCode(client *rest.Client, execID string) (int, *ce.CustomError) {
	path := "/exec/" + execID + "/json"

	resp, err := client.Do(rest.Context, http.MethodGet, path, nil, nil, nil)
	if err != nil {
		return 1, &ce.CustomError{Title: "Unable to inspect exec instance", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		msg := StringsTrim(string(b))
		if msg == "" {
			msg = resp.Status
		}
		return 1, &ce.CustomError{Title: "Exec inspect failed", Message: fmt.Sprintf("GET %s returned %s", path, msg)}
	}

	var out ExecInspectResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return 1, &ce.CustomError{Title: "Unable to decode exec inspect response", Message: err.Error()}
	}

	return out.ExitCode, nil
}

func setupResizeHandler(client *rest.Client, execID string) {
	// Send an initial resize, then keep it updated on SIGWINCH.
	resize := func() {
		ws, err := mobyterm.GetWinsize(os.Stdout.Fd())
		if err != nil {
			return
		}
		q := url.Values{}
		q.Set("h", fmt.Sprintf("%d", ws.Height))
		q.Set("w", fmt.Sprintf("%d", ws.Width))
		path := "/exec/" + execID + "/resize"
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
