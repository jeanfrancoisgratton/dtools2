// extras/inspect_helpers.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/02/01

package extras

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
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
			if v, ok := s["FinishedAt"].(string); ok && v != "" && v != "0001-01-01T00:00:00Z" {
				FormatInspectField(w, "Finished At", FormatTimestamp(v))
			}
		}
	}

	// SIZE
	if sizeRw > 0 || sizeRootFs > 0 {
		FormatInspectSection(w, "Size")
		if sizeRw > 0 {
			FormatInspectField(w, "Read/Write Layer", FormatSize(sizeRw))
		}
		if sizeRootFs > 0 {
			FormatInspectField(w, "Root Filesystem", FormatSize(sizeRootFs))
		}
	}

	// CONFIG
	if config != nil {
		if c, ok := config.(map[string]interface{}); ok {
			FormatInspectSection(w, "Configuration")

			if v, ok := c["Hostname"].(string); ok && v != "" {
				FormatInspectField(w, "Hostname", v)
			}
			if v, ok := c["User"].(string); ok && v != "" {
				FormatInspectField(w, "User", v)
			}
			if v, ok := c["WorkingDir"].(string); ok && v != "" {
				FormatInspectField(w, "Working Directory", v)
			}

			if v, ok := c["Env"].([]interface{}); ok && len(v) > 0 {
				envStrs := make([]string, 0, len(v))
				for _, e := range v {
					if s, ok := e.(string); ok {
						envStrs = append(envStrs, s)
					}
				}
				if len(envStrs) > 0 {
					FormatInspectList(w, "Environment", envStrs)
				}
			}

			if v, ok := c["Cmd"].([]interface{}); ok && len(v) > 0 {
				cmdStrs := make([]string, 0, len(v))
				for _, e := range v {
					if s, ok := e.(string); ok {
						cmdStrs = append(cmdStrs, s)
					}
				}
				if len(cmdStrs) > 0 {
					FormatInspectField(w, "Command", strings.Join(cmdStrs, " "))
				}
			}

			if v, ok := c["Entrypoint"].([]interface{}); ok && len(v) > 0 {
				epStrs := make([]string, 0, len(v))
				for _, e := range v {
					if s, ok := e.(string); ok {
						epStrs = append(epStrs, s)
					}
				}
				if len(epStrs) > 0 {
					FormatInspectField(w, "Entrypoint", strings.Join(epStrs, " "))
				}
			}

			if v, ok := c["Labels"].(map[string]interface{}); ok && len(v) > 0 {
				labels := make(map[string]string)
				for k, val := range v {
					if s, ok := val.(string); ok {
						labels[k] = s
					}
				}
				if len(labels) > 0 {
					FormatInspectMap(w, "Labels", labels)
				}
			}
		}
	}

	// NETWORK
	if networkSettings != nil {
		if n, ok := networkSettings.(map[string]interface{}); ok {
			networks, hasNetworks := n["Networks"].(map[string]interface{})
			ipAddr, hasIP := n["IPAddress"].(string)
			gateway, hasGateway := n["Gateway"].(string)

			if hasNetworks || hasIP || hasGateway {
				FormatInspectSection(w, "Network")

				if hasIP && ipAddr != "" {
					FormatInspectField(w, "IP Address", ipAddr)
				}
				if hasGateway && gateway != "" {
					FormatInspectField(w, "Gateway", gateway)
				}

				if hasNetworks && len(networks) > 0 {
					fmt.Fprintf(w, "  %s:\n", hftx.Blue("Networks"))
					for netName, netData := range networks {
						if nd, ok := netData.(map[string]interface{}); ok {
							fmt.Fprintf(w, "    %s:\n", hftx.Yellow(netName))
							if ip, ok := nd["IPAddress"].(string); ok && ip != "" {
								fmt.Fprintf(w, "      IP Address: %s\n", ip)
							}
							if gw, ok := nd["Gateway"].(string); ok && gw != "" {
								fmt.Fprintf(w, "      Gateway: %s\n", gw)
							}
							if mac, ok := nd["MacAddress"].(string); ok && mac != "" {
								fmt.Fprintf(w, "      MAC Address: %s\n", mac)
							}
						}
					}
				}
			}
		}
	}

	// MOUNTS
	if mounts != nil {
		if m, ok := mounts.([]interface{}); ok && len(m) > 0 {
			FormatInspectSection(w, "Mounts")
			for i, mount := range m {
				if md, ok := mount.(map[string]interface{}); ok {
					fmt.Fprintf(w, "  Mount %d:\n", i+1)
					if v, ok := md["Type"].(string); ok {
						fmt.Fprintf(w, "    Type: %s\n", v)
					}
					if v, ok := md["Source"].(string); ok {
						fmt.Fprintf(w, "    Source: %s\n", v)
					}
					if v, ok := md["Destination"].(string); ok {
						fmt.Fprintf(w, "    Destination: %s\n", v)
					}
					if v, ok := md["RW"].(bool); ok {
						mode := "read-only"
						if v {
							mode = "read-write"
						}
						fmt.Fprintf(w, "    Mode: %s\n", mode)
					}
				}
			}
		}
	}

	// HOST CONFIG highlights
	if hostConfig != nil {
		if h, ok := hostConfig.(map[string]interface{}); ok {
			hasContent := false

			// Check if we have any interesting host config to show
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
