// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/02 23:02
// Original filename: src/volumes/prune.go

package volumes

import (
	"dtools2/rest"
	"fmt"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// PruneVolumes removes all unused volumes
// IMPORTANT NOTE:
//
// There is a REST API endppoint, POST /volumes/prune, but I AM NOT USING IT.
// I implement a blacklist mechanism, client-side, that endpoint would just walk around it, so I had
// to implement my own prune function
// This is one of the few places where I somehow bypass the REST API.
// I still use the volume removal endpoint here, though : DELETE /volumes/{name}, tru RemoveVolumes()
func PruneVolumes(client *rest.Client) *ce.CustomError {
	var volumes []Volume
	var err *ce.CustomError
	var candidates []string

	if volumes, err = ListVolumes(client, false); err != nil {
		return err
	}

	for _, v := range volumes {
		if v.Scope == "local" && v.Driver == "local" && v.RefCount == 0 {
			_, isAnon := v.Labels["com.docker.volume.anonymous"]
			if !isAnon {
				if RemoveNamedVolumes {
					candidates = append(candidates, v.Name)
				}
			} else {
				candidates = append(candidates, v.Name)
			}
		}
	}
	rq := rest.QuietOutput
	rest.QuietOutput = true
	if err = RemoveVolumes(client, candidates); err != nil {
		return err
	}
	rest.QuietOutput = rq

	if !rest.QuietOutput {
		fmt.Println(hftx.EnabledSign("Volumes pruned"))
	}
	return nil
}
