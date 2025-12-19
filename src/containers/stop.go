// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/07 20:12
// Original filename: src/containers/stop.go

package containers

import (
	"dtools2/rest"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// StopContainers stops one or many containers whose names are provided in the
// containers slice. Behaviour depends on the StopTimeout value:
//
//   - StopTimeout > 0: containers are stopped sequentially, and the timeout
//     value (in seconds) is passed to the Docker/Podman API as the `t` query
//     parameter.
//   - StopTimeout == 0: containers are stopped concurrently using goroutines,
//     each with a sensible default timeout.
func StopContainers(client *rest.Client, containers []string) *ce.CustomError {
	var (
		cerr *ce.CustomError
		cs   []ContainerSummary
	)

	OnlyRunningContainers = false

	// Fetch the list of containers currently present on the daemon, regardless of their state.
	if cs, cerr = ListContainers(client, false); cerr != nil {
		return cerr
	}

	// Filter candidates: only containers that are currently running and that are
	// explicitly requested in the containers slice.
	var targets []ContainerSummary
	for _, c := range cs {
		if strings.ToLower(c.State) != "running" {
			continue
		}
		if slices.Contains(containers, c.Names[0][1:]) {
			targets = append(targets, c)
		}
	}

	if len(targets) == 0 {
		return nil
	}

	if StopTimeout == 0 {
		return stopContainersConcurrent(client, targets)
	}

	return stopContainersSequential(client, targets, StopTimeout)
}

// stopContainersSequential stops all containers one after another, using the
// provided timeout (in seconds) for the Docker/Podman API.
func stopContainersSequential(client *rest.Client, targets []ContainerSummary, timeout int) *ce.CustomError {
	for _, c := range targets {
		name := c.Names[0][1:]
		if cerr := stop(client, c.ID, name, timeout); cerr != nil {
			return cerr
		}
	}
	return nil
}

// stopContainersConcurrent stops all containers concurrently. Each container
// is stopped with a fixed, sensible timeout. Any errors are aggregated and
// returned as a single CustomError.
func stopContainersConcurrent(client *rest.Client, targets []ContainerSummary) *ce.CustomError {
	const concurrentTimeout = 10 // seconds

	errCh := make(chan *ce.CustomError, len(targets))
	var wg sync.WaitGroup
	wg.Add(len(targets))

	for _, c := range targets {
		c := c // capture
		go func() {
			defer wg.Done()
			name := c.Names[0][1:]
			if cerr := stop(client, c.ID, name, concurrentTimeout); cerr != nil {
				errCh <- cerr
			}
		}()
	}

	wg.Wait()
	close(errCh)

	var (
		errMsgs []string
	)

	for e := range errCh {
		if e == nil {
			continue
		}
		errMsgs = append(errMsgs, e.Title+": "+e.Message)
	}

	if len(errMsgs) == 0 {
		return nil
	}

	return &ce.CustomError{Title: "Errors occurred while stopping containers concurrently", Message: strings.Join(errMsgs, "; ")}
}

// stop performs the actual HTTP POST /containers/{id}/stop call.
// If id is empty, it is resolved from containerName. If containerName is empty,
// it is resolved from id. The timeout (in seconds) is passed as the `t` query
// parameter when greater than zero.
func stop(client *rest.Client, id string, containerName string, timeout int) *ce.CustomError {
	var cerr *ce.CustomError
	action := "/stop"
	if id == "" {
		if id, cerr = Name2ID(client, containerName); cerr != nil {
			return cerr
		}
	}
	if containerName == "" {
		if containerName, cerr = ID2Name(client, id); cerr != nil {
			return cerr
		}
	}

	// The KillSwitch is used mainly by the Kill / KillAll commands
	if KillSwitch {
		action = "/kill"
	}
	path := "/containers/" + id + action
	q := url.Values{}

	if timeout > 0 {
		q.Set("t", strconv.Itoa(timeout))
	}

	resp, err := client.Do(rest.Context, http.MethodPost, path, q, nil, nil)
	if err != nil {
		return &ce.CustomError{Title: "Unable to stop/kill the container " + containerName, Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return &ce.CustomError{Title: "http request returned an error", Message: "POST " + path + " returned " + resp.Status}
	}

	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign("Container " + containerName + hftx.Red(" STOPPED")))
	}

	return nil
}

// StopAllContainers builds a list of all containers present on the daemon and delegates
// to StopContainers, so it benefits from the timeout and concurrency logic.
func StopAllContainers(client *rest.Client) *ce.CustomError {
	var (
		cerr *ce.CustomError
		cs   []ContainerSummary
	)

	OnlyRunningContainers = true

	// Fetch the list of containers currently present on the daemon, regardless of their state.
	if cs, cerr = ListContainers(client, false); cerr != nil {
		return cerr
	}

	var containerList []string
	for _, c := range cs {
		containerList = append(containerList, c.Names[0][1:])
	}

	if len(containerList) == 0 {
		fmt.Println(hftx.NoteSign("Not a single container is running, STOPALL is thus un-needed"))
		return nil
	}
	return StopContainers(client, containerList)
}
