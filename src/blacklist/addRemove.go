// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/26 01:44
// Original filename: src/blacklist/addRemove.go

package blacklist

import (
	"fmt"
	"strings"
)

// Add ensures that RESOURCENAME is present in the given resource type.
// It returns true if the blacklist was modified.
func (rb *ResourceBlacklist) Add(resourceType, name string) (bool, error) {
	slicePtr, err := getSlice(rb, resourceType)
	if err != nil {
		return false, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return false, fmt.Errorf("resource name cannot be empty")
	}

	slice := *slicePtr
	for _, existing := range slice {
		if existing == name {
			// already present: "update" is effectively a no-op
			return false, nil
		}
	}

	slice = append(slice, name)
	*slicePtr = slice
	return true, nil
}

// AddToFile adds a resource name to the given resource type and persists the file.
func AddToFile(resourceType, name string) error {
	rb, err := Load()
	if err != nil {
		return err
	}

	changed, err := rb.Add(resourceType, name)
	if err != nil {
		return err
	}
	if !changed {
		// no change needed; still considered success
		return nil
	}

	return rb.Save()
}

// Remove removes RESOURCENAME from the given resource type.
// It returns true if it was actually removed, false if it was not found.
func (rb *ResourceBlacklist) Remove(resourceType, name string) (bool, error) {
	slicePtr, err := getSlice(rb, resourceType)
	if err != nil {
		return false, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return false, fmt.Errorf("resource name cannot be empty")
	}

	slice := *slicePtr
	out := slice[:0]
	removed := false

	for _, existing := range slice {
		if existing == name {
			removed = true
			continue
		}
		out = append(out, existing)
	}

	if removed {
		*slicePtr = out
	}

	return removed, nil
}

// RemoveFromFile removes a resource name from the given resource type and persists the file.
// It returns (false, nil) if the resource was not in the blacklist (non-fatal).
func RemoveFromFile(resourceType, name string) (bool, error) {
	rb, err := Load()
	if err != nil {
		return false, err
	}

	removed, err := rb.Remove(resourceType, name)
	if err != nil {
		return false, err
	}

	if removed {
		if err := rb.Save(); err != nil {
			return false, err
		}
	}

	return removed, nil
}

func AddResource(resourceType string, resourceName []string) {
	for _, rsc := range resourceName {
		AddToFile(resourceType, rsc)
	}
}
