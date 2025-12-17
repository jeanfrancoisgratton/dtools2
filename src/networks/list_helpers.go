// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/15 18:47
// Original filename: src/networks/list_helpers.go

package networks

import (
	"dtools2/extras"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// This is where we actually fetch the network list
func fetchNetworkList(client *rest.Client) ([]NetworkSummary, *ce.CustomError) {
	// Create & execute the http request
	resp, err := client.Do(rest.Context, http.MethodGet, "/networks", url.Values{}, nil, nil)
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to list networks", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, &ce.CustomError{Title: "http request returned an error", Message: "GET /networks returned " + resp.Status}
	}

	// Decode JSON only if we actually have content
	var networks []NetworkSummary
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(&networks); err != nil {
			return nil, &ce.CustomError{Title: "Unable to decode JSON", Message: err.Error()}
		}
	}
	if len(networks) == 0 {
		fmt.Println(hftx.WarningSign(" No network were found"))
		return nil, nil
	}

	if extras.Debug {
		fmt.Println(hftx.ScrollSign(fmt.Sprintf("Found %v networks", len(networks))))
		return nil, nil
	}
	return networks, nil
}

// This is where we validate if a network is being used by a container
// TODO : function signature needs refining
func networkInUse(client *rest.Client, networkname NetworkSummary) (bool, *ce.CustomError) {
	return true, nil
}
