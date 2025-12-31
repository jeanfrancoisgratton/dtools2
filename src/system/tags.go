// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/31 04:40
// Original filename: src/system/tags.go

package system

import (
	"context"
	"dtools2/env"
	"dtools2/extras"
	"dtools2/registry"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"os"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfjson "github.com/jeanfrancoisgratton/helperFunctions/v4/prettyjson"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

// GetTags : fetches all tags of a given image

func GetTags(repo string) *ce.CustomError {
	var dreg string
	var clt *registry.Client
	var err *ce.CustomError
	var returnedBytes []byte

	//var err *ce.CustomError

	if drn, err := extras.GetDefaultRegistry(env.RegConfigFile); err != nil {
		return err
	} else {
		dreg = drn
	}

	if clt, err = registry.NewClient(dreg); err != nil {
		return err
	}
	if returnedBytes, err = clt.TagsJSON(context.Background(), repo, nil); err != nil {
		return err
	}

	var payload TagsListResponse
	if err := json.Unmarshal(returnedBytes, &payload); err != nil {
		return &ce.CustomError{Title: "Error unmarshalling the JSON payload", Message: err.Error()}
	}
	if JSONoutputfile != "" {
		if !rest.QuietOutput {
			fmt.Println(hftx.EnabledSign("Output sent to " + JSONoutputfile))
		}
		jStream, jerr := json.MarshalIndent(payload, "", "  ")
		if jerr != nil {
			return &ce.CustomError{Title: "Error marshaling the JSON payload", Message: jerr.Error()}
		}
		if werr := os.WriteFile(JSONoutputfile, jStream, 0600); werr != nil {
			return &ce.CustomError{Title: "Error writing the JSON output file", Message: werr.Error()}
		}
		return nil
	}
	hfjson.Print(returnedBytes)
	return nil
}
