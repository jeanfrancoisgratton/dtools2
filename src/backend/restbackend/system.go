// dtools2
// src/backend/restbackend/system.go

package restbackend

import (
	"dtools2/rest"
	"dtools2/system"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type systemService struct{ client *rest.Client }

func (s systemService) RmContainers() *ce.CustomError {
	return system.RmContainers(s.client)
}

func (s systemService) Clean() *ce.CustomError {
	return system.Clean(s.client)
}

func (s systemService) Info() *ce.CustomError {
	return system.Info(s.client)
}
