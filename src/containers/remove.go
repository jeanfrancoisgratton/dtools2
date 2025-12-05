// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/02 18:45
// Original filename: src/containers/remove.go

package containers

import (
	"dtools2/blacklist"
	"dtools2/rest"
	"fmt"

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
			if ForceRemoval {
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

	//resp, err := client.Do(rest.Context, http.MethodDelete, "/containers/"+cname+"/json", q, nil, nil)
	//if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
	//	return nil,
	//		&customError.CustomError{Title: "Unable to decode JSON", Message: err.Error(), Code: 201}
	//}
	return nil
}
