// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/01
// Original filename: src/volumes/helpers.go

package volumes

import (
	"dtools2/containers"
	"fmt"
	"sort"
	"strings"
	"time"
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

func containerDisplayName(c containers.ContainerSummary) string {
	name := ""
	if len(c.Names) > 0 {
		name = strings.TrimPrefix(c.Names[0], "/")
	}
	if name == "" {
		if len(c.ID) >= 12 {
			name = c.ID[:12]
		} else {
			name = c.ID
		}
	}
	if c.State != "" {
		return fmt.Sprintf("%s (%s)", name, c.State)
	}
	return name
}

// computeVolumeUsage builds a lookup map from volume name to the list of containers
// that reference that volume, based solely on GET /containers/json.
//
// This keeps the operation at 2 API calls total when combined with GET /volumes.
func computeVolumeUsage(cs []containers.ContainerSummary) map[string][]string {
	usedBy := make(map[string][]string)
	seen := make(map[string]map[string]struct{})

	for _, c := range cs {
		cname := containerDisplayName(c)
		for _, m := range c.Mounts {
			if m.Type != "volume" || m.Name == "" {
				continue
			}
			if _, ok := seen[m.Name]; !ok {
				seen[m.Name] = make(map[string]struct{})
			}
			if _, ok := seen[m.Name][cname]; ok {
				continue
			}
			seen[m.Name][cname] = struct{}{}
			usedBy[m.Name] = append(usedBy[m.Name], cname)
		}
	}

	for v := range usedBy {
		sort.Strings(usedBy[v])
	}

	return usedBy
}
