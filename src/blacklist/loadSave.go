// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/26 01:29
// Original filename: src/blacklist/loadSave.go

package blacklist

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// Load loads the blacklist file. If the file does not exist or is empty,
// it returns an empty ResourceBlacklist without error.
func Load() (*ResourceBlacklist, *ce.CustomError) {
	path := filepath.Join(os.Getenv("HOME"), ".config", "JFG", "dtools2", BlacklistFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// No file yet: return an empty struct
			return &ResourceBlacklist{}, nil
		}
		return nil, &ce.CustomError{Title: "cannot read blacklist file " + path, Message: err.Error(), Code: 101}
	}

	if len(bytes.TrimSpace(data)) == 0 {
		return &ResourceBlacklist{}, nil
	}

	var rb ResourceBlacklist
	if err := json.Unmarshal(data, &rb); err != nil {
		return nil, &ce.CustomError{Title: "cannot parse blacklist JSON " + path, Message: err.Error(), Code: 101}
	}

	return &rb, nil
}

// Save writes the blacklist to disk, creating directories and file as needed.
func (rb *ResourceBlacklist) Save() *ce.CustomError {
	path := filepath.Join(os.Getenv("HOME"), ".config", "JFG", "dtools2", BlacklistFile)

	data, err := json.MarshalIndent(rb, "", "  ")
	if err != nil {
		return &ce.CustomError{Title: "cannot marshal blacklist struct", Message: err.Error(), Code: 101}
	}

	// 0600 so only the user can read/write
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return &ce.CustomError{Title: "cannot write blacklist file " + path, Message: err.Error(), Code: 101}
	}

	return nil
}
