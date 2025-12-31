// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/29 02:26
// Original filename: src/registry/types.go

package registry

import (
	"os"
	"path/filepath"
)

var RegConfigFile = filepath.Join(os.Getenv("HOME"), ".config", "dtools2", "defaultRegistry.json")
var RegEntryComment = ""
var RegEntryUsername = ""
var RegEntryPassword = ""

type RegistryEntry struct {
	RegistryName  string `json:"RegistryName,omitempty"`
	Comments      string `json:"Comments,omitempty"`
	Username      string `json:"Username,omitempty"`
	EncodedPasswd string `json:"EncodedPasswd,omitempty"`
}
