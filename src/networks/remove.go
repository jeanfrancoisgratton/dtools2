package networks

import (
	"dtools2/blacklist"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
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
				}
				if err := removeNet(client, net); err != nil {
					return err
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
		if resp.StatusCode == http.StatusNotFound {
			return &ce.CustomError{Title: "Unable to post DELETE", Message: "The network " + networkName + " is not found"}
		}
		// The daemon's message tells the cases apart: 403 is returned both for built-in networks
		// and for networks that still have active endpoints (containers attached)
		msg := resp.Status
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		var der dockerErrorResponse
		if json.Unmarshal(bodyBytes, &der) == nil && der.Message != "" {
			msg = der.Message
		}
		return &ce.CustomError{Title: "Unable to remove network " + networkName + " (HTTP " + strconv.Itoa(resp.StatusCode) + ")", Message: msg}
	}
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign("Network " + networkName + hftx.Red(" REMOVED")))
	}
	return nil
}
