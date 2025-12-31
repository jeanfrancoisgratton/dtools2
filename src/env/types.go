// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/29 02:26
// Original filename: src/registry/types.go

package env

var RegConfigFile string
var RegEntryComment string
var RegEntryUsername string
var RegEntryPassword string

type RegistryEntry struct {
	RegistryName  string `json:"RegistryName,omitempty"`
	Comments      string `json:"Comments,omitempty"`
	Username      string `json:"Username,omitempty"`
	EncodedPasswd string `json:"EncodedPasswd,omitempty"`
}
