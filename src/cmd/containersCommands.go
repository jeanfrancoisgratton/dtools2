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

var containerPauseCmd = &cobra.Command{
	Use:     "pause",
	Example: "dtools2 pause container1 container2 .. containerN",
	Short:   "Pauses one or many containers",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := containers.PauseContainer(restClient, args); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

var containerUnpauseCmd = &cobra.Command{
	Use:     "unpause",
	Example: "dtools2 unpause container1 container2 .. containerN",
	Short:   "Unpauses one or many containers",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := containers.UnpauseContainer(restClient, args); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

var containerStartCmd = &cobra.Command{
	Use:     "start",
	Aliases: []string{"up"},
	Example: "dtools2 start container1 container2 .. containerN",
	Short:   "Starts one or many containers",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := containers.StartContainer(restClient, args); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

var containerStartAllCmd = &cobra.Command{
	Use:     "startall",
	Example: "dtools2 startall",
	Short:   "Starts all non-running containers",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := containers.StartAll(restClient); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

var containerStopCmd = &cobra.Command{
	Use:     "stop",
	Aliases: []string{"down"},
	Example: "dtools2 stop container1 container2 .. containerN",
	Short:   "Stops one or many containers",
	Long:    "Using a timeout of 0 (-t 0) will stop them concurrently, but conclusion is still dependent on the containers gracefully shut down",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := containers.StopContainer(restClient, args); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

var containerStopAllCmd = &cobra.Command{
	Use:     "stopall",
	Example: "dtools2 stopall",
	Short:   "Stops all running containers",
	Long:    "Using a timeout of 0 (-t 0) will stop them concurrently, but conclusion is still dependent on the containers gracefully shut down",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := containers.StopAll(restClient); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

var containerRenameCmd = &cobra.Command{
	Use:     "rename",
	Example: "dtools2 rename OLD_NAME NEW_NAME",
	Short:   "Rename a container",
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := containers.RenameContainer(restClient, args[0], args[1]); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(containerCmd, containerListCmd, containerInfoCmd, containerRemoveCmd, containerPauseCmd,
		containerUnpauseCmd, containerStartCmd, containerStartAllCmd, containerStopCmd, containerStopAllCmd, containerRenameCmd)
	containerCmd.AddCommand(containerListCmd, containerInfoCmd, containerRemoveCmd, containerPauseCmd,
		containerUnpauseCmd, containerUnpauseCmd, containerStartCmd, containerStartAllCmd, containerStopCmd, containerStopAllCmd, containerRenameCmd)

	containerStopCmd.Flags().IntVarP(&containers.StopTimeout, "timeout", "t", 10, "timeout (seconds) when stopping containers; 0 to stop all concurrently")
	containerStopAllCmd.Flags().IntVarP(&containers.StopTimeout, "timeout", "t", 10, "timeout (seconds) when stopping containers; 0 to stop all concurrently")
	containerRemoveCmd.Flags().BoolVarP(&containers.KillRunningContainers, "kill", "k", false, "remove container even if running")
	containerRemoveCmd.Flags().BoolVarP(&containers.RemoveUnamedVolumes, "remove-vols", "r", true, "remove non-named volume")
	containerRemoveCmd.Flags().BoolVarP(&containers.RemoveBlacklistedContainers, "force", "f", false, "remove container even if blacklisted")
	containerListCmd.Flags().BoolVarP(&containers.OnlyRunningContainers, "running", "r", false, "List only the running containers")
	containerListCmd.Flags().BoolVarP(&containers.ExtendedContainerInfo, "extended", "x", false, "Show extended container info")
}
