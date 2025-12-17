// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/14 20:28
// Original filename: src/networks/list.go

package networks

import (
	"dtools2/containers"
	"dtools2/rest"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// NETWORK LIST
// Steps are:
// 		1. fetch the network list
//		2. fetch the containers list, to see if a container actually uses the networks
//		3. loop the network list
//			a. loop the container list
//			b. if the network appears in the container's info, we set the USED flag to true, no need to further process the loop

func ListNetworks(client *rest.Client) *ce.CustomError {
	var cerr *ce.CustomError
	var ns []NetworkSummary
	var cs []containers.ContainerSummary
	//NetWorksAllFree := false

	// fetch the network list
	if ns, cerr = fetchNetworkList(client); cerr != nil {
		return cerr
	}

	// fetch container info
	containers.OnlyRunningContainers = false
	if cs, cerr = containers.ListContainers(client, false); cerr != nil {
		return cerr
	}

	for _, n := range ns {
		for _, c := range cs {
			if networkInUse(n.Name, c.Names[0][1:]) {
				
			}
		}
	}
	//// If no container is on the daemon, this means that no network is in use
	//if len(cs) == 0 {
	//	NetWorksAllFree = true
	//}
	//

	return nil
}
