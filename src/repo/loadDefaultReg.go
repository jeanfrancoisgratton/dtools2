// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/24 06:24
// Original filename: src/repo/loadDefaultReg.go

package repo

import (
	"encoding/json"
	"os"
	"path/filepath"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

type DefaultRegistryStruct struct {
	RegistryURL      string `json:"RegistryURL"`
	RegistryUsername string `json:"RegistryUsername,omitempty"`
	Comments         string `json:"Comments,omitempty"`
}

// This loads the default registry url and username (if present) to be used as the default registry in various calls
func LoadDefaultRegistry() (DefaultRegistryStruct, *ce.CustomError) {
	var payload DefaultRegistryStruct

	jsonfile, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".config", "JFG", "dtools2", "defaultRegistry.json"))
	if err != nil {
		return DefaultRegistryStruct{}, &ce.CustomError{Title: "Unable to read default registry file", Message: err.Error()}
	}

	err = json.Unmarshal(jsonfile, &payload)
	if err != nil {
		return DefaultRegistryStruct{}, &ce.CustomError{Title: "Unable to unmarshal data", Message: err.Error()}
	}
	return payload, nil
}
