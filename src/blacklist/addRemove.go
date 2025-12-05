// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/26 01:44
// Original filename: src/blacklist/addRemove.go

package blacklist

import (
	"fmt"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// Add ensures that RESOURCENAME is present in the given resource type.
// It returns true if the blacklist was modified.
func (rb *ResourceBlacklist) Add(resourceType, name string) (bool, *ce.CustomError) {
	slice, err := getSlice(rb, resourceType)
	if err != nil {
		return false, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return false, &ce.CustomError{Title: "resource name cannot be empty", Code: 101}
	}

	//slice := *slicePtr
	for _, existing := range slice {
		if existing == name {
			// already present: "update" is effectively a no-op
			return false, nil
		}
	}

	slice = append(slice, name)
	//*slicePtr = slice
	return true, nil
}

// AddToFile adds a resource name to the given resource type and persists the file.
func AddToFile(resourceType, name string) *ce.CustomError {
	rb, err := Load()
	if err != nil {
		return err
	}

	changed, err := rb.Add(resourceType, name)
	if err != nil {
		return err
	}
	if !changed {
		fmt.Println(hftx.WarningSign(fmt.Sprintf(" resource name %s is already present in resources type %s",
			hftx.Yellow(name), hftx.Yellow(resourceType))))
		return nil
	}

	return rb.Save()
}

// Remove removes RESOURCENAME from the given resource type.
// It returns true if it was actually removed, false if it was not found.
func (rb *ResourceBlacklist) Remove(resourceType, name string) (bool, *ce.CustomError) {
	slice, err := getSlice(rb, resourceType)
	if err != nil {
		return false, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return false, &ce.CustomError{Title: "resource name cannot be empty", Code: 101}
	}

	//slice := *slicePtr
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
		slice = out
	}

	return removed, nil
}

// RemoveFromFile removes a resource name from the given resource type and persists the file.
// It returns (false, nil) if the resource was not in the blacklist (non-fatal).
func RemoveFromFile(resourceType, name string) (bool, *ce.CustomError) {
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

func AddResource(resourceType string, resourceName []string) *ce.CustomError {
	for _, rsc := range resourceName {
		if err := AddToFile(resourceType, rsc); err != nil {
			return err
		}
	}
	return nil
}

func DeleteResource(resourceType string, resourceName []string) (bool, *ce.CustomError) {
	var removed bool
	var err *ce.CustomError
	for _, rsc := range resourceName {
		if removed, err = RemoveFromFile(resourceType, rsc); err != nil {
			return removed, err
		}
	}
	return removed, nil
}
