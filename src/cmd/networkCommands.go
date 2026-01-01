// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/12/14 20:22
// Original filename: src/cmd/networkCommands.go

package cmd

import (
	"dtools2/networks"
	"dtools2/rest"
	"fmt"

	"github.com/spf13/cobra"
)

var networkCmd = &cobra.Command{
	Use:     "network",
	Aliases: []string{"net", "networks"},
	Short:   "Manage docker/podman networks",
	Long:    "Manage docker networks via the Docker/Podman API.",
}

var networkListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"lsn", "ls"},
	Short:   "List networks",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		if err := networks.ListNetworks(restClient); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var networkAddCmd = &cobra.Command{
	Use:     "add",
	Aliases: []string{"create"},
	Short:   "Add a network",
	Long: `
	Add a network to the daemon.
	You should note that a single daemon cannot have more than a single host or null network`,
	Example: "dtools net add NETWORK_NAME [flags]",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		if err := networks.AddNetwork(restClient, args[0]); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var networkRmCmd = &cobra.Command{
	Use:     "rm",
	Aliases: []string{"remove", "rmn"},
	Short:   "Remove a network",
	Example: "dtools net rm NETWORK_NAME1 [NETWORK_NAME2..NETWORK_NAMEn]",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		if err := networks.RemoveNetwork(restClient, args); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var networkAttachCmd = &cobra.Command{
	Use:     "connect",
	Aliases: []string{"attach", "att"},
	Short:   "Connect (attach) a network to a container",
	Example: "dtools net connect NETWORK_NAME CONTAINER_NAME",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		//containerName := args[0]
		//networkName := args[1]
		if err := networks.AttachNetwork(restClient, args[0], args[1]); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var networkDetachCmd = &cobra.Command{
	Use:     "disconnect",
	Aliases: []string{"detach", "det"},
	Short:   "Disconnect (detach) a network from a container",
	Example: "dtools net disconnect NETWORK_NAME CONTAINER_NAME",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		//containerName := args[0]
		//networkName := args[1]
		if err := networks.DetachNetwork(restClient, args[0], args[1]); err != nil {
			fmt.Println(err)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(networkCmd, networkListCmd, networkRmCmd)
	networkCmd.AddCommand(networkListCmd, networkAddCmd, networkRmCmd, networkAttachCmd, networkDetachCmd)

	networkDetachCmd.Flags().BoolVarP(&networks.ForceNetworkDetach, "force", "f", false, "force-detach the network from the container")
	networkAddCmd.Flags().StringVarP(&networks.NetworkDriverName, "driver", "d", "bridge", "network driver network")
	networkAddCmd.Flags().BoolVarP(&networks.NetworkEnableIPv6, "ipv6", "6", false, "enable IPv6 on the network")
	networkAddCmd.Flags().BoolVarP(&networks.NetworkInternalUse, "internal", "i", false, "internal network only")
	networkAddCmd.Flags().BoolVarP(&networks.NetworkAttachable, "attachable", "a", false, "network is attachable (no effect on bridged networks)")
	networkRmCmd.Flags().BoolVarP(&networks.RemoveEvenIfBlackListed, "blacklist", "B", false, "remove network even if blacklisted")
}
