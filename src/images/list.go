// images/list.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/08 16:08
// Original filename: src/images/list.go

package images

import (
	"dtools2/extras"
	"dtools2/rest"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"time"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hfjson "github.com/jeanfrancoisgratton/helperFunctions/v4/prettyjson"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func ImagesList(client *rest.Client, displayOutput bool) ([]ImageSummary, *ce.CustomError) {
	var iInfoSlice []ImageSummary

	// Create & execute the http request
	resp, err := client.Do(rest.Context, http.MethodGet, "/images/json", url.Values{}, nil, nil)
	if err != nil {
		return nil, &ce.CustomError{Title: "Unable to list images", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, &ce.CustomError{Title: "http request returned an error", Message: "GET /images/json returned " + resp.Status}
	}

	// Decode JSON only if we actually have content
	var images []ImageSummary
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(&images); err != nil {
			return nil, &ce.CustomError{Title: "Unable to decode JSON", Message: err.Error()}
		}
	}

	// Now that we have the data in a JSON payload, we need to parse it
	// 1. Parse all images
	for _, img := range images {
		// 2. Parse all tags off an image if the daemon hosts multiple variants (tags) of a given image
		for _, tag := range img.RepoTags {
			var iInfo ImageSummary

			iInfo.RepoImgName, iInfo.ImgTag = extras.SplitURI(tag)
			// Drop "sha256:" prefix for display
			if len(img.ID) > 7 {
				iInfo.ID = img.ID[7:]
			} else {
				iInfo.ID = img.ID
			}
			iInfo.Created = img.Created
			iInfo.Size = img.Size
			iInfo.Containers = img.Containers

			iInfoSlice = append(iInfoSlice, iInfo)
		}
	}

	// If we're not supposed to display anything, return the parsed list.
	if !displayOutput {
		return iInfoSlice, nil
	}

	// Optional: write JSON payload to a file and/or render JSON to stdout.
	// This is only done when displayOutput is true (i.e., list commands).
	var payloadBytes []byte
	if extras.OutputFile != "" {
		b, cerr := extras.Send2File(iInfoSlice, extras.OutputFile)
		if cerr != nil {
			return nil, cerr
		}
		payloadBytes = b
	}

	// --format takes precedence over --json, tables and ignores --quiet.
	if extras.OutputFormat != "" {
		rows, cerr := extras.ExtractFormatRows(iInfoSlice, extras.OutputFormat)
		if cerr != nil {
			return nil, cerr
		}
		if cerr := extras.PrintFormatRows(rows); cerr != nil {
			return nil, cerr
		}
		return iInfoSlice, nil
	}

	if extras.OutputJSON {
		// Marshal once if we didn't already (for --file).
		if payloadBytes == nil {
			b, cerr := extras.MarshalJSON(iInfoSlice)
			if cerr != nil {
				return nil, cerr
			}
			payloadBytes = b
		}

		hfjson.Print(payloadBytes)
		return iInfoSlice, nil
	}

	// JSON output not requested; if quiet, return data only.
	if rest.QuietOutput {
		return iInfoSlice, nil
	}

	// Build and render the table ONCE
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{
		"Repository/image name",
		"Image tag",
		"Image ID",
		"Creation time",
		"Size",
		"# containers",
	})

	// When there are no images, append a single empty row to keep the
	// table borders and layout intact.
	if len(iInfoSlice) == 0 {
		t.AppendRow(table.Row{"", "", "", "", "", ""})
	} else {
		for _, imgspec := range iInfoSlice {
			// imgspec.ID is already stripped of "sha256:" above; trim to 12 chars for display
			displayID := imgspec.ID
			if len(displayID) >= 12 {
				displayID = displayID[:12]
			}

			t.AppendRow(table.Row{
				imgspec.RepoImgName,
				imgspec.ImgTag,
				displayID,
				time.Unix(imgspec.Created, 0).Format("2006.01.02 15:04:05"),
				formatSize(imgspec.Size),
				imgspec.Containers,
			})
		}
	}

	t.SortBy([]table.SortBy{
		{Name: "Repository/image name", Mode: table.Asc},
	})
	t.SetStyle(table.StyleColoredBlackOnBlueWhite)
	t.Style().Format.Header = text.FormatDefault
	t.SetRowPainter(func(row table.Row) text.Colors {
		switch row[5] {
		case 0:
			return text.Colors{text.FgBlack}
		default:
			//return text.Colors{text.FgHiGreen}
			return text.Colors{text.FgHiGreen, text.BgWhite}
		}
	})

	t.Render()
	return iInfoSlice, nil
}
