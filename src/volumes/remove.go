// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/01 15:11
// Original filename: src/volumes/remove.go

package volumes

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

func RemoveVolumes(client *rest.Client, volList []string) *ce.CustomError {
	for _, vol := range volList {
		isBL, err := blacklist.IsResourceBlackListed("volumes", vol)
		if err != nil {
			return err
		}
		if isBL {
			if !rest.QuietOutput {
				fmt.Println(hftx.WarningSign(" Volume " + vol + " is blacklisted"))
			}
			if RemoveBlackListed {
				if !rest.QuietOutput {
					fmt.Println(hftx.InfoSign("Force removal flag is present, continuing"))
					if err := removeVol(client, vol); err != nil {
						return err
					}
				}
			} else {
				if !rest.QuietOutput {
					fmt.Println(hftx.InfoSign("Removal flag is absent, skipping volume"))
				}
			}
		} else {
			if err := removeVol(client, vol); err != nil {
				return err
			}
		}
	}
	return nil
}

func removeVol(client *rest.Client, volumeName string) *ce.CustomError {
	q := url.Values{}
	q.Set("force", strconv.FormatBool(ForceRemove))

	path := "/volumes/" + volumeName
	resp, derr := client.Do(rest.Context, http.MethodDelete, path, q, nil, nil)
	if derr != nil {
		return &ce.CustomError{Title: "Unable to post DELETE", Message: derr.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return &ce.CustomError{Title: "Error removing the volume", Message: "http request returned " + resp.Status}
	}
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign("Volume " + volumeName + hftx.Red(" REMOVED")))
	}
	return nil
}
