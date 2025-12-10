// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/10 08:31
// Original filename: src/images/tag.go

package images

import (
	"dtools2/rest"
	"fmt"
	"net/http"
	"net/url"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// Retag an image

func TagImage(client *rest.Client, oldtag, newtag string) *ce.CustomError {
	repo, tag := splitURI(newtag)
	path := "/images/" + oldtag + "/tag?repo=" + repo + "&tag=" + tag

	resp, err := client.Do(rest.Context, http.MethodPost, path, url.Values{}, nil, nil)
	if err != nil {
		return &ce.CustomError{Title: "Unable to POST request", Message: err.Error(), Code: 201}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return &ce.CustomError{Title: "POST request returned an error", Message: "http requested returned " + resp.Status, Code: 201}
	}
	if !rest.QuietOutput {
		fmt.Println("Image " + hftx.Green(oldtag) + " tagged as " + hftx.Green(newtag))
	}
	return nil
}
