// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/15 18:47
// Original filename: src/networks/list_helpers.go

package networks

import (
	"dtools2/containers"
	"dtools2/extras"
	"dtools2/rest"
	"encoding/json"
	"net/http"
	"net/url"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// fetchNetworkList fetches the daemon's network summaries.
func fetchNetworkList(client *rest.Client) ([]NetworkSummary, *ce.CustomError) {
	resp, err := client.Do(rest.Context, http.MethodGet, "/networks", url.Values{}, nil, nil)
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to list networks", Message: err.Error()}
	}
	defer resp.Body.Close()

	// Docker: 200 OK (JSON array). Some implementations may return 204 when empty.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, &ce.CustomError{
			Title:   "http request returned an error",
			Message: "GET /networks returned " + resp.Status,
		}
	}

	// 204 => empty list
	if resp.StatusCode == http.StatusNoContent {
		return []NetworkSummary{}, nil
	}

	var networks []NetworkSummary
	if err := json.NewDecoder(resp.Body).Decode(&networks); err != nil {
		return nil, &ce.CustomError{Title: "Unable to decode JSON", Message: err.Error()}
	}

	if extras.Debug {
		// Keep behavior non-destructive: debug should not short-circuit business logic.
		// Callers decide what to print.
	}

	return networks, nil
}

// computeNetworkUsage builds two lookup sets (by network name and by network ID)
// from the container summaries returned by GET /containers/json.
//
// This allows callers to mark networks as "in use" without doing N calls to
// GET /networks/{id}.
func computeNetworkUsage(cs []containers.ContainerSummary) (map[string]struct{}, map[string]struct{}) {
	usedByName := make(map[string]struct{})
	usedByID := make(map[string]struct{})

	for _, c := range cs {
		if c.NetworkSettings == nil || c.NetworkSettings.Networks == nil {
			continue
		}
		for netName, ep := range c.NetworkSettings.Networks {
			if netName != "" {
				usedByName[netName] = struct{}{}
			}
			if ep.NetworkID != "" {
				usedByID[ep.NetworkID] = struct{}{}
			}
		}
	}

	return usedByName, usedByID
}
