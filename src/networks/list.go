// networks/list.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/14 20:28
// Original filename: src/networks/list.go

package networks

import (
	"dtools2/containers"
	"dtools2/extras"
	"dtools2/rest"
	"fmt"
	"os"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfjson "github.com/jeanfrancoisgratton/helperFunctions/v4/prettyjson"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// NetworkList lists all networks and marks each as "in use" by looking at
// container network attachments. This keeps the operation to 2 API calls total:
//   - GET /networks
//   - GET /containers/json?all=1
//
// It avoids doing N calls to GET /networks/{id}.
func NetworkList(client *rest.Client, outputDisplay bool) ([]NetworkSummary, *ce.CustomError) {
	// 1) Fetch networks
	ns, cerr := fetchNetworkList(client)
	if cerr != nil {
		return nil, cerr
	}

	// 2) Fetch containers (must include stopped containers; they still occupy networks)
	containers.OnlyRunningContainers = false
	cs, ccerr := containers.ListContainers(client, false)
	if ccerr != nil {
		return nil, &ce.CustomError{Title: ccerr.Title, Message: ccerr.Message}
	}

	// 3) Compute network usage from container summaries (O(nets + containers))
	_, usedByName, usedByID := computeNetworkUsage(cs)
	for i := range ns {
		_, byName := usedByName[ns[i].Name]
		_, byID := usedByID[ns[i].ID]
		ns[i].InUse = byName || byID
	}

	if extras.Debug {
		fmt.Printf("Found %d networks; scanned %d containers\n", len(ns), len(cs))
	}

	if !outputDisplay {
		return ns, nil
	}

	// Optional: write JSON payload to a file and/or render JSON to stdout.
	// This is only done when outputDisplay is true (i.e., list commands).
	var payloadBytes []byte
	if extras.OutputFile != "" {
		b, cerr := extras.Send2File(ns, extras.OutputFile)
		if cerr != nil {
			return nil, cerr
		}
		payloadBytes = b
	}

	// --format takes precedence over --json, tables and ignores --quiet.
	if extras.OutputFormat != "" {
		rows, cerr := extras.ExtractFormatRows(ns, extras.OutputFormat)
		if cerr != nil {
			return nil, cerr
		}
		if cerr := extras.PrintFormatRows(rows); cerr != nil {
			return nil, cerr
		}
		return ns, nil
	}

	if extras.OutputJSON {
		// Marshal once if we didn't already (for --file).
		if payloadBytes == nil {
			b, cerr := extras.MarshalJSON(ns)
			if cerr != nil {
				return nil, cerr
			}
			payloadBytes = b
		}

		hfjson.Print(payloadBytes)
		return ns, nil
	}

	// JSON output not requested; if quiet, return data only.
	if rest.QuietOutput {
		return ns, nil
	}

	// 4) Render output
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.AppendHeader(table.Row{"Name", "Driver", "Scope", "Used", "Network ID"})

	if len(ns) == 0 {
		tw.AppendRow(table.Row{"", "", "", "", ""})
	} else {
		for _, n := range ns {
			displayID := n.ID
			if len(displayID) > 12 {
				displayID = displayID[:12]
			}

			//inUse := hftx.ThumbsDownSign("")
			inUse := hftx.ErrorSign("")
			if n.InUse {
				//inUse = hftx.ThumbsUpSign("")
				inUse = hftx.EnabledSign("")
			}

			tw.AppendRow(table.Row{
				n.Name,
				n.Driver,
				n.Scope,
				inUse,
				displayID,
			})
		}
	}

	tw.SortBy([]table.SortBy{{Name: "Name", Mode: table.Asc}})
	tw.SetStyle(table.StyleBold)
	tw.Style().Format.Header = text.FormatDefault
	tw.SetRowPainter(func(row table.Row) text.Colors {
		// "In use" column
		if row[3] == "yes" {
			return text.Colors{text.FgHiGreen}
		}
		return text.Colors{text.FgHiWhite}
	})

	tw.Render()
	return ns, nil
}
