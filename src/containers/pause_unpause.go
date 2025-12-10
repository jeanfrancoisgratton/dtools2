// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/06 02:48
// Original filename: src/containers/pause_unpause.go

package containers

import (
	"dtools2/rest"
	"fmt"
	"net/http"
	"net/url"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// Pauses one or many containers

func PauseContainer(client *rest.Client, containers []string) *ce.CustomError {
	var id string
	var cerr *ce.CustomError
	q := url.Values{}

	for _, container := range containers {
		if id, cerr = Name2ID(client, container); cerr != nil {
			return cerr
		}
		path := "/containers/{" + id + "}/pause"

		resp, err := client.Do(rest.Context, http.MethodPost, path, q, nil, nil)
		if err != nil {
			return &ce.CustomError{Title: "Unable to POST request", Message: err.Error(), Code: 201}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			return &ce.CustomError{Title: "http request returned an error", Message: "POST" + path + " returned " + resp.Status, Code: 201}
		}
		if !rest.QuietOutput {
			fmt.Println(hftx.InProgressSign("Container " + container + hftx.Yellow(" PAUSED")))
		}
	}
	return nil
}

// The reverse: we unpause one or many containers

func UnpauseContainer(client *rest.Client, containers []string) *ce.CustomError {
	var id string
	var cerr *ce.CustomError

	for _, container := range containers {
		if id, cerr = Name2ID(client, container); cerr != nil {
			return cerr
		}
		path := "/containers/{" + id + "}/unpause"

		resp, err := client.Do(rest.Context, http.MethodPost, path, url.Values{}, nil, nil)
		if err != nil {
			return &ce.CustomError{Title: "Unable to POST request", Message: err.Error(), Code: 201}
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
			return &ce.CustomError{Title: "http request returned an error", Message: "POST" + path + " returned " + resp.Status, Code: 201}
		}
		if !rest.QuietOutput {
			fmt.Println(hftx.InProgressSign("Container " + container + hftx.Green(" UNPAUSED")))
		}
	}
	return nil
}
