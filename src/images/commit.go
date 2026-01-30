// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/30
// Original filename: src/images/commit.go

package images

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"dtools2/rest"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
)

type commitResponse struct {
	ID string `json:"Id"`
}

// ImageCommit emulates `docker commit` using the daemon's /commit endpoint.
// Unlike docker, repository:tag is mandatory.
//
//   - author (docker commit -a) -> query param "author"
//   - message (docker commit -m) -> query param "comment"
//   - changes (docker commit -c) -> query param "changes" (repeatable)
func ImageCommit(client *rest.Client, containerRef, repoTag, author, message string, changes []string) *ce.CustomError {
	repo, tag := splitRepoTag(repoTag)
	if repo == "" || tag == "" {
		return &ce.CustomError{Title: "image commit error", Message: "repository:tag is required (got " + repoTag + ")"}
	}

	q := url.Values{}
	q.Set("container", containerRef)
	q.Set("repo", repo)
	q.Set("tag", tag)

	if author != "" {
		q.Set("author", author)
	}
	if message != "" {
		q.Set("comment", message)
	}
	for _, c := range changes {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		q.Add("changes", c)
	}

	resp, err := client.Do(rest.Context, http.MethodPost, "/commit", q, nil, nil)
	if err != nil {
		return &ce.CustomError{Title: "image commit error", Message: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		msg, _ := readSmallBody(resp.Body, 8192)
		if msg != "" {
			return &ce.CustomError{Title: "image commit failed", Message: "http response is " + resp.Status + " (" + msg + ")"}
		}
		return &ce.CustomError{Title: "image commit failed", Message: "http response is " + resp.Status}
	}

	if !rest.QuietOutput {
		fmt.Println(hftx.GreenGoSign("Commited container " + containerRef + " to image " + repoTag))
	}

	return nil
}
