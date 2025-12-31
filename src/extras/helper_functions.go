// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/14 14:37
// Original filename: src/images/helpers.go

package extras

import (
	"dtools2/registry"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// SplitURI takes a RepoTag entry (e.g. "registry:5000/repo/img:tag")
// and returns (imageName, tag).
// If no tag exists, tag = "latest".
func SplitURI(ref string) (string, string) {
	// Find last colon. Tags are ALWAYS after the last colon,
	// except cases like registry:5000 without a tag.
	idx := strings.LastIndex(ref, ":")
	if idx == -1 {
		// No colon → no explicit tag
		return ref, "latest"
	}

	// Check if this colon belongs to a registry port, not a tag.
	// That happens if ":" appears before the last "/" in the path.
	slash := strings.LastIndex(ref, "/")
	if slash != -1 && idx < slash {
		// Example: "registry:5000/repo/image" → no tag
		return ref, "latest"
	}

	// Split into name and tag
	return ref[:idx], ref[idx+1:]
}

// GetDefaultRegistry : fetches the default registry from the JSON file
// An error here should not be fatal

func GetDefaultRegistry(regfile string) (string, *ce.CustomError) {
	var err *ce.CustomError
	var dre *registry.RegistryEntry
	
	if dre, err = registry.Load(regfile); err != nil {
		return "", err
	}
	return dre.RegistryName, nil
}
