// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/31 13:49
// Original filename: src/volumes/list.go

package volumes

import (
	"bytes"
	"dtools2/containers"
	"dtools2/extras"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfjson "github.com/jeanfrancoisgratton/helperFunctions/v4/prettyjson"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func ListVolumes(client *rest.Client, displayOutput bool) ([]Volume, *ce.CustomError) {
	// 1) Fetch volumes
	vols, verr := fetchVolumeList(client)
	if verr != nil {
		return nil, verr
	}

	// 2) Fetch containers (include stopped; they still reference volumes)
	oldOnlyRunning := containers.OnlyRunningContainers
	containers.OnlyRunningContainers = false
	cs, cerr := containers.ListContainers(client, false)
	containers.OnlyRunningContainers = oldOnlyRunning
	if cerr != nil {
		return nil, &ce.CustomError{Title: cerr.Title, Message: cerr.Message}
	}

	// 3) Build volume -> containers lookup (O(vols + containers))
	usedBy := computeVolumeUsage(cs)

	// 4) Add extra fields to the Volume struct; those 2 fields are not part of the official REST API structure.
	// IMPORTANT: ranging over a slice returns a COPY of the element; so writing to `v` would not mutate `vols`.
	for i := range vols {
		users := usedBy[vols[i].Name]
		vols[i].UsedByStr = strings.Join(users, "\n")
		vols[i].RefCount = len(users)
	}

	// 5) Render output if displayOutput is set
	if !displayOutput {
		return vols, nil
	}

	// Optional: write JSON payload to a file and/or render JSON to stdout.
	// This is only done when displayOutput is true (i.e., list commands).
	var payloadBytes []byte
	if extras.OutputFile != "" {
		b, cerr := extras.Send2File(vols, extras.OutputFile)
		if cerr != nil {
			return nil, cerr
		}
		payloadBytes = b
	}

	if extras.OutputJSON {
		// Marshal once if we didn't already (for --file).
		if payloadBytes == nil {
			b, cerr := extras.MarshalJSON(vols)
			if cerr != nil {
				return nil, cerr
			}
			payloadBytes = b
		}

		hfjson.Print(payloadBytes)
		return vols, nil
	}

	// JSON output not requested; if quiet, return data only.
	if rest.QuietOutput {
		return vols, nil
	}
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Name", "Driver", "Scope", "Created", "RefCount", "Used by"})

	if len(vols) == 0 {
		t.AppendRow(table.Row{"", "", "", "", "", ""})
	} else {
		for _, v := range vols {
			t.AppendRow(table.Row{v.Name, v.Driver, v.Scope, formatCreated(v.CreatedAt), v.RefCount, v.UsedByStr})
		}
	}

	t.SortBy([]table.SortBy{{Name: "Name", Mode: table.Asc}})
	t.SetStyle(table.StyleBold)
	t.Style().Format.Header = text.FormatDefault

	// Paint rows where the volume is used by at least one container.
	t.SetRowPainter(func(row table.Row) text.Colors {
		// Used by column is now index 5
		if s, ok := row[5].(string); ok && s != "" {
			return text.Colors{text.FgHiGreen}
		}
		return nil
	})

	t.Render()
	fmt.Println()
	return vols, nil
}

// fetchVolumeList fetches the daemon's volumes.
//
// Docker typically returns an object shaped like VolumeListResponse:
//
//	{"Volumes":[...],"Warnings":[...]}
//
// Some implementations (or older endpoints) may return a JSON array.
func fetchVolumeList(client *rest.Client) ([]Volume, *ce.CustomError) {
	resp, err := client.Do(rest.Context, http.MethodGet, "/volumes", url.Values{}, nil, nil)
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to list volumes", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, &ce.CustomError{Title: "http request returned an error", Message: "GET /volumes returned " + resp.Status}
	}
	if resp.StatusCode == http.StatusNoContent {
		return []Volume{}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to read HTTP response", Message: err.Error()}
	}
	trim := bytes.TrimSpace(body)
	if len(trim) == 0 {
		return []Volume{}, nil
	}

	switch trim[0] {
	case '{':
		var r VolumeListResponse
		if err := json.Unmarshal(trim, &r); err != nil {
			return nil, &ce.CustomError{Title: "Unable to decode JSON", Message: err.Error()}
		}
		return r.Volumes, nil
	case '[':
		var vols []Volume
		if err := json.Unmarshal(trim, &vols); err != nil {
			return nil, &ce.CustomError{Title: "Unable to decode JSON", Message: err.Error()}
		}
		return vols, nil
	default:
		return nil, &ce.CustomError{Title: "Unexpected JSON payload", Message: "GET /volumes returned an unsupported JSON shape"}
	}
}
