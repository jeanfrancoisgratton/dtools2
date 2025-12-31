// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/29 02:26
// Original filename: src/registry/add_remove.go

package registry

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// RemoveReg : a simple matter of removing a file, and then create a new one with
// an empty JSON struct
func (re RegistryEntry) RemoveReg() *ce.CustomError {
	err := os.Remove(RegConfigFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// re is empty, so the following line creates an empty JSON file (that is, one with no values)
			return re.AddReg()
		}
		return &ce.CustomError{Title: "Error removing the default registry file", Message: err.Error()}
	}
	return nil
}

// AddReg : Create a new JSON file, overwriting the previous one
func (re RegistryEntry) AddReg() *ce.CustomError {
	payload, jerr := json.MarshalIndent(re, "", "  ")
	if jerr != nil {
		return &ce.CustomError{Title: "Unable to marshal the JSON payload", Message: jerr.Error()}
	}

	if err := os.WriteFile(RegConfigFile, payload, 0600); err != nil {
		return &ce.CustomError{Title: "Error adding the default registry file", Message: err.Error()}
	}
	return nil
}

// Load : Loads the default registry JSON file
func Load(regfile string) (*RegistryEntry, *ce.CustomError) {
	var re *RegistryEntry = nil

	if regfile == "" {
		regfile = RegConfigFile
	}
	p := strings.ToLower(regfile)
	if !strings.HasSuffix(p, ".json") {
		regfile += ".json"
	}
	jFile, err := os.ReadFile(regfile)
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to read the registry file", Message: err.Error()}
	}
	err = json.Unmarshal(jFile, &re)
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to unmarshal the JSON payload", Message: err.Error()}
	} else {
		return re, nil
	}
}
