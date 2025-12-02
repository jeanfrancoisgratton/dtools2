// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/24 20:20
// Original filename: src/cmd/containersCommands.go

package cmd

import (
	"dtools2/containers"
	"dtools2/rest"
	"fmt"

	"github.com/spf13/cobra"
)

// containersCmd groups container-related subcommands.
var containersCmd = &cobra.Command{
	Use:   "container",
	Short: "Manage containers",
	Long:  "Manage containers via the Docker/Podman API (pull, list, etc.).",
}

var containersListCmd = &cobra.Command{
	Use:     "ls [flags]",
	Aliases: []string{"lsc"},
	Example: "dtools2 containers ls [-r|-a]]",
	Short:   "Lists the containers",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		_, errCode := containers.ListContainers(restClient, true)
		if errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

var containersInfoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"lsc"},
	Example: "dtools2 containers ls [-r|-a]]",
	Short:   "Lists the containers",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()

		if errCode := containers.InfoContainers(restClient, args[0]); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(containersCmd, containersListCmd, containersInfoCmd)
	containersCmd.AddCommand(containersListCmd, containersInfoCmd)

	containersListCmd.Flags().BoolVarP(&containers.OnlyRunningContainers, "running", "r", false, "List only the running containers")
	containersListCmd.Flags().BoolVarP(&containers.ExtendedContainerInfo, "extended", "x", false, "Show extended container info")
}
