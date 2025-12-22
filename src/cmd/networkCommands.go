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
	Long:    "Manage docker networks via the Docker/Podman API (pull, list, etc.).",
}

var networkListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"lsn"},
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

func init() {
	rootCmd.AddCommand(networkCmd, networkListCmd)
	networkCmd.AddCommand(networkListCmd)
}
