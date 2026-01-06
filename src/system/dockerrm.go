// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/06 00:27
// Original filename: src/extras/dockerrm.go

package system

import (
	"dtools2/containers"
	"dtools2/rest"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// RmContainers : removes all created or exited containers (not paused or running ones)

func RmContainers(client *rest.Client) *ce.CustomError {
	containers.OnlyRunningContainers = false
	containers.ForceRemoveContainer = ForceRemove
	containers.RemoveUnamedVolumes = RemoveUnamedVolumes
	containers.RemoveBlacklisted = RemoveBlacklisted

	candidates := []string{}

	if cs, err := containers.ListContainers(client, false); err != nil {
		return err
	} else {
		for _, c := range cs {
			a := strings.ToLower(c.State)
			if a == "created" || a == "exited" {
				candidates = append(candidates, c.Names[0][1:])
			}
		}
	}
	return containers.RemoveContainer(client, candidates)
}
