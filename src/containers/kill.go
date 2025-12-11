// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/11 15:16
// Original filename: src/containers/kill.go

package containers

import (
	"dtools2/rest"
	"fmt"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// ENDPOINT : POST /containers/{id}/kill

func KillContainer(client *rest.Client, containers []string) *ce.CustomError {
	fmt.Println(hftx.Red("KILLCONTAINER"))
	return nil
}

func KillAllContainers(client *rest.Client) *ce.CustomError {
	fmt.Println(hftx.Red("KILLALLCONTAINERS"))
	return nil
}
