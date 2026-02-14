// images/inspect.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/02/13

package images

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
	"strings"
	"text/tabwriter"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// InspectImage retrieves detailed information about an image.
//
// Behavior matches containers.InspectContainer / networks.InspectNetwork / volumes.InspectVolume:
//   - if rest.QuietOutput is true: print raw JSON payload and return the parsed object
//   - else if extras.OutputJSON is true: pretty-print JSON and return an empty object
//   - else: print formatted text output and return an empty object
func InspectImage(client *rest.Client, imageRef string) (ImageInspect, *ce.CustomError) {
	ctx := rest.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Docker/Podman accept either an image ID or a repo reference (e.g. repo:tag) here.
	// url.PathEscape() would percent-encode ':' and '/', which breaks repo:tag and
	// namespaced references. Keep those delimiters intact.
	escaped := url.PathEscape(imageRef)
	escaped = strings.ReplaceAll(escaped, "%2F", "/")
	escaped = strings.ReplaceAll(escaped, "%3A", ":")
	path := fmt.Sprintf("/images/%s/json", escaped)

	resp, err := client.Do(ctx, http.MethodGet, path, nil, nil, nil)
	if err != nil {
		return ImageInspect{}, &ce.CustomError{Title: "Failed to inspect image", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ImageInspect{}, &ce.CustomError{
			Title:   fmt.Sprintf("Image inspect failed (HTTP %d)", resp.StatusCode),
			Message: string(body),
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ImageInspect{}, &ce.CustomError{Title: "Failed to read inspect response", Message: err.Error()}
	}

	var inspectData ImageInspect
	if err := json.Unmarshal(body, &inspectData); err != nil {
		return ImageInspect{}, &ce.CustomError{Title: "Failed to parse inspect response", Message: err.Error()}
	}

	if rest.QuietOutput {
		fmt.Println(string(body))
		return inspectData, nil
	}

	if extras.OutputJSON {
		jsonBytes, cerr := extras.MarshalJSON(inspectData)
		if cerr != nil {
			return ImageInspect{}, cerr
		}
		if cerr := extras.PrintJSONBytes(jsonBytes); cerr != nil {
			return ImageInspect{}, cerr
		}
		return ImageInspect{}, nil
	}

	printFormattedInspect(&inspectData)
	return ImageInspect{}, nil
}

func printFormattedInspect(data *ImageInspect) {
	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	extras.PrintImageInspectData(w, data)
	_ = w.Flush()
}
