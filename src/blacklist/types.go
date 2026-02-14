// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/26 01:15
// Original filename: src/blacklist/types.go

package blacklist

var BlacklistFile = "blacklist.json"
var AllBlackLists = false
var ResourceNamesList = []string{"volume", "volumes", "network", "networks", "image", "images", "container", "containers"}

// ResourceBlacklist : this is a list of all resources that are blacklisted
// A blacklisted resource is a resource that cannot be removed by bulk commands
// Such as `dtools2 container rmall`, `dtools2 image rmiall`, etc.
// If a resource is blacklisted, it will get protected from such bulk command.
type ResourceBlacklist struct {
	Volumes    []string `json:"Volumes,omitempty"`
	Networks   []string `json:"Networks,omitempty"`
	Images     []string `json:"Images,omitempty"`
	Containers []string `json:"Containers,omitempty"`
}
