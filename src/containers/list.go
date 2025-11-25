// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/24 20:29
// Original filename: src/containers/list.go

package containers

import (
	"context"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func ContainersList(ctx context.Context, client *rest.Client, outputDisplay bool) ([]ContainerSummary, error) {
	q := url.Values{}
	if OnlyRunningContainers {
		q.Set("all", "false")
	} else {
		q.Set("all", "true")
	}

	resp, err := client.Do(ctx, http.MethodGet, "/containers/json", q, nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET /containers/json returned %s", resp.Status)
	}

	// If we're not supposed to display anything, just return an empty slice.
	if !outputDisplay {
		return []ContainerSummary{}, nil
	}

	var containers []ContainerSummary
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	stateRow := 0
	if !ExtendedContainerInfo {
		stateRow = 3
		t.AppendHeader(table.Row{"Image", "Name", "Created", "State", "Status", "Ports", "Mounts"})
	} else {
		stateRow = 4
		t.AppendHeader(table.Row{"Container ID", "Image", "Name", "Created", "State", "Status", "Ports", "Command"})
	}

	// Option B: when there are no containers, append a single empty row to keep
	// the table borders and layout intact.
	if len(containers) == 0 {
		if !ExtendedContainerInfo {
			// 7 columns: Image, Name, Created, State, Status, Ports, Mounts
			t.AppendRow(table.Row{"", "", "", "", "", "", ""})
		} else {
			// 8 columns: Container ID, Image, Name, Created, State, Status, Ports, Command
			t.AppendRow(table.Row{"", "", "", "", "", "", "", ""})
		}
	} else {
		for _, container := range containers {
			containerImage := getImageTag(container.Image)
			prettyPorts := prettifyPortsList(container.Ports)
			//prettyMounts := prettifyMounts(container.Mounts)

			if !ExtendedContainerInfo {
				t.AppendRow([]interface{}{
					containerImage,
					container.Names[0][1:],
					time.Unix(container.Created, 0).Format("2006.01.02 15:04:05"),
					container.State,
					container.Status,
					container.Command, // note: Ports/Mounts not used yet in non-extended view
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

func prettifyPortsList(ports []PortsStruct) string {
	var portsString, sourcePort string
	for _, val := range ports {
		if val.PublicPort == 0 {
			sourcePort = ""
		} else {
			sourcePort = fmt.Sprintf("%d->", val.PublicPort)
		}
		portsString += fmt.Sprintf("%s/%s%d\n", val.Type, sourcePort, val.PrivatePort)
	}
	return portsString
}

// Standardizes the image:tag format, adding :latest when the tag is missing.
// Handles registry prefixes with or without ports.
func getImageTag(name string) string {
	slashIndex := strings.LastIndex(name, "/")
	colonIndex := strings.LastIndex(name, ":")

	// If the last colon comes after the last slash, we already have a tag.
	if colonIndex > slashIndex {
		return name
	}
	return name + ":latest"
}

func prettifyMounts(mounts []MountsStruct) string {
	mountspecs := ""
	for _, mount := range mounts {
		src := ""
		if mount.Type != "bind" {
			src = "[" + mount.Source + "]"
		} else {
			src = mount.Source
		}
		if mount.RW {
			mountspecs += fmt.Sprintf("%s %s:%s\n", hftx.EnabledSign(""), src, mount.Destination)
		} else {
			mountspecs += fmt.Sprintf("%s %s:%s\n", hftx.ErrorSign(""), src, mount.Destination)
		}
	}
	return mountspecs
}
