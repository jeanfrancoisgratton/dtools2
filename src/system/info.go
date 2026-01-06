// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/06 02:31
// Original filename: src/system/info.go

package system

import (
	"bytes"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfjson "github.com/jeanfrancoisgratton/helperFunctions/v4/prettyjson"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// Info fetches the daemon's /info payload and renders an output that mirrors the
// "Server" section of `docker info`.
//
// NOTE: this intentionally does not print a "Client" section.
func Info(client *rest.Client) *ce.CustomError {
	resp, err := client.Do(rest.Context, http.MethodGet, "/info", url.Values{}, nil, nil)
	if err != nil {
		return &ce.CustomError{Title: "Unable to fetch daemon info", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &ce.CustomError{Title: "http request returned an error", Message: "GET /info returned " + resp.Status}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ce.CustomError{Title: "Unable to read HTTP response", Message: err.Error()}
	}
	trim := bytes.TrimSpace(body)
	if len(trim) == 0 {
		return &ce.CustomError{Title: "Unexpected empty payload", Message: "GET /info returned an empty response body"}
	}

	var info InfoResponse
	if uerr := json.Unmarshal(trim, &info); uerr != nil {
		// Some daemons (or future API versions) may return a different schema.
		// Fall back to a raw JSON print rather than failing hard.
		hfjson.Print(trim)
		return nil
	}

	printInfo(info)
	return nil
}

func printInfo(info InfoResponse) {
	fmt.Println(hftx.Blue("Server:"))

	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)

	// Core summary.
	printKV(w, "Containers", fmt.Sprintf("%d", info.Containers))
	printKV(w, " Running", fmt.Sprintf("%d", info.ContainersRunning))
	printKV(w, " Paused", fmt.Sprintf("%d", info.ContainersPaused))
	printKV(w, " Stopped", fmt.Sprintf("%d", info.ContainersStopped))
	if info.Images != 0 {
		printKV(w, "Images", fmt.Sprintf("%d", info.Images))
	}

	if info.ServerVersion != "" {
		printKV(w, "Server Version", info.ServerVersion)
	}
	if info.KernelVersion != "" {
		printKV(w, "Kernel Version", info.KernelVersion)
	}
	if info.OperatingSystem != "" {
		printKV(w, "Operating System", info.OperatingSystem)
	}
	if info.OSType != "" {
		printKV(w, "OSType", info.OSType)
	}
	if info.Architecture != "" {
		printKV(w, "Architecture", info.Architecture)
	}
	if info.NCPU != 0 {
		printKV(w, "CPUs", fmt.Sprintf("%d", info.NCPU))
	}
	if info.MemTotal != 0 {
		printKV(w, "Total Memory", formatBytesBinary(info.MemTotal))
	}
	if info.Name != "" {
		printKV(w, "Name", info.Name)
	}
	if info.ID != "" {
		printKV(w, "ID", info.ID)
	}
	if info.DockerRootDir != "" {
		printKV(w, "Docker Root Dir", info.DockerRootDir)
	}
	if info.SystemTime != "" {
		printKV(w, "System Time", info.SystemTime)
	}

	// Drivers.
	if info.Driver != "" {
		printKV(w, "Storage Driver", info.Driver)
		for _, kv := range info.DriverStatus {
			if len(kv) == 2 {
				printKV(w, " "+kv[0], kv[1])
			}
		}
	}
	if info.LoggingDriver != "" {
		printKV(w, "Logging Driver", info.LoggingDriver)
	}
	if info.CgroupDriver != "" {
		printKV(w, "Cgroup Driver", info.CgroupDriver)
	}
	if info.CgroupVersion != "" {
		printKV(w, "Cgroup Version", info.CgroupVersion)
	}

	// Plugins.
	if !info.Plugins.Empty() {
		printKV(w, "Plugins", "")
		if len(info.Plugins.Volume) > 0 {
			printKV(w, " Volume", strings.Join(info.Plugins.Volume, " "))
		}
		if len(info.Plugins.Network) > 0 {
			printKV(w, " Network", strings.Join(info.Plugins.Network, " "))
		}
		if len(info.Plugins.Authorization) > 0 {
			printKV(w, " Authorization", strings.Join(info.Plugins.Authorization, " "))
		}
		if len(info.Plugins.Log) > 0 {
			printKV(w, " Log", strings.Join(info.Plugins.Log, " "))
		}
	}

	// Swarm.
	if info.Swarm.LocalNodeState != "" {
		printKV(w, "Swarm", strings.ToLower(info.Swarm.LocalNodeState))
	}

	// Runtimes.
	if len(info.Runtimes) > 0 {
		keys := make([]string, 0, len(info.Runtimes))
		for k := range info.Runtimes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		printKV(w, "Runtimes", strings.Join(keys, " "))
	}
	if info.DefaultRuntime != "" {
		printKV(w, "Default Runtime", info.DefaultRuntime)
	}
	if info.InitBinary != "" {
		printKV(w, "Init Binary", info.InitBinary)
	}

	// Security options.
	if len(info.SecurityOptions) > 0 {
		printKV(w, "Security Options", "")
		for _, s := range info.SecurityOptions {
			printKV(w, " ", s)
		}
	}

	// Proxies.
	if info.HTTPProxy != "" {
		printKV(w, "HTTP Proxy", info.HTTPProxy)
	}
	if info.HTTPSProxy != "" {
		printKV(w, "HTTPS Proxy", info.HTTPSProxy)
	}
	if info.NoProxy != "" {
		printKV(w, "No Proxy", info.NoProxy)
	}

	// Registry.
	if info.IndexServerAddress != "" {
		printKV(w, "Registry", info.IndexServerAddress)
	}
	if len(info.RegistryConfig.Mirrors) > 0 {
		printKV(w, "Registry Mirrors", strings.Join(info.RegistryConfig.Mirrors, ", "))
	}
	if len(info.RegistryConfig.InsecureRegistryCIDRs) > 0 {
		printKV(w, "Insecure Registries", "")
		for _, cidr := range info.RegistryConfig.InsecureRegistryCIDRs {
			printKV(w, " ", cidr)
		}
	}

	// Misc.
	printKV(w, "Debug Mode", formatBool(info.Debug))
	if info.ServerVersion != "" {
		printKV(w, "Experimental", formatBool(info.ExperimentalBuild))
		printKV(w, "Live Restore Enabled", formatBool(info.LiveRestoreEnabled))
	}

	w.Flush()
	fmt.Println()
}

func printKV(w *tabwriter.Writer, k, v string) {
	if v == "" {
		fmt.Fprintf(w, "%s\t\n", hftx.Blue(k))
		return
	}
	fmt.Fprintf(w, "%s\t%s\n", hftx.Blue(k), v)
}

func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func formatBytesBinary(b int64) string {
	if b == 0 {
		return "0 B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div := int64(unit)
	exp := 0
	for n := b / unit; n >= unit && exp < 6; n /= unit {
		div *= unit
		exp++
	}
	suffix := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB", "ZiB"}
	return fmt.Sprintf("%.2f %s", float64(b)/float64(div), suffix[exp])
}

func (p PluginsInfo) Empty() bool {
	return len(p.Volume) == 0 && len(p.Network) == 0 && len(p.Authorization) == 0 && len(p.Log) == 0
}
