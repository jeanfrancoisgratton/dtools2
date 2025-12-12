// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/11 15:16
// Original filename: src/containers/kill.go

package containers

import (
	"dtools2/rest"

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

	return KillContainers(client, containerList)
}
