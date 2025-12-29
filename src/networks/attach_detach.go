// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/27 20:02
// Original filename: src/networks/attach_detach.go

package networks

import (
	"bytes"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

func AttachNetwork(client *rest.Client, network, container string) *ce.CustomError {
	// First, we set the request parameters payload. For now, we only support the container name as param, we
	// Do not modify the EndpointConfig options. Maybe later ? Maybe, but for now I do not see the point of it

	requestPayload, jerr := json.Marshal(NetworkConnectRequest{Container: container, EndpointConfig: nil})
	if jerr != nil {
		return &ce.CustomError{Title: "Unable to marshal the JSON payload", Message: jerr.Error()}
	}
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	if id, err := Name2ID(client, network); err != nil {
		return err
	} else {
		if aerr := attachDetach_action(client, true, requestPayload, headers, id, network); aerr != nil {
			return aerr
		}
	}
	return nil
}

func DetachNetwork(client *rest.Client, network, container string) *ce.CustomError {
	// First, we set the request parameters payload. For now, we only support the container name as param, we
	// Do not modify the EndpointConfig options. Maybe later ? Maybe, but for now I do not see the point of it

	requestPayload, jerr := json.Marshal(NetworkDisconnectRequest{Container: container, Force: ForceNetworkDetach})
	if jerr != nil {
		return &ce.CustomError{Title: "Unable to marshal the JSON payload", Message: jerr.Error()}
	}
	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	if id, err := Name2ID(client, network); err != nil {
		return err
	} else {
		if aerr := attachDetach_action(client, false, requestPayload, headers, id, network); aerr != nil {
			return aerr
		}
	}
	return nil
}

// attachDetach_action :
// As both attach (connect) and detach (disconnect) share much of the same code, it made sense to merge the actual actions in a single function
func attachDetach_action(client *rest.Client, attachAction bool, requestPayload []byte, headers http.Header, id, networkname string) *ce.CustomError {
	path := "/networks/" + id
	action := "connect"

	if attachAction {
		path += "/connect"
	} else {
		path += "/disconnect"
		action = "disconnect"
	}
	resp, err := client.Do(rest.Context, http.MethodPost, path,
		url.Values{}, bytes.NewReader(requestPayload), headers)
	if err != nil {
		return &ce.CustomError{Title: "Unable to " + action + " network", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ce.CustomError{Title: "HTTP request returned an error", Message: resp.Status}
	}

	if !rest.QuietOutput {
		fmt.Println(hftx.GreenGoSign("Network " + hftx.Green(networkname) + " " + action + "ed"))
	}
	return nil
}
