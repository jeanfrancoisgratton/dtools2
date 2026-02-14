// networks/inspect.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/02/13

package networks

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

// InspectNetwork retrieves detailed information about a network.
//
// Behavior matches containers.InspectContainer:
//   - if rest.QuietOutput is true: print raw JSON payload and return the parsed object
//   - else if extras.OutputJSON is true: pretty-print JSON and return an empty object
//   - else: print formatted text output and return an empty object
func InspectNetwork(client *rest.Client, networkID string) (NetworkInspect, *ce.CustomError) {
	ctx := rest.Context
	if ctx == nil {
		ctx = context.Background()
	}

	path := fmt.Sprintf("/networks/%s", url.PathEscape(networkID))

	resp, err := client.Do(ctx, http.MethodGet, path, nil, nil, nil)
	if err != nil {
		return NetworkInspect{}, &ce.CustomError{Title: "Failed to inspect network", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return NetworkInspect{}, &ce.CustomError{Title: fmt.Sprintf("Network inspect failed (HTTP %d)", resp.StatusCode), Message: string(body)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NetworkInspect{}, &ce.CustomError{Title: "Failed to read inspect response", Message: err.Error()}
	}

	var inspectData NetworkInspect
	if err := json.Unmarshal(body, &inspectData); err != nil {
		return NetworkInspect{}, &ce.CustomError{Title: "Failed to parse inspect response", Message: err.Error()}
	}

	if rest.QuietOutput {
		fmt.Println(string(body))
		return inspectData, nil
	}

	if extras.OutputJSON {
		jsonBytes, cerr := extras.MarshalJSON(inspectData)
		if cerr != nil {
			return NetworkInspect{}, cerr
		}
		if cerr := extras.PrintJSONBytes(jsonBytes); cerr != nil {
			return NetworkInspect{}, cerr
		}
		return NetworkInspect{}, nil
	}

	printFormattedInspect(&inspectData)
	return NetworkInspect{}, nil
}

func printFormattedInspect(data *NetworkInspect) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	extras.PrintNetworkInspectData(w, data)
	_ = w.Flush()
}
