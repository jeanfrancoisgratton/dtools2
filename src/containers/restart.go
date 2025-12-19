package containers

import (
	"dtools2/rest"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

func RestartContainers(client *rest.Client, containers []string) *ce.CustomError {
	if KillSwitch {
		if err := KillContainers(client, containers); err != nil {
			return err
		}
	} else {
		if err := StopContainers(client, containers); err != nil {
			return err
		}
	}

	if err := StartContainers(client, containers); err != nil {
		return err
	}
	return nil
}

func RestartAllContainers(client *rest.Client) *ce.CustomError {
	if KillSwitch {
		if err := KillAllContainers(client); err != nil {
			return err
		}
	} else {
		if err := StopAllContainers(client); err != nil {
			return err
		}
	}

	if err := StartAllContainers(client); err != nil {
		return err
	}
	return nil
}
