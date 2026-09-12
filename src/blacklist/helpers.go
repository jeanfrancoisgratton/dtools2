// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/26 01:41
// Original filename: src/blacklist/helpers.go

package blacklist

import (
	"encoding/json"
	"slices"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfjson "github.com/jeanfrancoisgratton/helperFunctions/v5/prettyjson"
)

// getSlice returns a pointer to the slice corresponding to the resource type.
// Accepts case-insensitive names and both singular/plural: volume(s), network(s), image(s), container(s).
func getSlice(rb *ResourceBlacklist, resourceType string) (*[]string, *ce.CustomError) {
	t := strings.ToLower(resourceType)

	switch t {
	case "volume", "volumes":
		return &rb.Volumes, nil
	case "network", "networks":
		return &rb.Networks, nil
	case "image", "images":
		return &rb.Images, nil
	case "container", "containers":
		return &rb.Containers, nil
	default:
		return nil, &ce.CustomError{Title: "unknown resource type : " + resourceType}
	}
}

// outputBList displays the resource blacklist as pretty-printed JSON.

func outputBList(rbl map[string][]string) *ce.CustomError {
	b, err := json.MarshalIndent(rbl, "", "  ")
	if err != nil {
		return &ce.CustomError{Title: "Unable to marshal JSON", Message: err.Error()}
	}
	hfjson.Print(b)
	return nil
}

// Some commands (container rm, image rm, etc.) might prevent resources to be removed
// We check here for that

func IsResourceBlackListed(resourceType, resourceName string) (bool, *ce.CustomError) {
	rb, err := Load()
	if err != nil {
		return false, err
	}

	slicePtr, err := getSlice(rb, resourceType)
	if err != nil {
		return false, err
	}

	resources := *slicePtr
	return slices.Contains(resources, strings.ToLower(resourceName)), nil
}
