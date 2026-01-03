// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/31 13:42
// Original filename: src/cmd/volumesCommands.go

package cmd

import (
	"dtools2/rest"
	"dtools2/volumes"
	"fmt"

	"github.com/spf13/cobra"
)

var volumeCmd = &cobra.Command{
	Use:     "volume",
	Aliases: []string{"vol"},
	Short:   "Manage docker/podman volumes",
	Long:    "Manage docker volumes via the Docker/Podman API.",
}

var volumeListCmd = &cobra.Command{
	Use:   "lsv",
	Short: "List volumes",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		if _, err := volumes.ListVolumes(restClient, true); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var volumeRmCmd = &cobra.Command{
	Use:     "rmv",
	Example: "docker volume rmv [flags] volume_name1 [volume_name2..volume_nameN]",
	Short:   "Remove one or many volumes",
	Long: `Remove docker volumes via the Docker/Podman API.
		Blacklisted volumes will not be removed, unless the -B flag is passed.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		if err := volumes.RemoveVolumes(restClient, args); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var volumePruneCmd = &cobra.Command{
	Use:     "prune",
	Example: "docker volume prune [flags]",
	Short:   "Prune volumes",
	Long: `Prune volumes via the Docker/Podman API.
		Blacklisted volumes will not be removed, unless the -B flag is passed.`,
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		if err := volumes.PruneVolumes(restClient); err != nil {
			fmt.Println(err)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(volumeCmd, volumeListCmd, volumeRmCmd)
	volumeCmd.AddCommand(volumeListCmd, volumeRmCmd, volumePruneCmd)

	volumePruneCmd.Flags().BoolVarP(&volumes.RemoveEvenIfBlackListed, "blacklist", "B", false, "remove volume even if blacklisted")
	volumePruneCmd.Flags().BoolVarP(&volumes.RemoveNamedVolumes, "all", "a", false, "remove anonymous AND non-anonymous volumes")
	volumeRmCmd.Flags().BoolVarP(&volumes.RemoveEvenIfBlackListed, "blacklist", "B", false, "remove volume even if blacklisted")
	volumeRmCmd.Flags().BoolVarP(&volumes.ForceRemoval, "force", "f", false, "force-remove volume")
	//networkDetachCmd.Flags().BoolVarP(&networks.ForceNetworkDetach, "force", "f", false, "force-detach the network from the container")
	//networkAddCmd.Flags().StringVarP(&networks.NetworkDriverName, "driver", "d", "bridge", "network driver network")
	//networkAddCmd.Flags().BoolVarP(&networks.NetworkEnableIPv6, "ipv6", "6", false, "enable IPv6 on the network")
	//networkAddCmd.Flags().BoolVarP(&networks.NetworkInternalUse, "internal", "i", false, "internal network only")
	//networkAddCmd.Flags().BoolVarP(&networks.NetworkAttachable, "attachable", "a", false, "network is attachable (no effect on bridged networks)")
	//networkRmCmd.Flags().BoolVarP(&networks.RemoveEvenIfBlackListed, "blacklist", "B", false, "remove network even if blacklisted")
}
