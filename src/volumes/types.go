// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/31 13:18
// Original filename: src/volumes/types.go

package volumes

var RemoveEvenIfBlackListed = false
var ForceRemoval = false

type VolumeListResponse struct {
	Volumes  []Volume `json:"Volumes"`
	Warnings []string `json:"Warnings"`
}

type VolumeCreateOptions struct {
	Name       string            `json:"Name,omitempty"`
	Driver     string            `json:"Driver,omitempty"`
	DriverOpts map[string]string `json:"DriverOpts,omitempty"`
	Labels     map[string]string `json:"Labels,omitempty"`
}

type Volume struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Mountpoint string            `json:"Mountpoint"`
	CreatedAt  string            `json:"CreatedAt,omitempty"`
	Status     map[string]string `json:"Status,omitempty"`
	Labels     map[string]string `json:"Labels"`
	Scope      string            `json:"Scope"`
	Options    map[string]string `json:"Options"`
	UsageData  *VolumeUsageData  `json:"UsageData,omitempty"`
}

type VolumeUsageData struct {
	Size     int64 `json:"Size"`
	RefCount int64 `json:"RefCount"`
}

type VolumePruneResponse struct {
	VolumesDeleted []string `json:"VolumesDeleted"`
	SpaceReclaimed int64    `json:"SpaceReclaimed"`
}
