// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/10/18
// Original filename: src/volumes/backup.go

package volumes

import (
	"compress/gzip"
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

// BackupVolume streams /containers/{id}/archive (download) to a local file (optionally gzip/xz-compressed).
// We request ?path=/data (the volume's bind mount point) so the tar stream only contains the
// volume's own content, rooted at "data/..." (e.g. "data/pg_data/..."), not the rest of the temp
// container's filesystem. This matches the restore side which PUTs to ?path=/, causing "data/" to
// land at /data/.
//
// Should anything fail once the attached containers are stopped, they are restarted:
// a backup never alters the volume, so this is always safe.
func BackupVolume(client *rest.Client, volumeName, archivePath string) *ce.CustomError {
	archivePath = normalizeArchiveName(archivePath)

	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign(fmt.Sprintf("Backing up %s to %s", volumeName, archivePath)))
		showTimeoutNotes()
	}
	attachedContainers, err := getContainersUsingVolume(client, volumeName)
	if !rest.QuietOutput {
		showAttachedContainers(attachedContainers)
	}
	if err != nil {
		return err
	}
	if len(attachedContainers) > 0 {
		if !rest.QuietOutput {
			fmt.Println(hftx.InProgressSign(fmt.Sprintf("Temporarily stopping the running container(s) using %s", volumeName)))
		}
		if stopped, e := stopContainers(client, attachedContainers); e != nil {
			restartAfterFailure(client, stopped)
			return e
		}
	}

	if e := backupToArchive(client, volumeName, archivePath); e != nil {
		restartAfterFailure(client, attachedContainers)
		return e
	}

	// restart the containers that were stopped before the backup
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign("Restarting the containers that were stopped before the backup"))
	}
	if scerr := startContainers(client, attachedContainers); scerr != nil {
		return scerr
	}
	if !rest.QuietOutput {
		fmt.Printf("%s volume %s backed up as %s\n", hftx.EnabledSign(""), hftx.Blue(volumeName), hftx.Blue(archivePath))
	}
	return nil
}

// backupToArchive handles the temp container lifecycle and the archive transfer
func backupToArchive(client *rest.Client, volumeName, archivePath string) *ce.CustomError {
	// Create a temp container bound to the volume and start it
	if !rest.QuietOutput {
		fmt.Println(hftx.InProgressSign(fmt.Sprintf("Creating and starting a temp Alpine container to temporarily attach the volume %s", volumeName)))
	}
	if e := ensureImageExists(client, BackupImage); e != nil {
		return e
	}
	containerID, cerr := startTempContainer(client, BackupImage, volumeName)
	if cerr != nil {
		return cerr
	}

	if e := downloadArchive(client, containerID, archivePath); e != nil {
		// Never leave a truncated archive behind: it would look like a valid backup
		_ = os.Remove(archivePath)
		cleanupTempContainer(client, containerID)
		return e
	}

	if !NoCleanup {
		if !rest.QuietOutput {
			fmt.Println(hftx.InProgressSign("Cleanup: stoping and removing the temp Alpine container"))
		}
		if e := stopAndRemoveContainer(client, containerID); e != nil {
			return e
		}
	}
	return nil
}

func downloadArchive(client *rest.Client, containerID, archivePath string) *ce.CustomError {
	// Request path=/data (the volume's bind mount point, see createTempContainer) so the tar
	// stream only contains the volume's own content, rooted as "data/..." — not the rest of the
	// temp container's filesystem (which would include /etc, /bin, etc. and break restore, since
	// those are Docker-managed bind mounts that can't be overwritten/removed on extraction).
	// On restore we PUT to path=/ so "data/" extracts cleanly to /data/ inside the volume.
	q := url.Values{}
	q.Set("path", "/data")

	resp, doErr := client.Do(rest.Context, http.MethodGet, "/containers/"+containerID+"/archive", q, nil, nil)
	if doErr != nil {
		return fail("Failed to retrieve archive path from container", doErr.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fail("Error retrieving archive", checkStatus(resp, http.StatusOK).Error())
	}

	// Open destination file
	outf, ferr := os.Create(archivePath)
	if ferr != nil {
		return fail("Failed to create archive file", ferr.Error())
	}
	defer outf.Close()

	// If compressed, wrap the destination; DO NOT re-tar — the daemon already sends a tar stream
	var writer io.Writer = outf
	var compressor io.WriteCloser
	switch {
	case isGzipArchive(archivePath):
		compressor = gzip.NewWriter(outf)
	case isXzArchive(archivePath):
		xzw, xerr := xz.NewWriter(outf)
		if xerr != nil {
			return fail("Unable to create the xz writer", xerr.Error())
		}
		compressor = xzw
	}
	if compressor != nil {
		writer = compressor
	}

	// Directly copy the daemon's tar stream to the destination (raw or compressed)
	if _, werr := io.Copy(writer, resp.Body); werr != nil {
		return fail("Failed to write data to archive", werr.Error())
	}

	// Closing the compressor flushes its trailer: an error here means a corrupted archive
	if compressor != nil {
		if cerr := compressor.Close(); cerr != nil {
			return fail("Failed to finalize the compressed archive", cerr.Error())
		}
	}
	if cerr := outf.Close(); cerr != nil {
		return fail("Failed to close the archive file", cerr.Error())
	}
	return nil
}
