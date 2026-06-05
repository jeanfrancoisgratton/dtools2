// extras/inspect_helpers.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/02/01

package extras

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// FormatInspectSection is a reusable helper to print a section header
func FormatInspectSection(w *tabwriter.Writer, title string) {
	fmt.Fprintf(w, "\n%s\n", hftx.Blue(strings.ToUpper(title)))
}

// FormatInspectField prints a key-value pair with proper formatting
func FormatInspectField(w *tabwriter.Writer, key, value string) {
	if value != "" {
		fmt.Fprintf(w, "  %s:\t%s\n", hftx.Blue(key), value)
	}
}

// FormatInspectFieldInt prints an integer field
func FormatInspectFieldInt(w *tabwriter.Writer, key string, value int) {
	fmt.Fprintf(w, "  %s:\t%d\n", hftx.Blue(key), value)
}

// FormatInspectFieldInt64 prints an int64 field
func FormatInspectFieldInt64(w *tabwriter.Writer, key string, value int64) {
	fmt.Fprintf(w, "  %s:\t%d\n", hftx.Blue(key), value)
}

// FormatInspectFieldBool prints a boolean field
func FormatInspectFieldBool(w *tabwriter.Writer, key string, value bool) {
	valStr := hftx.Red("false")
	if value {
		valStr = hftx.Green("true")
	}
	fmt.Fprintf(w, "  %s:\t%s\n", hftx.Blue(key), valStr)
}

// FormatInspectList prints a list of strings
func FormatInspectList(w *tabwriter.Writer, key string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(w, "  %s:\n", hftx.Blue(key))
	for _, v := range values {
		fmt.Fprintf(w, "    - %s\n", v)
	}
}

// FormatInspectMap prints a map of key-value pairs
func FormatInspectMap(w *tabwriter.Writer, key string, values map[string]string) {
	if len(values) == 0 {
		return
	}
	fmt.Fprintf(w, "  %s:\n", hftx.Blue(key))
	for k, v := range values {
		fmt.Fprintf(w, "    %s: %s\n", k, v)
	}
}

// FormatSize converts bytes to human-readable format
func FormatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	if exp >= len(units) {
		exp = len(units) - 1
	}

	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// FormatTimestamp parses and formats an RFC3339 timestamp
func FormatTimestamp(timestamp string) string {
	if timestamp == "" || timestamp == "0001-01-01T00:00:00Z" {
		return "N/A"
	}

	t, err := time.Parse(time.RFC3339Nano, timestamp)
	if err != nil {
		// Try without nano precision
		t, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return timestamp // Return as-is if parsing fails
		}
	}

	return t.Format("2006.01.02 15:04:05")
}

// FormatState returns a colored state string
func FormatState(state string, running bool) string {
	if running {
		return hftx.Green("running")
	}

	switch strings.ToLower(state) {
	case "paused", "suspended":
		return hftx.Yellow(state)
	case "exited", "stopped":
		return hftx.White(state)
	case "dead", "crashed":
		return hftx.Red(state)
	case "created":
		return hftx.Blue(state)
	default:
		return state
	}
}

// PrintContainerInspectData is the actual implementation for containers
// This should be called from containers package after importing extras
func PrintContainerInspectData(
	w *tabwriter.Writer,
	id, name, created string,
	state interface{},
	image, platform, driver string,
	sizeRw, sizeRootFs int64,
	config interface{},
	hostConfig interface{},
	networkSettings interface{},
	mounts interface{},
) {
	// CONTAINER INFO
	FormatInspectSection(w, "Container Information")
	FormatInspectField(w, "ID", id)
	FormatInspectField(w, "Name", strings.TrimPrefix(name, "/"))
	FormatInspectField(w, "Created", FormatTimestamp(created))
	FormatInspectField(w, "Image", image)
	if platform != "" {
		FormatInspectField(w, "Platform", platform)
	}
	if driver != "" {
		FormatInspectField(w, "Storage Driver", driver)
	}

	// STATE
	if state != nil {
		if s, ok := state.(map[string]interface{}); ok {
			FormatInspectSection(w, "State")

			status := ""
			running := false
			if v, ok := s["Status"].(string); ok {
				status = v
			}
			if v, ok := s["Running"].(bool); ok {
				running = v
			}

			FormatInspectField(w, "Status", FormatState(status, running))
			FormatInspectFieldBool(w, "Running", running)

			if v, ok := s["Paused"].(bool); ok && v {
				FormatInspectFieldBool(w, "Paused", v)
			}
			if v, ok := s["Restarting"].(bool); ok && v {
				FormatInspectFieldBool(w, "Restarting", v)
			}
			if v, ok := s["Dead"].(bool); ok && v {
				FormatInspectFieldBool(w, "Dead", v)
			}
			if v, ok := s["OOMKilled"].(bool); ok && v {
				FormatInspectFieldBool(w, "OOMKilled", v)
			}

			if v, ok := s["Pid"].(float64); ok && v > 0 {
				FormatInspectFieldInt(w, "PID", int(v))
			}
			if v, ok := s["ExitCode"].(float64); ok && (v != 0 || !running) {
				FormatInspectFieldInt(w, "Exit Code", int(v))
			}
			if v, ok := s["Error"].(string); ok && v != "" {
				FormatInspectField(w, "Error", hftx.Red(v))
			}
			if v, ok := s["StartedAt"].(string); ok && v != "" {
				FormatInspectField(w, "Started At", FormatTimestamp(v))
			}
			if v, ok := s["FinishedAt"].(string); ok && v != "" {
				FormatInspectField(w, "Finished At", FormatTimestamp(v))
			}
		}
	}

	// SIZES
	if sizeRw > 0 || sizeRootFs > 0 {
		FormatInspectSection(w, "Size")
		if sizeRw > 0 {
			FormatInspectField(w, "RW Layer", FormatSize(sizeRw))
		}
		if sizeRootFs > 0 {
			FormatInspectField(w, "Root FS", FormatSize(sizeRootFs))
		}
	}

	// CONFIG
	if config != nil {
		if c, ok := config.(map[string]interface{}); ok {
			FormatInspectSection(w, "Config")

			if v, ok := c["Hostname"].(string); ok && v != "" {
				FormatInspectField(w, "Hostname", v)
			}
			if v, ok := c["Domainname"].(string); ok && v != "" {
				FormatInspectField(w, "Domainname", v)
			}
			if v, ok := c["User"].(string); ok && v != "" {
				FormatInspectField(w, "User", v)
			}
			if v, ok := c["WorkingDir"].(string); ok && v != "" {
				FormatInspectField(w, "WorkingDir", v)
			}
			if v, ok := c["Entrypoint"].([]interface{}); ok && len(v) > 0 {
				var parts []string
				for _, p := range v {
					if s, ok := p.(string); ok {
						parts = append(parts, s)
					}
				}
				FormatInspectField(w, "Entrypoint", strings.Join(parts, " "))
			}
			if v, ok := c["Cmd"].([]interface{}); ok && len(v) > 0 {
				var parts []string
				for _, p := range v {
					if s, ok := p.(string); ok {
						parts = append(parts, s)
					}
				}
				FormatInspectField(w, "Cmd", strings.Join(parts, " "))
			}
			if v, ok := c["Env"].([]interface{}); ok && len(v) > 0 {
				FormatInspectSection(w, "Environment")
				for _, e := range v {
					if s, ok := e.(string); ok && s != "" {
						fmt.Fprintf(w, "  - %s\n", s)
					}
				}
			}
			if v, ok := c["Labels"].(map[string]interface{}); ok && len(v) > 0 {
				FormatInspectSection(w, "Labels")
				for k, vv := range v {
					FormatInspectField(w, k, fmt.Sprint(vv))
				}
			}
		}
	}

	// HOST CONFIG (best-effort subset)
	if hostConfig != nil {
		if h, ok := hostConfig.(map[string]interface{}); ok {
			hasContent := false

			if v, ok := h["NetworkMode"].(string); ok && v != "" && v != "default" {
				if !hasContent {
					FormatInspectSection(w, "Host Configuration")
					hasContent = true
				}
				FormatInspectField(w, "Network Mode", v)
			}
			if v, ok := h["Privileged"].(bool); ok && v {
				if !hasContent {
					FormatInspectSection(w, "Host Configuration")
					hasContent = true
				}
				FormatInspectFieldBool(w, "Privileged", v)
			}
			if v, ok := h["RestartPolicy"].(map[string]interface{}); ok {
				if name, ok := v["Name"].(string); ok && name != "" && name != "no" {
					if !hasContent {
						FormatInspectSection(w, "Host Configuration")
						hasContent = true
					}
					FormatInspectField(w, "Restart Policy", name)
					if count, ok := v["MaximumRetryCount"].(float64); ok && count > 0 {
						FormatInspectFieldInt(w, "Max Retry Count", int(count))
					}
				}
			}
		}
	}

	w.Flush()
}

// PrintNetworkInspectData prints a human-friendly "network inspect" output.
// This is called from networks.InspectNetwork.
func PrintNetworkInspectData(w *tabwriter.Writer, n interface{}) {
	if n == nil {
		return
	}

	// Convert to map for flexible access (Docker/Podman may differ a bit).
	jsonBytes, err := json.Marshal(n)
	if err != nil {
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return
	}

	FormatInspectSection(w, "Network Information")
	if v, ok := m["Name"].(string); ok {
		FormatInspectField(w, "Name", v)
	}
	if v, ok := m["Id"].(string); ok {
		FormatInspectField(w, "ID", v)
	}
	if v, ok := m["Created"].(string); ok {
		FormatInspectField(w, "Created", FormatTimestamp(v))
	}
	if v, ok := m["Scope"].(string); ok {
		FormatInspectField(w, "Scope", v)
	}
	if v, ok := m["Driver"].(string); ok {
		FormatInspectField(w, "Driver", v)
	}
	if v, ok := m["Internal"].(bool); ok {
		FormatInspectFieldBool(w, "Internal", v)
	}
	if v, ok := m["Attachable"].(bool); ok {
		FormatInspectFieldBool(w, "Attachable", v)
	}
	if v, ok := m["Ingress"].(bool); ok {
		FormatInspectFieldBool(w, "Ingress", v)
	}
	if v, ok := m["EnableIPv6"].(bool); ok {
		FormatInspectFieldBool(w, "EnableIPv6", v)
	}
	if v, ok := m["ConfigOnly"].(bool); ok && v {
		FormatInspectFieldBool(w, "ConfigOnly", v)
	}
	if v, ok := m["ConfigFrom"].(map[string]interface{}); ok {
		if netName, ok := v["Network"].(string); ok && netName != "" {
			FormatInspectField(w, "ConfigFrom", netName)
		}
	}

	// IPAM
	if ipam, ok := m["IPAM"].(map[string]interface{}); ok {
		FormatInspectSection(w, "IPAM")
		if drv, ok := ipam["Driver"].(string); ok {
			FormatInspectField(w, "Driver", drv)
		}
		if cfg, ok := ipam["Config"].([]interface{}); ok && len(cfg) > 0 {
			for i, c := range cfg {
				cm, _ := c.(map[string]interface{})
				if cm == nil {
					continue
				}
				FormatInspectField(w, fmt.Sprintf("Config %d", i+1), "")
				if v, ok := cm["Subnet"].(string); ok && v != "" {
					FormatInspectField(w, "Subnet", v)
				}
				if v, ok := cm["IPRange"].(string); ok && v != "" {
					FormatInspectField(w, "IPRange", v)
				}
				if v, ok := cm["Gateway"].(string); ok && v != "" {
					FormatInspectField(w, "Gateway", v)
				}
			}
		}
	}

	// Options/Labels
	if opts, ok := m["Options"].(map[string]interface{}); ok && len(opts) > 0 {
		FormatInspectSection(w, "Options")
		for k, v := range opts {
			FormatInspectField(w, k, fmt.Sprint(v))
		}
	}
	if labels, ok := m["Labels"].(map[string]interface{}); ok && len(labels) > 0 {
		FormatInspectSection(w, "Labels")
		for k, v := range labels {
			FormatInspectField(w, k, fmt.Sprint(v))
		}
	}

	// Attached containers
	if containers, ok := m["Containers"].(map[string]interface{}); ok && len(containers) > 0 {
		FormatInspectSection(w, "Attached Containers")
		for id, raw := range containers {
			cm, _ := raw.(map[string]interface{})
			name, _ := cm["Name"].(string)
			ipv4, _ := cm["IPv4Address"].(string)
			ipv6, _ := cm["IPv6Address"].(string)

			line := id
			if name != "" {
				line = fmt.Sprintf("%s (%s)", id, name)
			}
			if ipv4 != "" {
				line = fmt.Sprintf("%s  IPv4=%s", line, ipv4)
			}
			if ipv6 != "" {
				line = fmt.Sprintf("%s  IPv6=%s", line, ipv6)
			}
			fmt.Fprintf(w, "  - %s\n", line)
		}
	}
}

// PrintVolumeInspectData prints a human-friendly "volume inspect" output.
// This is called from volumes.InspectVolume.
func PrintVolumeInspectData(w *tabwriter.Writer, v interface{}) {
	if v == nil {
		return
	}

	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return
	}

	FormatInspectSection(w, "Volume Information")
	if s, ok := m["Name"].(string); ok {
		FormatInspectField(w, "Name", s)
	}
	if s, ok := m["Driver"].(string); ok {
		FormatInspectField(w, "Driver", s)
	}
	if s, ok := m["Mountpoint"].(string); ok {
		FormatInspectField(w, "Mountpoint", s)
	}
	if s, ok := m["Scope"].(string); ok {
		FormatInspectField(w, "Scope", s)
	}
	if s, ok := m["CreatedAt"].(string); ok {
		FormatInspectField(w, "CreatedAt", s)
	}

	if opts, ok := m["Options"].(map[string]interface{}); ok && len(opts) > 0 {
		FormatInspectSection(w, "Options")
		for k, val := range opts {
			FormatInspectField(w, k, fmt.Sprint(val))
		}
	}
	if labels, ok := m["Labels"].(map[string]interface{}); ok && len(labels) > 0 {
		FormatInspectSection(w, "Labels")
		for k, val := range labels {
			FormatInspectField(w, k, fmt.Sprint(val))
		}
	}
	if status, ok := m["Status"].(map[string]interface{}); ok && len(status) > 0 {
		FormatInspectSection(w, "Status")
		for k, val := range status {
			FormatInspectField(w, k, fmt.Sprint(val))
		}
	}
	if usage, ok := m["UsageData"].(map[string]interface{}); ok && len(usage) > 0 {
		FormatInspectSection(w, "Usage")
		if sz, ok := usage["Size"].(float64); ok {
			FormatInspectField(w, "Size", FormatSize(int64(sz)))
		}
		if rc, ok := usage["RefCount"].(float64); ok {
			FormatInspectFieldInt64(w, "RefCount", int64(rc))
		}
	}
}

// PrintImageInspectData prints a human-friendly "image inspect" output.
// This is called from images.InspectImage.
func PrintImageInspectData(w *tabwriter.Writer, img interface{}) {
	if img == nil {
		return
	}

	// Convert to map for flexible access (Docker/Podman may differ a bit).
	jsonBytes, err := json.Marshal(img)
	if err != nil {
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return
	}

	FormatInspectSection(w, "Image Information")
	if v, ok := m["Id"].(string); ok {
		FormatInspectField(w, "ID", v)
	}
	if v, ok := m["Created"].(string); ok {
		FormatInspectField(w, "Created", FormatTimestamp(v))
	}
	if v, ok := m["Author"].(string); ok && v != "" {
		FormatInspectField(w, "Author", v)
	}
	if v, ok := m["Architecture"].(string); ok && v != "" {
		FormatInspectField(w, "Architecture", v)
	}
	if v, ok := m["Os"].(string); ok && v != "" {
		FormatInspectField(w, "OS", v)
	}
	if v, ok := m["Variant"].(string); ok && v != "" {
		FormatInspectField(w, "Variant", v)
	}
	if v, ok := m["Size"].(float64); ok && v > 0 {
		FormatInspectField(w, "Size", FormatSize(int64(v)))
	}
	if v, ok := m["VirtualSize"].(float64); ok && v > 0 {
		FormatInspectField(w, "VirtualSize", FormatSize(int64(v)))
	}

	if tags, ok := m["RepoTags"].([]interface{}); ok && len(tags) > 0 {
		FormatInspectSection(w, "RepoTags")
		for _, t := range tags {
			if s, ok := t.(string); ok && s != "" {
				fmt.Fprintf(w, "  - %s\n", s)
			}
		}
	}
	if digs, ok := m["RepoDigests"].([]interface{}); ok && len(digs) > 0 {
		FormatInspectSection(w, "RepoDigests")
		for _, d := range digs {
			if s, ok := d.(string); ok && s != "" {
				fmt.Fprintf(w, "  - %s\n", s)
			}
		}
	}

	// Config (best-effort subset)
	if cfg, ok := m["Config"].(map[string]interface{}); ok && len(cfg) > 0 {
		FormatInspectSection(w, "Config")

		if v, ok := cfg["WorkingDir"].(string); ok && v != "" {
			FormatInspectField(w, "WorkingDir", v)
		}
		if v, ok := cfg["User"].(string); ok && v != "" {
			FormatInspectField(w, "User", v)
		}
		if v, ok := cfg["Entrypoint"].([]interface{}); ok && len(v) > 0 {
			var parts []string
			for _, p := range v {
				if s, ok := p.(string); ok {
					parts = append(parts, s)
				}
			}
			if len(parts) > 0 {
				FormatInspectField(w, "Entrypoint", strings.Join(parts, " "))
			}
		}
		if v, ok := cfg["Cmd"].([]interface{}); ok && len(v) > 0 {
			var parts []string
			for _, p := range v {
				if s, ok := p.(string); ok {
					parts = append(parts, s)
				}
			}
			if len(parts) > 0 {
				FormatInspectField(w, "Cmd", strings.Join(parts, " "))
			}
		}
		if v, ok := cfg["Env"].([]interface{}); ok && len(v) > 0 {
			FormatInspectSection(w, "Environment")
			for _, e := range v {
				if s, ok := e.(string); ok && s != "" {
					fmt.Fprintf(w, "  - %s\n", s)
				}
			}
		}
		if v, ok := cfg["Labels"].(map[string]interface{}); ok && len(v) > 0 {
			FormatInspectSection(w, "Labels")
			for k, vv := range v {
				FormatInspectField(w, k, fmt.Sprint(vv))
			}
		}
	}

	// RootFS (best-effort)
	if rootfs, ok := m["RootFS"].(map[string]interface{}); ok && len(rootfs) > 0 {
		FormatInspectSection(w, "RootFS")
		if t, ok := rootfs["Type"].(string); ok && t != "" {
			FormatInspectField(w, "Type", t)
		}
		if layers, ok := rootfs["Layers"].([]interface{}); ok && len(layers) > 0 {
			fmt.Fprintf(w, "  %s:\n", hftx.Blue("Layers"))
			for _, l := range layers {
				if s, ok := l.(string); ok && s != "" {
					fmt.Fprintf(w, "    - %s\n", s)
				}
			}
		}
	}

	w.Flush()
}
