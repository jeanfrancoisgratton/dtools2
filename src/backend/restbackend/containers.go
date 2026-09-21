// dtools2
// src/backend/restbackend/containers.go

package restbackend

import (
	"dtools2/containers"
	"dtools2/rest"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type containerService struct{ client *rest.Client }

func (s containerService) List(displayOutput bool) *ce.CustomError {
	_, err := containers.ListContainers(s.client, displayOutput)
	return err
}

func (s containerService) Info(container string) *ce.CustomError {
	return containers.InfoContainers(s.client, container)
}

func (s containerService) Inspect(id string) *ce.CustomError {
	_, err := containers.InspectContainer(s.client, id)
	return err
}

func (s containerService) Remove(containerList []string) *ce.CustomError {
	return containers.RemoveContainer(s.client, containerList)
}

func (s containerService) Pause(names []string) *ce.CustomError {
	return containers.PauseContainer(s.client, names)
}

func (s containerService) Unpause(names []string) *ce.CustomError {
	return containers.UnpauseContainer(s.client, names)
}

func (s containerService) Start(names []string) *ce.CustomError {
	return containers.StartContainers(s.client, names)
}

func (s containerService) StartAll() *ce.CustomError {
	return containers.StartAllContainers(s.client)
}

func (s containerService) Stop(names []string) *ce.CustomError {
	return containers.StopContainers(s.client, names)
}

func (s containerService) StopAll() *ce.CustomError {
	return containers.StopAllContainers(s.client)
}

func (s containerService) Rename(oldName, newName string) *ce.CustomError {
	return containers.RenameContainer(s.client, oldName, newName)
}

func (s containerService) Kill(names []string) *ce.CustomError {
	return containers.KillContainers(s.client, names)
}

func (s containerService) KillAll() *ce.CustomError {
	return containers.KillAllContainers(s.client)
}

func (s containerService) Restart(names []string) *ce.CustomError {
	return containers.RestartContainers(s.client, names)
}

func (s containerService) RestartAll() *ce.CustomError {
	return containers.RestartAllContainers(s.client)
}
