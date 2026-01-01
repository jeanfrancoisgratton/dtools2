// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/01
// Original filename: src/volumes/helpers.go

package volumes

import (
	"bytes"
	"dtools2/containers"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// fetchVolumeList fetches the daemon's volumes.
//
// Docker typically returns an object shaped like VolumeListResponse:
//   {"Volumes":[...],"Warnings":[...]}
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
