// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/31 13:49
// Original filename: src/volumes/list.go

package volumes

import (
	"dtools2/containers"
	"dtools2/rest"
	"fmt"
	"os"
	"strings"
	"time"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func formatCreated(created string) string {
	created = strings.TrimSpace(created)
	if created == "" {
		return ""
	}

	// Docker/Podman commonly return RFC3339 timestamps; sometimes with fractional seconds.
	tm, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		tm, err = time.Parse(time.RFC3339, created)
		if err != nil {
			// If parsing fails, keep original to avoid losing information.
			return created
		}
	}

	return tm.Format("2006.01.02 15:04:05")
}

func ListVolumes(client *rest.Client) *ce.CustomError {
	// 1) Fetch volumes
	vols, verr := fetchVolumeList(client)
	if verr != nil {
		return verr
	}

	// 2) Fetch containers (include stopped; they still reference volumes)
	oldOnlyRunning := containers.OnlyRunningContainers
	containers.OnlyRunningContainers = false
	cs, cerr := containers.ListContainers(client, false)
	containers.OnlyRunningContainers = oldOnlyRunning
	if cerr != nil {
		return &ce.CustomError{Title: cerr.Title, Message: cerr.Message}
	}

	// 3) Build volume -> containers lookup (O(vols + containers))
	usedBy := computeVolumeUsage(cs)

	// 4) Render output
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Name", "Driver", "Scope", "Created", "RefCount", "Used by"})

	if len(vols) == 0 {
		t.AppendRow(table.Row{"", "", "", "", "", ""})
	} else {
		for _, v := range vols {
			users := usedBy[v.Name]
			refCount := len(users) // computed from container mounts, not API UsageData
			usedByStr := strings.Join(users, "\n")

			t.AppendRow(table.Row{
				v.Name,
				v.Driver,
				v.Scope,
				formatCreated(v.CreatedAt),
				refCount,
				usedByStr,
			})
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
	return nil
}
