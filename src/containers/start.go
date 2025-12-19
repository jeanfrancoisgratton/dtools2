// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/05 23:36
// Original filename: src/containers/start.go

package containers

import (
	"dtools2/rest"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// Starts one or many containers

func StartContainers(client *rest.Client, containers []string) *ce.CustomError {
	var cerr *ce.CustomError
	var cs []ContainerSummary
	OnlyRunningContainers = false

	// Fetch the list of containers currently present on the daemon, regardless of their state
	if cs, cerr = ListContainers(client, false); cerr != nil {
		return cerr
	}
	for _, container := range cs {
		if strings.ToLower(container.State) == "running" {
			continue
		}
		if slices.Contains(containers, container.Names[0][1:]) {
			if cerr = start(client, container.ID, ""); cerr != nil {
				return cerr
			}
		}
	}
	return nil
}

// The actual mechanics of starting the container

func start(client *rest.Client, id string, containerName string) *ce.CustomError {
	var cerr *ce.CustomError
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

	path := "/containers/" + id + "/start"

	resp, err := client.Do(rest.Context, http.MethodPost, path, url.Values{}, nil, nil)
	if err != nil {
		return &ce.CustomError{Title: "Unable to start container " + containerName, Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return &ce.CustomError{Title: "http request returned an error", Message: "POST" + path + " returned " + resp.Status}
	}
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign("Container " + containerName + hftx.Green(" STARTED")))
	}
	return nil
}

// Starts all non-running containers

func StartAllContainers(client *rest.Client) *ce.CustomError {
	var cerr *ce.CustomError
	var cs []ContainerSummary
	var containerList []string
	OnlyRunningContainers = false

	// Fetch the list of containers currently present on the daemon, regardless of their state
	if cs, cerr = ListContainers(client, false); cerr != nil {
		return cerr
	}

	for _, container := range cs {
		containerList = append(containerList, container.Names[0][1:])
	}

	return StartContainers(client, containerList)
}
