// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/26 01:42
// Original filename: src/blacklist/list.go

package blacklist

import ce "github.com/jeanfrancoisgratton/customError/v3"

// ListAll returns all resources grouped by type.
// All resources are mapped in a key:value map for easier retrieval
func (rb *ResourceBlacklist) ListAll() map[string][]string {
	return map[string][]string{
		"volumes":    append([]string(nil), rb.Volumes...),
		"networks":   append([]string(nil), rb.Networks...),
		"images":     append([]string(nil), rb.Images...),
		"containers": append([]string(nil), rb.Containers...),
	}
}

// List returns all resources of a given type.
func (rb *ResourceBlacklist) List(resourceType string) ([]string, *ce.CustomError) {
	slicePtr, err := getSlice(rb, resourceType)
	if err != nil {
		return nil, err
	}
	// Return a copy to avoid external mutation.
	return append([]string(nil), (slicePtr)...), nil
}

// ListAllFromFile loads the blacklist and returns all entries.
func ListAllFromFile() *ce.CustomError {
	rb, err := Load()
	if err != nil {
		return err
	}
	//a:=rb.ListAll()
	return outputBList(rb.ListAll())
}

// ListFromFile lists entries for a given resource type.
// resourceType examples: "volumes", "networks", "images", "containers".
func ListFromFile(resourceType string) *ce.CustomError {
	//var rsources []string
	//var rbErr error
	rmp := make(map[string][]string)

	rb, err := Load()
	if err != nil {
		return err
	}
	if rsources, rbErr := rb.List(resourceType); rbErr != nil || len(rsources) == 0 {
		return rbErr
	} else {
		rmp = map[string][]string{
			resourceType: append([]string(nil), rsources...)}
	}
	return outputBList(rmp)
}
