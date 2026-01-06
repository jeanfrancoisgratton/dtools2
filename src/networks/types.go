// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/14 20:15
// Original filename: src/networks/types.go

package networks

var NetworkDriverName = "bridge"
var NetworkInternalUse = false
var NetworkAttachable = false
var NetworkEnableIPv6 = false
var RemoveBlacklisted = false
var ForceNetworkDetach = false

type IPAMConfig struct {
	Subnet     string            `json:"Subnet,omitempty"`
	IPRange    string            `json:"IPRange,omitempty"`
	Gateway    string            `json:"Gateway,omitempty"`
	AuxAddress map[string]string `json:"AuxAddress,omitempty"`
}

type IPAMSummary struct {
	Driver string       `json:"Driver"`
	Config []IPAMConfig `json:"Config"`
}

type NetworkSummary struct {
	Name       string            `json:"Name"`
	ID         string            `json:"Id"`
	Created    string            `json:"Created"`
	Scope      string            `json:"Scope"`
	Driver     string            `json:"Driver,omitempty"`
	EnableIPv6 bool              `json:"EnableIPv6,omitempty"`
	Internal   bool              `json:"Internal,omitempty"`
	Attachable bool              `json:"Attachable,omitempty"`
	Ingress    bool              `json:"Ingress,omitempty"`
	IPAM       IPAMSummary       `json:"IPAM,omitempty"`
	Options    map[string]string `json:"Options,omitempty"`
	Labels     map[string]string `json:"Labels,omitempty"`
	InUse      bool              `json:"InUse,omitempty"`
}

type NetworkCreateRequest struct {
	Name           string            `json:"Name"`
	CheckDuplicate bool              `json:"CheckDuplicate,omitempty"`
	Driver         string            `json:"Driver,omitempty"` // default "bridge"
	Internal       bool              `json:"Internal,omitempty"`
	Attachable     bool              `json:"Attachable,omitempty"`
	Ingress        bool              `json:"Ingress,omitempty"`
	IPAM           *IPAMSummary      `json:"IPAM,omitempty"`
	EnableIPv6     bool              `json:"EnableIPv6,omitempty"`
	Options        map[string]string `json:"Options,omitempty"`
	Labels         map[string]string `json:"Labels,omitempty"`
}

type NetworkCreateResponse struct {
	Id      string `json:"Id"`
	Warning string `json:"Warning,omitempty"`
}

type NetworkConnectRequest struct {
	Container      string            `json:"Container"`
	EndpointConfig *EndpointSettings `json:"EndpointConfig,omitempty"`
}

type NetworkDisconnectRequest struct {
	Container string `json:"Container"`
	Force     bool   `json:"Force,omitempty"`
}

type EndpointSettings struct {
	IPAMConfig *IPAMConfig `json:"IPAMConfig,omitempty"`
	Links      []string    `json:"Links,omitempty"`
	Aliases    []string    `json:"Aliases,omitempty"`
	// Keep the rest optional; Docker may return more fields than you send.
	MacAddress string            `json:"MacAddress,omitempty"`
	DriverOpts map[string]string `json:"DriverOpts,omitempty"`
}
