// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/10/18
// Original filename: src/volumes/backup_helpers.go

// Helpers shared by `volume backup` and `volume restore` (folded in from dvol)

package volumes

import (
	"bytes"
	"dtools2/containers"
	"dtools2/images"
	"dtools2/rest"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// normalizeArchiveName appends ".tar" when the archive name has no known suffix
func normalizeArchiveName(archivePath string) string {
	if !strings.HasSuffix(archivePath, "xz") && !strings.HasSuffix(archivePath, "gz") && !strings.HasSuffix(archivePath, "tar") {
		return archivePath + ".tar"
	}
	return archivePath
}

func isGzipArchive(archivePath string) bool {
	return strings.HasSuffix(archivePath, ".gz") || strings.HasSuffix(archivePath, ".tgz")
}

func isXzArchive(archivePath string) bool {
	return strings.HasSuffix(archivePath, ".xz") || strings.HasSuffix(archivePath, ".txz")
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func fail(title, message string) *ce.CustomError {
	return &ce.CustomError{Title: title, Message: message}
}

// showDiscardedError displays an error that is not returned to the caller, because a more
// relevant error (the original failure) is the one being returned
func showDiscardedError(err *ce.CustomError) {
	if err != nil {
		fmt.Println(err)
	}
}

// checkStatus returns nil if the response status is one of okCodes, otherwise an error
// carrying the daemon's message. The body is fully consumed in both cases.
func checkStatus(resp *http.Response, okCodes ...int) error {
	body, _ := io.ReadAll(resp.Body)
	if slices.Contains(okCodes, resp.StatusCode) {
		return nil
	}
	var m struct {
		Message string `json:"message"`
	}
	msg := strings.TrimSpace(string(body))
	if json.Unmarshal(body, &m) == nil && m.Message != "" {
		msg = m.Message
	}
	if msg == "" {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	return fmt.Errorf("HTTP %s: %s", resp.Status, msg)
}

func showTimeoutNotes() {
	if rest.FastFailTimeoutSeconds != 30 {
		fmt.Println(hftx.NoteSign(fmt.Sprintf("HTTP handshake (fast-fail) timeout set to %d seconds", rest.FastFailTimeoutSeconds)))
	}
	if rest.SessionTimeoutMinutes != 30 {
		fmt.Println(hftx.NoteSign(fmt.Sprintf("HTTP session timeout set to %d minutes", rest.SessionTimeoutMinutes)))
	}
}

func showAttachedContainers(attached []string) {
	if len(attached) == 0 {
		fmt.Println(hftx.InfoSign("No running containers were attached to the volume to be backed up"))
	} else {
		fmt.Println(hftx.InfoSign(fmt.Sprintf("%d running containers are attached to the volume. They will be restarted after the backup",
			len(attached))))
	}
}

// getContainersUsingVolume returns the IDs of RUNNING containers that have a volume mount matching volumeName
func getContainersUsingVolume(client *rest.Client, volumeName string) ([]string, *ce.CustomError) {
	title := fmt.Sprintf("Error listing the containers using the volume %s", volumeName)
	q := url.Values{}
	q.Set("all", "true")

	resp, err := client.Do(rest.Context, http.MethodGet, "/containers/json", q, nil, nil)
	if err != nil {
		return nil, fail(title, err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fail(title, checkStatus(resp, http.StatusOK).Error())
	}

	var cs []containers.ContainerSummary
	if err := json.NewDecoder(resp.Body).Decode(&cs); err != nil {
		return nil, fail("Error decoding containers list", err.Error())
	}

	result := []string{}
	for _, c := range cs {
		if c.State != "running" {
			continue
		}
		for _, m := range c.Mounts {
			if m.Type == "volume" && (m.Name == volumeName || m.Source == volumeName) {
				result = append(result, c.ID)
				break
			}
		}
	}
	return result, nil
}

// postContainerAction POSTs /containers/{id}/{action} (start, stop).
// 304 (already started/stopped) is treated as a success.
func postContainerAction(client *rest.Client, id, action string) error {
	resp, err := client.Do(rest.Context, http.MethodPost, "/containers/"+id+"/"+action, nil, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return checkStatus(resp, http.StatusNoContent, http.StatusNotModified)
}

// stopContainers stops the containers in ids, and returns those that were actually stopped,
// so that the caller can restart them should a later step fail
func stopContainers(client *rest.Client, ids []string) ([]string, *ce.CustomError) {
	var stopped []string
	for _, id := range ids {
		if err := postContainerAction(client, id, "stop"); err != nil {
			return stopped, fail(fmt.Sprintf("Error stopping container %s", shortID(id)), err.Error())
		}
		stopped = append(stopped, id)
	}
	return stopped, nil
}

func startContainers(client *rest.Client, ids []string) *ce.CustomError {
	for _, id := range ids {
		if err := postContainerAction(client, id, "start"); err != nil {
			return fail(fmt.Sprintf("Error starting container %s", shortID(id)), err.Error())
		}
	}
	return nil
}

// restartAfterFailure restarts the containers stopped before a failed operation.
// The original error is what matters to the caller, so a restart error is only displayed.
func restartAfterFailure(client *rest.Client, ids []string) {
	if len(ids) == 0 {
		return
	}
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign("Restarting the containers that were stopped before the failure"))
	}
	showDiscardedError(startContainers(client, ids))
}

// warnLeftStopped tells the user which containers were deliberately not restarted
// because the volume they use may be in an inconsistent state
func warnLeftStopped(volumeName string, ids []string) {
	if len(ids) == 0 {
		return
	}
	shortIDs := make([]string, len(ids))
	for i, id := range ids {
		shortIDs[i] = shortID(id)
	}
	fmt.Println(hftx.WarningSign(fmt.Sprintf("Volume %s might be incomplete; these containers were left stopped: %s",
		hftx.Yellow(volumeName), strings.Join(shortIDs, ", "))))
}

// ensureImageExists pulls the image if it is not already present on the daemon
func ensureImageExists(client *rest.Client, image string) *ce.CustomError {
	q := url.Values{}
	q.Set("all", "true")

	// A failed lookup is not fatal: we simply attempt the pull
	resp, err := client.Do(rest.Context, http.MethodGet, "/images/json", q, nil, nil)
	if err == nil {
		var imgs []struct {
			RepoTags []string `json:"RepoTags"`
		}
		derr := json.NewDecoder(resp.Body).Decode(&imgs)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK && derr == nil {
			for _, im := range imgs {
				if slices.Contains(im.RepoTags, image) {
					if !rest.QuietOutput {
						fmt.Println(hftx.InfoSign(fmt.Sprintf("Image %s is already present, no need to pull it", image)))
					}
					return nil
				}
			}
		}
	}

	// The image is not on the daemon, we need to pull it (progress is not displayed)
	if perr := images.PullRef(client, image, io.Discard); perr != nil {
		return fail("Error pulling the image", perr.Error())
	}
	return nil
}

// createTempContainer creates a container from image with volumeName bound to /data, running cmd
func createTempContainer(client *rest.Client, image, volumeName string, cmd []string) (string, *ce.CustomError) {
	payload := map[string]any{
		"Image": image,
		"Cmd":   cmd,
		"HostConfig": map[string]any{
			"Binds": []string{fmt.Sprintf("%s:/data", volumeName)},
		},
		"Tty": true,
	}
	data, merr := json.Marshal(payload)
	if merr != nil {
		return "", fail("Error building container create payload", merr.Error())
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	resp, err := client.Do(rest.Context, http.MethodPost, "/containers/create", nil, bytes.NewReader(data), headers)
	if err != nil {
		return "", fail("Error creating temp container", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fail("Error creating temp container", checkStatus(resp, http.StatusCreated).Error())
	}

	var res struct {
		ID string `json:"Id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil || res.ID == "" {
		return "", fail("Error reading container create reply", "empty ID or decode failure")
	}
	return res.ID, nil
}

// startTempContainer creates and starts the long-lived container the archive is read from/written to
func startTempContainer(client *rest.Client, image, volumeName string) (string, *ce.CustomError) {
	id, cerr := createTempContainer(client, image, volumeName, []string{"sh", "-lc", "sleep infinity"})
	if cerr != nil {
		return "", cerr
	}
	if err := postContainerAction(client, id, "start"); err != nil {
		cleanupTempContainer(client, id)
		return "", fail("Error starting the temp container", err.Error())
	}
	return id, nil
}

func stopAndRemoveContainer(client *rest.Client, id string) *ce.CustomError {
	if err := postContainerAction(client, id, "stop"); err != nil {
		return fail(fmt.Sprintf("Container %s could not be stopped", hftx.Blue(shortID(id))), err.Error())
	}
	return removeContainer(client, id)
}

func removeContainer(client *rest.Client, id string) *ce.CustomError {
	resp, err := client.Do(rest.Context, http.MethodDelete, "/containers/"+id, nil, nil, nil)
	if err != nil {
		return fail(fmt.Sprintf("Container %s could not be removed", hftx.Blue(shortID(id))), err.Error())
	}
	defer resp.Body.Close()
	if serr := checkStatus(resp, http.StatusNoContent); serr != nil {
		return fail(fmt.Sprintf("Container %s could not be removed", hftx.Blue(shortID(id))), serr.Error())
	}
	return nil
}

// cleanupTempContainer removes the temp container after a failure (unless --no-cleanup);
// errors are only displayed, as the original failure is the one returned to the caller
func cleanupTempContainer(client *rest.Client, id string) {
	if NoCleanup || id == "" {
		return
	}
	showDiscardedError(stopAndRemoveContainer(client, id))
}

// resetVolume gives restore an empty volume to work with.
// The volume is deleted and re-created when possible. If it is still referenced by (stopped)
// containers, the daemon refuses the deletion (HTTP 409); its content is then wiped instead.
func resetVolume(client *rest.Client, volume, image string) *ce.CustomError {
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign(fmt.Sprintf("Deleting and recreating volume %s", hftx.Green(volume))))
	}

	resp, err := client.Do(rest.Context, http.MethodDelete, "/volumes/"+volume, nil, nil, nil)
	if err != nil {
		return fail("Error deleting the volume", err.Error())
	}
	status := resp.StatusCode
	serr := checkStatus(resp, http.StatusNoContent, http.StatusNotFound)
	resp.Body.Close()

	switch {
	case status == http.StatusConflict:
		if !rest.QuietOutput {
			fmt.Println(hftx.InfoSign(fmt.Sprintf("Volume %s is still referenced by containers, wiping its content instead", volume)))
		}
		return wipeVolume(client, volume, image)
	case serr != nil:
		return fail("Error deleting the volume", serr.Error())
	}

	buf, jerr := json.Marshal(map[string]string{"Name": volume})
	if jerr != nil {
		return fail("Error marshalling json", jerr.Error())
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	resp, err = client.Do(rest.Context, http.MethodPost, "/volumes/create", nil, bytes.NewReader(buf), headers)
	if err != nil {
		return fail(fmt.Sprintf("Error re-creating the volume %s", volume), err.Error())
	}
	defer resp.Body.Close()
	if serr := checkStatus(resp, http.StatusCreated); serr != nil {
		return fail(fmt.Sprintf("Error re-creating the volume %s", volume), serr.Error())
	}
	return nil
}

// wipeVolume removes everything inside the volume through a one-shot container
func wipeVolume(client *rest.Client, volume, image string) *ce.CustomError {
	title := fmt.Sprintf("Error wiping the volume %s", volume)

	id, cerr := createTempContainer(client, image, volume, []string{"sh", "-c", "find /data -mindepth 1 -delete"})
	if cerr != nil {
		return cerr
	}
	// The one-shot container is always removed: unlike the temp container, there is nothing to inspect in it
	defer func() { showDiscardedError(removeContainer(client, id)) }()

	if err := postContainerAction(client, id, "start"); err != nil {
		return fail(title, err.Error())
	}

	resp, err := client.Do(rest.Context, http.MethodPost, "/containers/"+id+"/wait", nil, nil, nil)
	if err != nil {
		return fail(title, err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fail(title, checkStatus(resp, http.StatusOK).Error())
	}

	var res struct {
		StatusCode int `json:"StatusCode"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return fail(title, err.Error())
	}
	if res.StatusCode != 0 {
		return fail(title, fmt.Sprintf("wipe container exited with code %d", res.StatusCode))
	}
	return nil
}
