// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/06 01:10
// Original filename: src/images/dockerclean.go

package system

import (
	"dtools2/images"
	"dtools2/networks"
	"dtools2/rest"
	"dtools2/volumes"

	ce "github.com/jeanfrancoisgratton/customError/v3"
)

// clean : removes all unused images
// this will purge all unused images, volumes and networks

func CleanImages(client *rest.Client) *ce.CustomError {
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
				imgCandidates = append(imgCandidates, i.ID)
			}
		}
		if err := images.RemoveImage(client, imgCandidates); err != nil {
			return err
		}
	}

	// Remove volumes
	if err := volumes.PruneVolumes(client); err != nil {
		return err
	}

	// Remove networks
	if ns, err := networks.NetworkList(client, false); err != nil {
		return err
	} else {
		for _, i := range ns {
			if i.Name != "host" && i.Name != "none" && !i.InUse {
				netCandidates = append(netCandidates, i.ID)
			}
		}
		if err := networks.RemoveNetwork(client, netCandidates); err != nil {
			return err
		}
	}
	return nil
}
