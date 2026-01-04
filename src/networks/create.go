package networks

import (
	"bytes"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

type dockerErrorResponse struct {
	Message string `json:"message"`
}

/*
Create network
Endpoint:
	POST /networks/create
Errors:
	201: NetworkCreateResponse
	403: ErrorResponse
	404: ErrorResponse (“plugin not found”)
	500: ErrorResponse
*/

func AddNetwork(client *rest.Client, networkName string) *ce.CustomError {
	ncr := NetworkCreateRequest{
		Name:           networkName,
		CheckDuplicate: true,
		Driver:         NetworkDriverName,
		Internal:       NetworkInternalUse,
		Attachable:     NetworkAttachable,
		EnableIPv6:     NetworkEnableIPv6,
	}

	payload, jerr := json.MarshalIndent(ncr, "", "  ")
	if jerr != nil {
		return &ce.CustomError{Title: "Unable to marshal the JSON payload", Message: jerr.Error()}
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	resp, err := client.Do(rest.Context, http.MethodPost, "/networks/create",
		url.Values{}, bytes.NewReader(payload), headers)
	if err != nil {
		return &ce.CustomError{Title: "Unable to create the network", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Best-effort decode of daemon error {"message": "..."}
		bodyBytes, _ := io.ReadAll(resp.Body)
		msg := "POST /networks/create returned " + resp.Status
		if len(bodyBytes) > 0 {
			var der dockerErrorResponse
			if json.Unmarshal(bodyBytes, &der) == nil && der.Message != "" {
				msg = der.Message
			} else {
				msg = msg + ": " + string(bodyBytes)
			}
		}
		return &ce.CustomError{Title: "HTTP request returned an error", Message: msg}
	}

	if !rest.QuietOutput {
		fmt.Println(hftx.GreenGoSign("Network " + networkName + " has been created"))
	}

	return nil
}
