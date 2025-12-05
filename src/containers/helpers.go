// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/04 19:27
// Original filename: src/containers/helpers.go

package containers

import (
	"dtools2/rest"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// Name2ID takes the human-readable container name and returns its docker/podman ID
func Name2ID(client *rest.Client, containerName string) (string, *ce.CustomError) {
	OnlyRunningContainers = false
	ExtendedContainerInfo = false
	DisplaySizeValues = false

	// first off, let's fetch the list of containers
	if cs, err := ListContainers(client, false); err != nil {
		return "", err
	} else {
		if len(cs) == 0 {
			return "", &ce.CustomError{Fatality: ce.Warning, Message: "No containers found"}
		}
		for _, container := range cs {
			if strings.ToLower(container.Names[0][1:]) == containerName {
				return container.ID, nil
			}
		}
	}
	return "", &ce.CustomError{Fatality: ce.Warning, Message: "No containers found"}
}

// ID2Name takes a container ID and returns its human-readable name
func ID2Name(client *rest.Client, containerID string) (string, *ce.CustomError) {
	OnlyRunningContainers = false
	ExtendedContainerInfo = false
	DisplaySizeValues = false

	// first off, let's fetch the list of containers
	if cs, err := ListContainers(client, false); err != nil {
		return "", err
	} else {
		if len(cs) == 0 {
			return "", &ce.CustomError{Fatality: ce.Warning, Message: "No containers found"}
		}
		for _, container := range cs {
			if strings.ToLower(container.ID) == strings.ToLower(containerID) {
				return container.Names[0][1:], nil
			}
		}
	}
	return "", &ce.CustomError{Fatality: ce.Warning, Message: "No containers found"}
}
