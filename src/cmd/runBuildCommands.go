// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 23:45
// Original filename: src/cmd/runBuildCommands.go

package cmd

import (
	"dtools2/build"
	"dtools2/rest"
	"dtools2/run"
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:     "run [flags] IMAGE [COMMAND] [ARG...]",
	Short:   "Run a command in a new container",
	Example: "dtools run -it --rm alpine:latest /bin/sh",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()

		image := args[0]
		command := []string{}
		if len(args) > 1 {
			command = args[1:]
		}

		_, id, cerr := run.RunContainer(restClient, image, command)
		if cerr != nil {
			fmt.Println(cerr)
			return
		}

		if run.RunDetach {
			if id != "" {
				fmt.Println(id)
			}
			return
		}
		return
	},
}

var buildCmd = &cobra.Command{
	Use:     "build [flags] PATH",
	Short:   "Build an image from a Dockerfile",
	Example: "dtools build -t myimg:latest -f Dockerfile .",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()

		if err := build.BuildImage(restClient, args[0]); err != nil {
			fmt.Println(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd, buildCmd)

	runCmd.Flags().BoolVarP(&run.RunDetach, "detach", "d", false, "Run container in background and print container ID")
	runCmd.Flags().BoolVarP(&run.RunInteractive, "interactive", "i", false, "Keep STDIN open even if not attached")
	runCmd.Flags().BoolVarP(&run.RunTTY, "tty", "t", false, "Allocate a pseudo-TTY")
	runCmd.Flags().BoolVar(&run.RunRemove, "rm", false, "Automatically remove the container when it exits")
	runCmd.Flags().StringVar(&run.RunName, "name", "", "Assign a name to the container")
	runCmd.Flags().StringVarP(&run.RunUser, "user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	runCmd.Flags().StringVarP(&run.RunWorkdir, "workdir", "w", "", "Working directory inside the container")
	runCmd.Flags().StringArrayVarP(&run.RunEnv, "env", "e", nil, "Set environment variables")
	runCmd.Flags().StringArrayVarP(&run.RunPublish, "publish", "p", nil, "Publish a container's port(s) to the host")
	runCmd.Flags().StringArrayVarP(&run.RunVolume, "volume", "v", nil, "Bind mount a volume")
	runCmd.Flags().StringVar(&run.RunNetwork, "network", "", "Connect a container to a network")
	runCmd.Flags().StringVar(&run.RunEntrypoint, "entrypoint", "", "Overwrite the default ENTRYPOINT of the image")
	runCmd.Flags().StringVarP(&run.RunHostname, "hostname", "h", "", "Container host name")
	buildCmd.Flags().StringVarP(&build.Dockerfile, "file", "f", "Dockerfile", "Name of the Dockerfile (relative to PATH)")
	buildCmd.Flags().StringArrayVarP(&build.Tags, "tag", "t", nil, "Name and optional tag in the 'name:tag' format")
	buildCmd.Flags().StringArrayVar(&build.BuildArgs, "build-arg", nil, "Set build-time variables")
	buildCmd.Flags().BoolVar(&build.NoCache, "no-cache", false, "Do not use cache when building the image")
	buildCmd.Flags().BoolVar(&build.Pull, "pull", false, "Always attempt to pull a newer version of the base images")
	buildCmd.Flags().BoolVar(&build.RemoveIntermediate, "rm", true, "Remove intermediate containers after a successful build")
	buildCmd.Flags().BoolVar(&build.ForceRemoveIntermediate, "force-rm", false, "Always remove intermediate containers, even upon failure")
	buildCmd.Flags().StringVar(&build.Target, "target", "", "Set the target build stage to build")
	buildCmd.Flags().StringVar(&build.Platform, "platform", "", "Set platform if supported by the daemon")
	buildCmd.Flags().StringVar(&build.Progress, "progress", "auto", "Set type of progress output (auto|plain|tty)")
}
