// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/11 13:01
// Original filename: src/images/add_remove.go

package images

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

func RemoveImage(client *rest.Client, imglist []string) *ce.CustomError {
	for _, img := range imglist {
		isBL, err := blacklist.IsResourceBlackListed("images", img)
		if err != nil {
			return err
		}
		if isBL {
			if !rest.QuietOutput {
				fmt.Println(hftx.WarningSign(" Image " + img + " is blacklisted"))
			}
			if RemoveBlacklisted {
				if !rest.QuietOutput {
					fmt.Println(hftx.InfoSign("Force removal flag is present, continuing"))
					if err := remove(client, img); err != nil {
						return err
					}
				}
			} else {
				if !rest.QuietOutput {
					fmt.Println(hftx.InfoSign("Removal flag is absent, skipping container"))
				}
			}
		} else {
			if err := remove(client, img); err != nil {
				return err
			}
		}
	}
	return nil
}

// The actual removal function

func remove(client *rest.Client, imagename string) *ce.CustomError {
	q := url.Values{}
	q.Set("force", strconv.FormatBool(ForceRemove))

	path := "/images/" + imagename
	resp, derr := client.Do(rest.Context, http.MethodDelete, path, q, nil, nil)
	if derr != nil {
		return &ce.CustomError{Title: "Unable to post DELETE", Message: derr.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return &ce.CustomError{Title: "DELETE request returned an error", Message: "http requested returned " + resp.Status}
	}
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign("Image " + imagename + hftx.Red(" REMOVED")))
	}
	return nil
}
