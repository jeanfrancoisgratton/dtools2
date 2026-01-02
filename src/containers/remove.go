// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/02 18:45
// Original filename: src/containers/add_remove_load.go

package containers

import (
	"dtools2/blacklist"
	"dtools2/rest"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// RemoveContainer Remove a single or multiple containers
// This can be wrapped in RemoveAll
func RemoveContainer(client *rest.Client, containerList []string) *ce.CustomError {
	for _, container := range containerList {
		id, err := Name2ID(client, container)
		if err != nil {
			return err
		}

		isBL, err := blacklist.IsResourceBlackListed("containers", container)
		if err != nil {
			return err
		}

		// FIXME: this is tangled.. we need to clean this up
		if isBL {
			if !rest.QuietOutput {
				fmt.Println(hftx.WarningSign(" Container " + container + " is blacklisted"))
			}
			if RemoveBlacklisted {
				if !rest.QuietOutput {
					fmt.Println(hftx.InfoSign("Force removal flag is present, continuing"))
				}
				if err := remove(client, container, id); err != nil {
					return err
				}
			} else {
				if !rest.QuietOutput {
					fmt.Println(hftx.InfoSign("Removal flag is absent, skipping container"))
				}
			}
		} else {
			if err := remove(client, container, id); err != nil {
				return err
			}
		}
	}
	return nil
}

// The actual removal call
func remove(client *rest.Client, name, id string) *ce.CustomError {
	q := url.Values{}

	q.Set("force", strconv.FormatBool(ForceRemoveContainer))
	q.Set("v", strconv.FormatBool(RemoveUnamedVolumes))

	path := "/containers/" + id
	resp, derr := client.Do(rest.Context, http.MethodDelete, path, q, nil, nil)
	if derr != nil {
		return &ce.CustomError{Title: "Unable to post DELETE", Message: derr.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return &ce.CustomError{Title: "DELETE request returned http 409 (Conflict)", Message: "Container " + name + " might be running"}
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return &ce.CustomError{Title: "DELETE request returned an error", Message: "http request returned " + resp.Status}
	}
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign("Container " + name + hftx.Red(" REMOVED")))
	}
	return nil
}
