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

// imagesPullCmd implements `dtools2 images pull`, wiring through to images.ImagePull().
// cmd/images.go

var containerListCmd = &cobra.Command{
	Use:     "ls [flags]",
	Aliases: []string{"lsc"},
	Example: "dtools2 containers ls [-r] [-x]",
	Short:   "Lists the containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if restClient == nil {
			return fmt.Errorf("REST client not initialized")
		}
		rest.Context = cmd.Context()
		_, errCode := containers.ContainersList(restClient, true)
		return errCode
	},
}

func init() {
	rootCmd.AddCommand(containerCmd, containerListCmd)
	containerCmd.AddCommand(containerListCmd)

	containerListCmd.Flags().BoolVarP(&containers.OnlyRunningContainers, "running", "r", false, "List only the running containers")
	containerListCmd.Flags().BoolVarP(&containers.ExtendedContainerInfo, "extended", "x", false, "Show extended container info")
}
