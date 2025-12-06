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

// containerCmd groups container-related subcommands.
var containerCmd = &cobra.Command{
	Use:   "container",
	Short: "Manage containers",
	Long:  "Manage containers via the Docker/Podman API (pull, list, etc.).",
}

var containerListCmd = &cobra.Command{
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

var containerInfoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"lsc"},
	Example: "dtools2 container ls [-r|-x]]",
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

var containerRemoveCmd = &cobra.Command{
	Use:     "rm [flags]",
	Aliases: []string{"remove", "del", "delete"},
	Example: "dtools2 container rm [-f] [-k] [-r] container1 container2 .. containerN",
	Short:   "Removes one or many containers",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := containers.RemoveContainer(restClient, args); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(containerCmd, containerListCmd, containerInfoCmd, containerRemoveCmd)
	containerCmd.AddCommand(containerListCmd, containerInfoCmd, containerRemoveCmd)

	containerRemoveCmd.Flags().BoolVarP(&containers.KillRunningContainers, "kill", "k", false, "remove container even if running")
	containerRemoveCmd.Flags().BoolVarP(&containers.RemoveUnamedVolumes, "remove-vols", "r", true, "remove non-named volume")
	containerRemoveCmd.Flags().BoolVarP(&containers.RemoveBlacklistedContainers, "force", "f", false, "remove container even if blacklisted")
	containerListCmd.Flags().BoolVarP(&containers.OnlyRunningContainers, "running", "r", false, "List only the running containers")
	containerListCmd.Flags().BoolVarP(&containers.ExtendedContainerInfo, "extended", "x", false, "Show extended container info")
}
