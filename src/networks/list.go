// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/14 20:28
// Original filename: src/networks/list.go

package networks

import (
	"dtools2/rest"
	"encoding/json"
	"net/http"
	"net/url"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

func ListNetworks(client *rest.Client) *ce.CustomError {
	// Create & execute the http request
	resp, err := client.Do(rest.Context, http.MethodGet, "/networks", url.Values{}, nil, nil)
	if err != nil {
		return &ce.CustomError{Title: "Unable to list networks", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return &ce.CustomError{Title: "http request returned an error", Message: "GET /networks returned " + resp.Status}
	}

	// Decode JSON only if we actually have content
	var networks []NetworkSummary
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(&networks); err != nil {
			return &ce.CustomError{Title: "Unable to decode JSON", Message: err.Error()}
		}
	}
