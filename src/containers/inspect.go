// containers/inspect.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/02/01

package containers

import (
	"context"
	"dtools2/extras"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"text/tabwriter"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// InspectContainer retrieves detailed information about a container.
// If rest.QuietOutput is true, it returns the raw JSON object.
// If extras.OutputJSON is true, it outputs JSON to stdout.
// Otherwise, it outputs formatted text.
func InspectContainer(client *rest.Client, containerID string) (ContainerInspect, *ce.CustomError) {
	ctx := rest.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Build the API path
	path := fmt.Sprintf("/containers/%s/json", url.PathEscape(containerID))

	// Add size parameter to get SizeRw and SizeRootFs
	query := url.Values{}
	query.Set("size", "true")

	resp, err := client.Do(ctx, http.MethodGet, path, query, nil, nil)
	if err != nil {
		return ContainerInspect{}, &ce.CustomError{
			Title:   "Failed to inspect container",
			Message: err.Error(),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ContainerInspect{}, &ce.CustomError{
			Title:   fmt.Sprintf("Container inspect failed (HTTP %d)", resp.StatusCode),
			Message: string(body),
		}
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ContainerInspect{}, &ce.CustomError{
			Title:   "Failed to read inspect response",
			Message: err.Error(),
		}
	}

	// Parse into ContainerInspect structure
	var inspectData ContainerInspect
	if err := json.Unmarshal(body, &inspectData); err != nil {
		return ContainerInspect{}, &ce.CustomError{
			Title:   "Failed to parse inspect response",
			Message: err.Error(),
		}
	}

	// Handle quiet output (return raw JSON)
	if rest.QuietOutput {
		fmt.Println(string(body))
		return inspectData, nil
	}

	// Handle JSON output
	if extras.OutputJSON {
		jsonBytes, cerr := extras.MarshalJSON(inspectData)
		if cerr != nil {
			return ContainerInspect{}, cerr
		}
		if cerr := extras.PrintJSONBytes(jsonBytes); cerr != nil {
			return ContainerInspect{}, cerr
		}
		return ContainerInspect{}, nil
	}

	// Handle formatted text output
	printFormattedInspect(&inspectData)

	return ContainerInspect{}, nil
}

// printFormattedInspect displays container inspect data in a formatted text output
func printFormattedInspect(data *ContainerInspect) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)

	// Convert structs to map[string]interface{} for the helper functions
	stateMap := structToMap(data.State)
	configMap := structToMap(data.Config)
	hostConfigMap := structToMap(data.HostConfig)
	networkSettingsMap := structToMap(data.NetworkSettings)
	mountsSlice := structToSliceMap(data.Mounts)

	extras.PrintContainerInspectData(w, data.ID, data.Name, data.Created, stateMap, data.Image, data.Platform,
		data.Driver, data.SizeRw, data.SizeRootFs, configMap, hostConfigMap, networkSettingsMap, mountsSlice)
}

// structToMap converts a struct to map[string]interface{} using JSON marshaling
func structToMap(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil
	}

	return result
}

// structToSliceMap converts a slice of structs to []interface{} via JSON
func structToSliceMap(v interface{}) []interface{} {
	if v == nil {
		return nil
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return nil
	}

	var result []interface{}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return nil
	}

	return result
}
