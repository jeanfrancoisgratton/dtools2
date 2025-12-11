// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/02 18:45
// Original filename: src/containers/remove.go

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
		cname, err := Name2ID(client, container)
		if err != nil {
			return err
		}

		isBL, err := blacklist.IsResourceBlackListed("containers", cname)
		if err != nil {
			return err
		}

		// FIXME: this is tangled.. we need to clean this up
		if isBL {
			if !rest.QuietOutput {
				fmt.Println(hftx.WarningSign("Container " + cname + "is blacklisted"))
			}
			if RemoveBlacklistedContainers {
				if !rest.QuietOutput {
					fmt.Println(hftx.InfoSign("Force removal flag is present, continuing"))
				}
			} else {
				if !rest.QuietOutput {
					fmt.Println(hftx.InfoSign("Removal flag is absent, skipping container"))
				}
				if err := remove(client, cname); err != nil {
					return err
				}
			}
		} else {
			if err := remove(client, cname); err != nil {
				return err
			}
		}
	}
	return nil
}

// The actual removal call
func remove(client *rest.Client, cname string) *ce.CustomError {
	var id string
	var cerr *ce.CustomError
	q := url.Values{}

	q.Set("force", strconv.FormatBool(KillRunningContainers))
	q.Set("v", strconv.FormatBool(RemoveUnamedVolumes))

	if id, cerr = Name2ID(client, cname); cerr != nil {
		return cerr
	}
	path := "/containers/" + id
	resp, err := client.Do(rest.Context, http.MethodDelete, path, url.Values{}, nil, nil)
	if err != nil {
		return &ce.CustomError{Title: "Unable to post DELETE", Message: err.Error(), Code: 201}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return &ce.CustomError{Title: "DELETE request returned an error", Message: "http requested returned " + resp.Status, Code: 201}
	}
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign("Container " + cname + hftx.Red(" REMOVED")))
	}
	return nil
}
