// dtools2
// src/backend/restbackend/volumes.go

package restbackend

import (
	"dtools2/rest"
	"dtools2/volumes"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type volumeService struct{ client *rest.Client }

func (s volumeService) List(displayOutput bool) *ce.CustomError {
	_, err := volumes.ListVolumes(s.client, displayOutput)
	return err
}

func (s volumeService) Remove(volList []string) *ce.CustomError {
	return volumes.RemoveVolumes(s.client, volList)
}

func (s volumeService) Prune() *ce.CustomError {
	return volumes.PruneVolumes(s.client)
}

func (s volumeService) Create(volumeName string) *ce.CustomError {
	return volumes.CreateVolume(s.client, volumeName)
}

func (s volumeService) Inspect(volumeName string) *ce.CustomError {
	_, err := volumes.InspectVolume(s.client, volumeName)
	return err
}
