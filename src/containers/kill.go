// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/11 15:16
// Original filename: src/containers/kill.go

package containers

import (
	"context"
	"dtools2/rest"
	"io"
	"net/http"
	"net/url"
	"time"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// ENDPOINT : POST /containers/{id}/kill

func KillContainers(client *rest.Client, containers []string) *ce.CustomError {
	KillSwitch = true

	for _, container := range containers {
		if err := stop(client, "", container, 0); err != nil {
			return err
		}

	}
	return nil
}

func KillAllContainers(client *rest.Client) *ce.CustomError {
	containerList, cerr := runningContainerNames(client)
	if cerr != nil {
		return cerr
	}
	return KillContainers(client, containerList)
}

// runningContainerNames returns the names of the containers currently running on the daemon
func runningContainerNames(client *rest.Client) ([]string, *ce.CustomError) {
	OnlyRunningContainers = true

	cs, cerr := ListContainers(client, false)
	if cerr != nil {
		return nil, cerr
	}

	var containerList []string
	for _, c := range cs {
		containerList = append(containerList, c.Names[0][1:])
	}
	return containerList, nil
}

// waitNotRunning blocks until every container in the list is no longer running.
// The daemon replies to POST /containers/{id}/kill as soon as the signal is sent, not once the
// container has exited: a container listed right after a kill can thus still be "running".
// Supported by both docker and podman; it returns immediately for a container that is not running.
func waitNotRunning(client *rest.Client, containers []string) *ce.CustomError {
	const waitTimeout = 30 * time.Second

	q := url.Values{}
	q.Set("condition", "not-running")

	parent := rest.Context
	if parent == nil {
		parent = context.Background()
	}

	for _, container := range containers {
		ctx, cancel := context.WithTimeout(parent, waitTimeout)
		resp, err := client.Do(ctx, http.MethodPost, "/containers/"+container+"/wait", q, nil, nil)
		if err != nil {
			cancel()
			return &ce.CustomError{Title: "Unable to wait for container " + container + " to exit", Message: err.Error()}
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		cancel()

		if resp.StatusCode != http.StatusOK {
			return &ce.CustomError{Title: "http request returned an error", Message: "POST /containers/" + container + "/wait returned " + resp.Status}
		}
	}
	return nil
}
