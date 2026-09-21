// dtools2
// src/backend/restbackend/networks.go

package restbackend

import (
	"dtools2/networks"
	"dtools2/rest"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type networkService struct{ client *rest.Client }

func (s networkService) List(displayOutput bool) *ce.CustomError {
	_, err := networks.NetworkList(s.client, displayOutput)
	return err
}

func (s networkService) Add(networkName string) *ce.CustomError {
	return networks.AddNetwork(s.client, networkName)
}

func (s networkService) Remove(netList []string) *ce.CustomError {
	return networks.RemoveNetwork(s.client, netList)
}

func (s networkService) Attach(network, container string) *ce.CustomError {
	return networks.AttachNetwork(s.client, network, container)
}

func (s networkService) Detach(network, container string) *ce.CustomError {
	return networks.DetachNetwork(s.client, network, container)
}

func (s networkService) Inspect(networkID string) *ce.CustomError {
	_, err := networks.InspectNetwork(s.client, networkID)
	return err
}
