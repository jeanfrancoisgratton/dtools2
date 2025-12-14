// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/26 01:44
// Original filename: src/blacklist/addRemove.go

package blacklist

import (
	"dtools2/extras"
	"dtools2/rest"
	"fmt"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

func (rb *ResourceBlacklist) Add(resourceType, name string) (bool, *ce.CustomError) {
	slicePtr, err := getSlice(rb, resourceType)
	if err != nil {
		return false, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return false, &ce.CustomError{Title: "resource name cannot be empty"}
	}
	if resourceType == "image" {
		r, t := extras.SplitURI(name)
		name = r + ":" + t
	}
	slice := *slicePtr

	for _, existing := range slice {
		if existing == name {
			return false, nil
		}
	}

	slice = append(slice, name)
	*slicePtr = slice

	if !rest.QuietOutput {
		fmt.Println(hftx.NoteSign("Resource " + name + " now blacklisted from " + resourceType))
	}
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

func AddResource(resourceType string, resourceName []string) *ce.CustomError {
	for _, rsc := range resourceName {
		if err := AddToFile(resourceType, rsc); err != nil {
			return err
		}
	}
	return nil
}

func (rb *ResourceBlacklist) Remove(resourceType, name string) (bool, *ce.CustomError) {
	slicePtr, err := getSlice(rb, resourceType)
	if err != nil {
		return false, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return false, &ce.CustomError{Title: "resource name cannot be empty"}
	}

	if resourceType == "image" {
		r, t := extras.SplitURI(name)
		name = r + ":" + t
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

	if !rest.QuietOutput {
		fmt.Println(hftx.NoteSign("Resource " + name + " removed from the " + resourceType + " list"))
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
