// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/24 20:29
// Original filename: src/containers/types.go

package containers

var OnlyRunningContainers bool
var ExtendedContainerInfo bool
var DisplaySizeValues bool = false
var RemoveBlacklisted bool = false
var RemoveUnamedVolumes = true
var KillSwitch = false
var ForceRemoveContainer = false

// StopTimeout controls stop behaviour:
//
//	>0 => sequential stop, value passed as Docker/Podman `t` parameter
//	 0 => concurrent stop, internal default timeout used per container
var StopTimeout int = 10

type PortsStruct struct {
	PrivatePort uint16 `json:"PrivatePort"`
	PublicPort  uint16 `json:"PublicPort"`
	Type        string `json:"Type"`
	IP          string `json:"IP"`
}

type MountsStruct struct {
	Type        string `json:"Type"`
	Name        string `json:"Name"`
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	RW          bool   `json:"RW"`
}

type ContainerSummary struct {
	ID              string                    `json:"Id"`
	Names           []string                  `json:"Names"`
	Image           string                    `json:"Image"`
	ImageID         string                    `json:"ImageID"`
	Command         string                    `json:"Command"`
	Created         int64                     `json:"Created"`
	Ports           []PortsStruct             `json:"Ports"`
	SizeRw          int64                     `json:"SizeRw,omitempty"`
	SizeRootFs      int64                     `json:"SizeRootFs,omitempty"`
	Labels          map[string]string         `json:"Labels"`
	State           string                    `json:"State"`
	Status          string                    `json:"Status"`
	Mounts          []MountsStruct            `json:"Mounts"`
	NetworkSettings *ContainerNetworkSettings `json:"NetworkSettings,omitempty"`
	//Networks        map[string]EndpointSummary `json:"Networks,omitempty"`
}

type ContainerNetworkSettings struct {
	Networks map[string]EndpointSummary `json:"Networks,omitempty"` // key = network name
}

type EndpointSummary struct {
	NetworkID           string `json:"NetworkID,omitempty"`
	EndpointID          string `json:"EndpointID,omitempty"`
	Gateway             string `json:"Gateway,omitempty"`
	IPAddress           string `json:"IPAddress,omitempty"`
	IPPrefixLen         int    `json:"IPPrefixLen,omitempty"`
	IPv6Gateway         string `json:"IPv6Gateway,omitempty"`
	GlobalIPv6Address   string `json:"GlobalIPv6Address,omitempty"`
	GlobalIPv6PrefixLen int    `json:"GlobalIPv6PrefixLen,omitempty"`
	MacAddress          string `json:"MacAddress,omitempty"`
}
