package containers

import (
	"dtools2/rest"
	"fmt"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

func RestartContainers(client *rest.Client, containers []string) *ce.CustomError {
	if KillSwitch {
		if err := KillContainers(client, containers); err != nil {
			return err
		}
		// A killed container might still be "running" for a short while; StartContainers would skip it
		if err := waitNotRunning(client, containers); err != nil {
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

// RestartAllContainers restarts the containers that are running when it is invoked, and only those:
// containers that were already stopped beforehand are left alone
func RestartAllContainers(client *rest.Client) *ce.CustomError {
	running, err := runningContainerNames(client)
	if err != nil {
		return err
	}
	if len(running) == 0 {
		if !rest.QuietOutput {
			fmt.Println(hftx.NoteSign("Not a single container is running, RESTARTALL is thus un-needed"))
		}
		return nil
	}
	return RestartContainers(client, running)
}
