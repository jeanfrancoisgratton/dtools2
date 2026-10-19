// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/10/18
// Original filename: src/volumes/restore.go

package volumes

import (
	"compress/gzip"
	"dtools2/blacklist"
	"dtools2/rest"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/ulikunitz/xz"
)

// RestoreVolume uploads a tar(.gz|.xz|.txz) archive to /containers/{id}/archive (upload).
// The archive was created by BackupVolume with ?path=/data, so entries are rooted at "data/..."
// (e.g. "data/pg_data/..."). We PUT to ?path=/ so "data/" extracts to /data/ inside the volume,
// matching the original mount point exactly.
//
// Failure handling: until the volume is reset, nothing has been altered and the stopped
// containers are restarted. From then on, the volume might be empty or partially restored, so
// the containers are left stopped rather than restarted on top of inconsistent data.
func RestoreVolume(client *rest.Client, volumeName, archivePath string) *ce.CustomError {
	archivePath = normalizeArchiveName(archivePath)

	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign(fmt.Sprintf("Restoring %s from %s", hftx.Blue(volumeName), hftx.Blue(archivePath))))
		showTimeoutNotes()
	}

	// Blacklisting guards against bulk volume removal; a restore is not a data removal operation,
	// so a blacklisted volume only yields a warning
	if !rest.QuietOutput {
		if isBL, blerr := blacklist.IsResourceBlackListed("volumes", volumeName); blerr != nil {
			fmt.Println(hftx.WarningSign("Unable to check the volume blacklist: " + blerr.Error()))
		} else if isBL {
			fmt.Println(hftx.WarningSign(fmt.Sprintf("Volume %s is blacklisted; it is restored anyway", hftx.Yellow(volumeName))))
		}
	}

	// Fail early on an unreadable archive, before anything is stopped or deleted
	file, oerr := os.Open(archivePath)
	if oerr != nil {
		return fail("Unable to open archive", oerr.Error())
	}
	defer file.Close()

	if err := ensureImageExists(client, BackupImage); err != nil {
		return err
	}

	// Stop containers using the volume
	attachedContainers, err := getContainersUsingVolume(client, volumeName)
	if !rest.QuietOutput {
		showAttachedContainers(attachedContainers)
	}
	if err != nil {
		return err
	}
	if len(attachedContainers) > 0 {
		if !rest.QuietOutput {
			fmt.Println(hftx.InProgressSign(fmt.Sprintf("Stopping the %d container(s) attached to %s", len(attachedContainers), volumeName)))
		}
		if stopped, e := stopContainers(client, attachedContainers); e != nil {
			restartAfterFailure(client, stopped)
			return e
		}
	}

	if e := restoreFromArchive(client, volumeName, archivePath, file); e != nil {
		warnLeftStopped(volumeName, attachedContainers)
		return e
	}

	return startContainers(client, attachedContainers)
}

// restoreFromArchive resets the volume, then uploads the archive through the temp container
func restoreFromArchive(client *rest.Client, volumeName, archivePath string, file *os.File) *ce.CustomError {
	// Destroy & recreate the volume before restore
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign(fmt.Sprintf("Deleting the volume %s before creating a brand-new restored volume", volumeName)))
	}
	if e := resetVolume(client, volumeName, BackupImage); e != nil {
		return e
	}

	// Create & start temp container
	containerID, cerr := startTempContainer(client, BackupImage, volumeName)
	if cerr != nil {
		return cerr
	}

	if e := uploadArchive(client, containerID, archivePath, file); e != nil {
		cleanupTempContainer(client, containerID)
		return e
	}

	if !NoCleanup {
		if e := stopAndRemoveContainer(client, containerID); e != nil {
			return e
		}
	}
	return nil
}

func uploadArchive(client *rest.Client, containerID, archivePath string, file *os.File) *ce.CustomError {
	// If compressed, decompress to expose a raw tar stream to the PUT
	var body io.Reader = file
	switch {
	case isGzipArchive(archivePath):
		gr, gerr := gzip.NewReader(file)
		if gerr != nil {
			return fail("Unable to create the gzip reader", gerr.Error())
		}
		defer gr.Close()
		body = gr
	case isXzArchive(archivePath):
		xzr, xerr := xz.NewReader(file)
		if xerr != nil {
			return fail("Unable to create the xz reader", xerr.Error())
		}
		body = xzr
	}

	// PUT to path=/ so "data/" entries from the archive land at /data/ inside the volume.
	q := url.Values{}
	q.Set("path", "/")

	headers := http.Header{}
	headers.Set("Content-Type", "application/x-tar")

	resp, derr := client.Do(rest.Context, http.MethodPut, "/containers/"+containerID+"/archive", q, body, headers)
	if derr != nil {
		return fail("Unable to perform the restore http request", derr.Error())
	}
	defer resp.Body.Close()

	if serr := checkStatus(resp, http.StatusOK, http.StatusNoContent); serr != nil {
		return fail("Restore failed", serr.Error())
	}
	return nil
}
