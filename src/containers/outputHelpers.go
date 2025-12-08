// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/25 18:35
// Original filename: src/containers/outputHelpers.go

package containers

import (
	"fmt"
	"math"
	"strings"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

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

// Formats the ports list to make it more human-readable
func prettifyPortsList(ports []PortsStruct, delimiter string) string {
	seen := make(map[string]struct{})
	var portsString, sourcePort string

	for ndx, val := range ports {
		key := fmt.Sprintf("%s-%d-%d", val.Type, val.PublicPort, val.PrivatePort)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		if val.PublicPort == 0 {
			sourcePort = ""
		} else {
			sourcePort = fmt.Sprintf("%d->", val.PublicPort)
		}
		if ndx < len(ports)-1 {
			portsString += fmt.Sprintf("%s/%s%d%s", val.Type, sourcePort, val.PrivatePort, delimiter)
		} else {
			portsString += fmt.Sprintf("%s/%s%d", val.Type, sourcePort, val.PrivatePort)
		}
	}
	return portsString
}

// Same principle here as for prettifyPortsList
func prettifyMounts(mounts []MountsStruct, delimiter string) string {
	var mountspecs string

	for ndx, mount := range mounts {
		// Last item: no trailing delimiter
		if ndx == len(mounts)-1 {
			delimiter = ""
		}

		var src string

		switch mount.Type {
		case "bind":
			// Bind mounts: show the host path
			src = mount.Source

		case "volume":
			// Named volumes: show the volume name, not the host path
			if mount.Name != "" {
				src = "[" + mount.Name + "]"
			} else if mount.Source != "" {
				// Anonymous volume: fall back to source path
				src = "[" + mount.Source + "]"
			} else {
				src = "[<anonymous>]"
			}

		default:
			// Anything else (tmpfs, npipe, etc): fall back to Source if present
			if mount.Source != "" {
				src = "[" + mount.Source + "]"
			} else {
				src = "[?]"
			}
		}

		if mount.RW {
			mountspecs += fmt.Sprintf("%s %s:%s%s",
				hftx.EnabledSign(""), src, mount.Destination, delimiter)
		} else {
			mountspecs += fmt.Sprintf("%s %s:%s%s",
				hftx.ErrorSign(""), src, mount.Destination, delimiter)
		}
	}

	return mountspecs
}

func formatSize(sz int64) string {
	numSize := (float64)(sz) / 1000.0 / 1000.0 // this will give us the size in MB
	if (int)(math.Log10(float64(numSize))) > 2 {
		return fmt.Sprintf("%.3f GB", numSize/1000.0)
	} else {
		return fmt.Sprintf("%.3f MB", numSize)
	}
}
