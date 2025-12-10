// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/10 08:14
// Original filename: src/containers/rename.go

package containers

import (
	"dtools2/rest"
	"fmt"
	"net/http"
	"net/url"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// Rename a container
func RenameContainer(client *rest.Client, oldname, newname string) *ce.CustomError {
	var cerr *ce.CustomError
	var id string
	q := url.Values{}

	if id, cerr = Name2ID(client, oldname); cerr != nil {
		return cerr
	}
	path := "/containers/" + id + "/rename?name=" + newname
	resp, err := client.Do(rest.Context, http.MethodPost, path, q, nil, nil)
	if err != nil {
		return &ce.CustomError{Title: "Unable to POST request", Message: err.Error(), Code: 201}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict {
		return &ce.CustomError{Title: "Unable to rename the container", Message: "Container already exists", Code: 201}
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return &ce.CustomError{Title: "POST request returned an error", Message: "http requested returned " + resp.Status, Code: 201}
	}
	if !rest.QuietOutput {
		fmt.Println("Container " + hftx.Green(oldname) + " renamed to " + hftx.Green(newname))
	}
	return nil
}
