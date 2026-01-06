package networks

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

func RemoveNetwork(client *rest.Client, netList []string) *ce.CustomError {
	for _, net := range netList {
		isBL, err := blacklist.IsResourceBlackListed("networks", net)
		if err != nil {
			return err
		}
		if isBL {
			if !rest.QuietOutput {
				fmt.Println(hftx.WarningSign(" Network " + net + " is blacklisted"))
			}
			if RemoveBlacklisted {
				if !rest.QuietOutput {
					fmt.Println(hftx.InfoSign("Force removal flag is present, continuing"))
					if err := removeNet(client, net); err != nil {
						return err
					}
				}
			} else {
				if !rest.QuietOutput {
					fmt.Println(hftx.InfoSign("Removal flag is absent, skipping network"))
				}
			}
		} else {
			if err := removeNet(client, net); err != nil {
				return err
			}
		}
	}
	return nil
}

func removeNet(client *rest.Client, networkName string) *ce.CustomError {
	var id string
	var err *ce.CustomError

	q := url.Values{}

	if id, err = Name2ID(client, networkName); err != nil {
		return err
	}
	path := "/networks/" + id
	resp, derr := client.Do(rest.Context, http.MethodDelete, path, q, nil, nil)
	if derr != nil {
		return &ce.CustomError{Title: "Unable to post DELETE", Message: derr.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		title := "Unable to post DELETE"
		msg := ""
		switch resp.StatusCode {
		case http.StatusForbidden:
			msg = "The network " + networkName + " is not allowed to be removed (built-in)"
		case http.StatusNotFound:
			msg = "The network " + networkName + " is not found"
		default:
			title = "DELETE request http request failed with status code " + strconv.Itoa(resp.StatusCode)
			msg = derr.Error()
		}
		return &ce.CustomError{Title: title, Message: msg}
	}
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign("Network " + networkName + hftx.Red(" REMOVED")))
	}
	return nil
}
