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
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// ListNetworks lists all networks and marks each as "in use" by looking at
// container network attachments. This keeps the operation to 2 API calls total:
//   - GET /networks
//   - GET /containers/json?all=1
//
// It avoids doing N calls to GET /networks/{id}.
func ListNetworks(client *rest.Client) *ce.CustomError {
	// 1) Fetch networks
	ns, cerr := fetchNetworkList(client)
	if cerr != nil {
		return cerr
	}

	// 2) Fetch containers (must include stopped containers; they still occupy networks)
	containers.OnlyRunningContainers = false
	cs, ccerr := containers.ListContainers(client, false)
	if ccerr != nil {
		return &ce.CustomError{Title: ccerr.Title, Message: ccerr.Message}
	}

	// 3) Compute network usage from container summaries (O(nets + containers))
	usedByName, usedByID := computeNetworkUsage(cs)
	for i := range ns {
		_, byName := usedByName[ns[i].Name]
		_, byID := usedByID[ns[i].ID]
		ns[i].InUse = byName || byID
	}

	if extras.Debug {
		fmt.Printf("Found %d networks; scanned %d containers\n", len(ns), len(cs))
	}

	// 4) Render output
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stdout)
	tw.AppendHeader(table.Row{"Name", "Driver", "Scope", "In use", "Network ID"})

	if len(ns) == 0 {
		tw.AppendRow(table.Row{"", "", "", "", ""})
	} else {
		for _, n := range ns {
			displayID := n.ID
			if len(displayID) > 12 {
				displayID = displayID[:12]
			}

			inUse := "no"
			if n.InUse {
				inUse = "yes"
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
	return nil
}
