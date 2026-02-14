// networks/types_inspect.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/02/13

package networks

// NetworkInspect models the payload returned by:
//
//	GET /networks/{id}
//
// Docker/Podman may return additional fields depending on driver/version; this
// structure keeps the common ones and stays permissive for the rest.
type NetworkInspect struct {
	Name       string `json:"Name"`
	ID         string `json:"Id"`
	Created    string `json:"Created,omitempty"`
	Scope      string `json:"Scope,omitempty"`
	Driver     string `json:"Driver,omitempty"`
	EnableIPv6 bool   `json:"EnableIPv6,omitempty"`
	Internal   bool   `json:"Internal,omitempty"`
	Attachable bool   `json:"Attachable,omitempty"`
	Ingress    bool   `json:"Ingress,omitempty"`

	IPAM    IPAMSummary       `json:"IPAM,omitempty"`
	Options map[string]string `json:"Options,omitempty"`
	Labels  map[string]string `json:"Labels,omitempty"`

	// Containers is a map keyed by container ID.
	Containers map[string]EndpointResource `json:"Containers,omitempty"`

	// Podman may return these; Docker sometimes does too.
	ConfigOnly bool `json:"ConfigOnly,omitempty"`
	ConfigFrom *struct {
		Network string `json:"Network,omitempty"`
	} `json:"ConfigFrom,omitempty"`
}

// EndpointResource models entries in NetworkInspect.Containers.
type EndpointResource struct {
	Name        string `json:"Name,omitempty"`
	EndpointID  string `json:"EndpointID,omitempty"`
	MacAddress  string `json:"MacAddress,omitempty"`
	IPv4Address string `json:"IPv4Address,omitempty"`
	IPv6Address string `json:"IPv6Address,omitempty"`
}
