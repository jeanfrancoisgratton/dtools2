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
	Aliases: []string{"vol", "volumes"},
	Short:   "Manage docker/podman volumes",
	Long:    "Manage docker volumes via the Docker/Podman API.",
}

var volumeListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"lsv", "ls"},
	Short:   "List volumes",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		if err := volumes.ListVolumes(restClient); err != nil {
			fmt.Println(err)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(volumeCmd, volumeListCmd)
	volumeCmd.AddCommand(volumeListCmd)

	//networkDetachCmd.Flags().BoolVarP(&networks.ForceNetworkDetach, "force", "f", false, "force-detach the network from the container")
	//networkAddCmd.Flags().StringVarP(&networks.NetworkDriverName, "driver", "d", "bridge", "network driver network")
	//networkAddCmd.Flags().BoolVarP(&networks.NetworkEnableIPv6, "ipv6", "6", false, "enable IPv6 on the network")
	//networkAddCmd.Flags().BoolVarP(&networks.NetworkInternalUse, "internal", "i", false, "internal network only")
	//networkAddCmd.Flags().BoolVarP(&networks.NetworkAttachable, "attachable", "a", false, "network is attachable (no effect on bridged networks)")
	//networkRmCmd.Flags().BoolVarP(&networks.RemoveEvenIfBlackListed, "blacklist", "B", false, "remove network even if blacklisted")
}
