// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/30
// Original filename: src/images/commit.go

package images

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"dtools2/rest"
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
func ImageCommit(client *rest.Client, containerRef, repoTag, author, message string, changes []string) error {
	if containerRef == "" {
		return fmt.Errorf("container name or ID is required")
	}
	if repoTag == "" {
		return fmt.Errorf("repository:tag is required")
	}

	repo, tag := splitRepoTag(repoTag)
	if repo == "" || tag == "" {
		return fmt.Errorf("repository:tag is required (got %q)", repoTag)
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
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		msg, _ := readSmallBody(resp.Body, 8192)
		if msg != "" {
			return fmt.Errorf("image commit failed: %s: %s", resp.Status, msg)
		}
		return fmt.Errorf("image commit failed: %s", resp.Status)
	}

	var cr commitResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return fmt.Errorf("unable to decode commit response: %w", err)
	}
	if cr.ID == "" {
		return fmt.Errorf("commit returned success, but response did not include an image id")
	}

	if !rest.QuietOutput {
		fmt.Fprintln(os.Stdout, cr.ID)
	}

	return nil
}
