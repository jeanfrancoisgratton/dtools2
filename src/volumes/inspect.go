// volumes/inspect.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/02/13

package volumes

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

// InspectVolume retrieves detailed information about a volume.
//
// Behavior matches containers.InspectContainer:
//   - if rest.QuietOutput is true: print raw JSON payload and return the parsed object
//   - else if extras.OutputJSON is true: pretty-print JSON and return an empty object
//   - else: print formatted text output and return an empty object
func InspectVolume(client *rest.Client, volumeName string) (Volume, *ce.CustomError) {
	ctx := rest.Context
	if ctx == nil {
		ctx = context.Background()
	}

	path := fmt.Sprintf("/volumes/%s", url.PathEscape(volumeName))

	resp, err := client.Do(ctx, http.MethodGet, path, nil, nil, nil)
	if err != nil {
		return Volume{}, &ce.CustomError{Title: "Failed to inspect volume", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Volume{}, &ce.CustomError{Title: fmt.Sprintf("Volume inspect failed (HTTP %d)", resp.StatusCode), Message: string(body)}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Volume{}, &ce.CustomError{Title: "Failed to read inspect response", Message: err.Error()}
	}

	var inspectData Volume
	if err := json.Unmarshal(body, &inspectData); err != nil {
		return Volume{}, &ce.CustomError{Title: "Failed to parse inspect response", Message: err.Error()}
	}

	if rest.QuietOutput {
		fmt.Println(string(body))
		return inspectData, nil
	}

	if extras.OutputJSON {
		jsonBytes, cerr := extras.MarshalJSON(inspectData)
		if cerr != nil {
			return Volume{}, cerr
		}
		if cerr := extras.PrintJSONBytes(jsonBytes); cerr != nil {
			return Volume{}, cerr
		}
		return Volume{}, nil
	}

	printFormattedInspect(&inspectData)
	return Volume{}, nil
}

func printFormattedInspect(data *Volume) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	extras.PrintVolumeInspectData(w, data)
	_ = w.Flush()
}
