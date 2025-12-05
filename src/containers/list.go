// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/24 20:29
// Original filename: src/containers/list.go

package containers

import (
	"dtools2/rest"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jeanfrancoisgratton/customError/v3"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func ListContainers(client *rest.Client, outputDisplay bool) ([]ContainerSummary, *customError.CustomError) {
	q := url.Values{}
	if OnlyRunningContainers {
		q.Set("all", "false")
	} else {
		q.Set("all", "true")
	}

	if DisplaySizeValues {
		q.Set("size", "true")
	} else {
		q.Set("size", "false")
	}

	resp, err := client.Do(rest.Context, http.MethodGet, "/containers/json", q, nil, nil)
	if err != nil {
		return nil, &customError.CustomError{Title: "Unable to list containers", Message: err.Error(), Code: 201}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &customError.CustomError{Title: "http request returned an error", Message: "GET /containers/json returned " + resp.Status, Code: 201}
	}

	var containers []ContainerSummary
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil,
			&customError.CustomError{Title: "Unable to decode JSON", Message: err.Error(), Code: 201}
	}
	// If we're not supposed to display anything, just return an empty slice.
	if !outputDisplay {
		return containers, nil
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	stateRow := 0
	if !ExtendedContainerInfo {
		stateRow = 3
		t.AppendHeader(table.Row{"Image", "Name", "Created", "State", "Status", "Ports"})
	} else {
		stateRow = 4
		t.AppendHeader(table.Row{"Container ID", "Image", "Name", "Created", "State", "Status", "Ports", "Command"})
	}

	// Option B: when there are no containers, append a single empty row to keep
	// the table borders and layout intact.
	if len(containers) == 0 {
		if !ExtendedContainerInfo {
			// 7 columns: Image, Name, Created, State, Status, Ports, Mounts
			t.AppendRow(table.Row{"", "", "", "", "", ""})
		} else {
			// 8 columns: Container ID, Image, Name, Created, State, Status, Ports, Command
			t.AppendRow(table.Row{"", "", "", "", "", "", "", ""})
		}
	} else {
		for _, container := range containers {
			containerImage := getImageTag(container.Image)
			prettyPorts := prettifyPortsList(container.Ports, "\n")

			if !ExtendedContainerInfo {
				t.AppendRow([]interface{}{
					containerImage,
					container.Names[0][1:],
					time.Unix(container.Created, 0).Format("2006.01.02 15:04:05"),
					container.State,
					container.Status,
					prettyPorts, // note: Ports/Mounts not used yet in non-extended view
				})
			} else {
				t.AppendRow([]interface{}{
					container.ID[:10],
					containerImage,
					container.Names[0][1:],
					time.Unix(container.Created, 0).Format("2006.01.02 15:04:05"),
					container.State,
					container.Status,
					prettyPorts,
					container.Command,
				})
			}
		}
	}

	t.SortBy([]table.SortBy{
		{Name: "Name", Mode: table.Asc},
	})
	t.SetStyle(table.StyleBold)

	t.Style().Format.Header = text.FormatDefault
	t.SetRowPainter(func(row table.Row) text.Colors {
		switch row[stateRow] {
		case "running":
			//return text.Colors{text.BgBlack, text.FgHiGreen}
			return text.Colors{text.FgHiGreen}
		case "crashed":
			return text.Colors{text.BgBlack, text.FgHiRed}
		case "blocked":
		case "suspended":
		case "paused":
			return text.Colors{text.FgHiYellow}
		}
		return nil
	})

	t.Render()
	return nil, nil
}
