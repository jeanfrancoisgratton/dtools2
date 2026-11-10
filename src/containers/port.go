// containers/port.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/11/10
// Original filename: src/containers/port.go

package containers

import (
	"dtools2/extras"
	"dtools2/rest"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfjson "github.com/jeanfrancoisgratton/helperFunctions/v5/prettyjson"
)

// PortMapping is one published container port -> host address binding
type PortMapping struct {
	ContainerPort int    `json:"ContainerPort"`
	Protocol      string `json:"Protocol"`
	HostIP        string `json:"HostIP"`
	HostPort      string `json:"HostPort"`
}

// PortContainer mimics `docker port`: it lists the published port mappings of a container,
// optionally filtered on a single PRIVATE_PORT[/PROTO] (protocol defaults to tcp)
func PortContainer(client *rest.Client, container, filter string) ([]PortMapping, *ce.CustomError) {
	var filterPort int
	var filterProto string
	var cerr *ce.CustomError

	if filter != "" {
		if filterPort, filterProto, cerr = parsePortSpec(filter); cerr != nil {
			return nil, cerr
		}
	}

	data, _, cerr := inspectContainerData(client, container)
	if cerr != nil {
		return nil, cerr
	}

	mappings := []PortMapping{}
	if data.NetworkSettings != nil {
		for spec, bindings := range data.NetworkSettings.Ports {
			port, proto, err := parsePortSpec(spec)
			if err != nil {
				continue
			}
			if filter != "" && (port != filterPort || proto != filterProto) {
				continue
			}
			// Exposed but unpublished ports have no bindings; docker port skips them too
			for _, b := range bindings {
				mappings = append(mappings, PortMapping{ContainerPort: port, Protocol: proto, HostIP: b.HostIP, HostPort: b.HostPort})
			}
		}
	}
	sort.Slice(mappings, func(i, j int) bool {
		a, b := mappings[i], mappings[j]
		if a.ContainerPort != b.ContainerPort {
			return a.ContainerPort < b.ContainerPort
		}
		if a.Protocol != b.Protocol {
			return a.Protocol < b.Protocol
		}
		return a.HostIP < b.HostIP
	})

	if filter != "" && len(mappings) == 0 {
		return nil, &ce.CustomError{Fatality: ce.Warning, Title: "No such port mapping",
			Message: fmt.Sprintf("no public port '%d/%s' published for %s", filterPort, filterProto, container)}
	}

	if extras.OutputJSON {
		b, cerr := extras.MarshalJSON(mappings)
		if cerr != nil {
			return nil, cerr
		}
		if err := hfjson.Print(b); err != nil {
			return nil, &ce.CustomError{Title: "Unable to print JSON output", Message: err.Error()}
		}
		return mappings, nil
	}

	if rest.QuietOutput || len(mappings) == 0 {
		return mappings, nil
	}

	w := tabwriter.NewWriter(os.Stdout, 1, 4, 2, ' ', 0)
	fmt.Fprintln(w, "CONTAINER PORT\tHOST IP\tHOST PORT")
	for _, m := range mappings {
		fmt.Fprintf(w, "%d/%s\t%s\t%s\n", m.ContainerPort, m.Protocol, m.HostIP, m.HostPort)
	}
	w.Flush()
	return mappings, nil
}

// parsePortSpec splits "80", "80/tcp" or "53/udp" into its port number and protocol
func parsePortSpec(spec string) (int, string, *ce.CustomError) {
	portStr, proto, found := strings.Cut(spec, "/")
	if !found || proto == "" {
		proto = "tcp"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return 0, "", &ce.CustomError{Title: "Invalid port", Message: "'" + spec + "' is not a valid PRIVATE_PORT[/PROTO]"}
	}
	return port, strings.ToLower(proto), nil
}
