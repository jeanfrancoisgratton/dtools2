// system/dockerclean.go
// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/06 01:10
// Original filename: src/images/dockerclean.go

package system

import (
	"dtools2/extras"
	"dtools2/images"
	"dtools2/networks"
	"dtools2/rest"
	"dtools2/volumes"
	"fmt"

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
)

// clean : removes all unused images
// this will purge all unused images, volumes and networks

func Clean(client *rest.Client) *ce.CustomError {
	// Ensure that the Blacklisted policy is enforced across resources
	images.RemoveBlacklisted = RemoveBlacklisted
	volumes.RemoveBlackListed = RemoveBlacklisted
	networks.RemoveBlacklisted = RemoveBlacklisted
	images.ForceRemove = ForceRemove
	volumes.ForceRemove = ForceRemove

	imgCandidates := []string{}
	netCandidates := []string{}

	// Remove images
	if is, err := images.ImagesList(client, false); err != nil {
		return err
	} else {
		for _, i := range is {
			if i.Containers == 0 {
				// Prefer repo:tag references over image IDs.
				// Rationale: a single image ID can be referenced by multiple tags.
				// Removing by ID can yield 409 Conflict responses from the daemon
				// depending on tag/reference state. By removing each repo:tag, we
				// untag cleanly and the final untag triggers the actual image delete.
				//
				// Note: dangling images often appear as "<none>:<none>" which is not
				// a valid name to delete; fall back to the image ID in that case.
				if i.RepoImgName != "" && i.ImgTag != "" && i.RepoImgName != "<none>" && i.ImgTag != "<none>" {
					imgCandidates = append(imgCandidates, fmt.Sprintf("%s:%s", i.RepoImgName, i.ImgTag))
				} else {
					imgCandidates = append(imgCandidates, i.ID)
				}
			}
		}
		// De-dupe IDs (for instance, when a image is used multiple times)
		imgCandidates = extras.Unique(imgCandidates)

		q := rest.QuietOutput
		rest.QuietOutput = true
		if err := images.RemoveImage(client, imgCandidates); err != nil {
			return err
		}
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Removed %d image(s)", len(imgCandidates))))
		rest.QuietOutput = q
	}

	// Remove volumes
	if err := volumes.PruneVolumes(client); err != nil {
		return err
	}

	// Remove networks
	if ns, err := networks.NetworkList(client, false); err != nil {
		return err
	} else {
		for _, n := range ns {
			if n.Name != "host" && n.Name != "none" && n.Name != "bridge" && !n.InUse {

				netCandidates = append(netCandidates, n.Name)
			}
		}
		q := rest.QuietOutput
		rest.QuietOutput = true
		if err := networks.RemoveNetwork(client, netCandidates); err != nil {
			return err
		}
		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Removed %d network(s)", len(netCandidates))))
		rest.QuietOutput = q
	}
	return nil
}
