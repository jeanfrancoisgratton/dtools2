// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/04 02:46
// Original filename: src/volumes/create.go

package volumes

import (
	"bytes"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

func CreateVolume(client *rest.Client, volumeName string) *ce.CustomError {
	vco := &VolumeCreateOptions{Name: volumeName, Driver: CreateVolDriver}

	payload, jerr := json.MarshalIndent(vco, "", "  ")
	if jerr != nil {
		return &ce.CustomError{Title: "Unable to marshal the JSON payload", Message: jerr.Error()}
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	resp, err := client.Do(rest.Context, http.MethodPost, "/volumes/create",
		url.Values{}, bytes.NewReader(payload), headers)
	if err != nil {
		return &ce.CustomError{Title: "Unable to create the volume", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return &ce.CustomError{Title: "HTTP request returned an error", Message: resp.Status}
	}

	if !rest.QuietOutput {
		fmt.Println(hftx.GreenGoSign("Volume " + hftx.Green(volumeName) + " created"))
	}
	return nil
}
