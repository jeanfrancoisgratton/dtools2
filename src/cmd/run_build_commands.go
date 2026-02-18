// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 23:45
// Original filename: src/cmd/runBuildCommands.go

package cmd

import (
	"dtools2/rest"
	"dtools2/run_build"
	"fmt"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:     "run [flags] IMAGE [COMMAND] [ARG...]",
	Aliases: []string{"run_build"},
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

		_, id, cerr := run_build.RunContainer(restClient, image, command)
		if cerr != nil {
			fmt.Println(cerr)
			return
		}

		if run_build.RunDetach {
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

		if err := run_build.BuildImage(restClient, args[0]); err != nil {
			fmt.Println(err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(runCmd, buildCmd)

	runCmd.Flags().BoolVarP(&run_build.RunDetach, "detach", "d", false, "Run container in background and print container ID")
	runCmd.Flags().BoolVarP(&run_build.RunInteractive, "interactive", "i", false, "Keep STDIN open even if not attached")
	runCmd.Flags().BoolVarP(&run_build.RunTTY, "tty", "t", false, "Allocate a pseudo-TTY")
	runCmd.Flags().BoolVar(&run_build.RunRemove, "rm", false, "Automatically remove the container when it exits")
	runCmd.Flags().StringVar(&run_build.RunName, "name", "", "Assign a name to the container")
	runCmd.Flags().StringVarP(&run_build.RunUser, "user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	runCmd.Flags().StringVarP(&run_build.RunWorkdir, "workdir", "w", "", "Working directory inside the container")
	runCmd.Flags().StringArrayVarP(&run_build.RunEnv, "env", "e", nil, "Set environment variables")
	runCmd.Flags().StringArrayVarP(&run_build.RunPublish, "publish", "p", nil, "Publish a container's port(s) to the host")
	runCmd.Flags().StringArrayVarP(&run_build.RunVolume, "volume", "v", nil, "Bind mount a volume")
	runCmd.Flags().StringArrayVar(&run_build.RunMount, "mount", nil, "Attach a filesystem mount to the container (e.g. type=bind,src=/host,dst=/ctr,ro)")
	runCmd.Flags().StringVar(&run_build.RunNetwork, "network", "", "Connect a container to a network")
	runCmd.Flags().StringVar(&run_build.RunEntrypoint, "entrypoint", "", "Overwrite the default ENTRYPOINT of the image")
	runCmd.Flags().StringVarP(&run_build.RunHostname, "hostname", "", "", "Container host name")

	buildCmd.Flags().StringVarP(&run_build.Dockerfile, "file", "f", "Dockerfile", "Name of the Dockerfile (relative to PATH)")
	buildCmd.Flags().StringArrayVarP(&run_build.Tags, "tag", "t", nil, "Name and optional tag in the 'name:tag' format")
	buildCmd.Flags().StringArrayVar(&run_build.BuildArgs, "build-arg", nil, "Set build-time variables")
	buildCmd.Flags().BoolVar(&run_build.NoCache, "no-cache", false, "Do not use cache when building the image")
	buildCmd.Flags().BoolVar(&run_build.Pull, "pull", false, "Always attempt to pull a newer version of the base images")
	buildCmd.Flags().BoolVar(&run_build.RemoveIntermediate, "rm", true, "Remove intermediate containers after a successful build")
	buildCmd.Flags().BoolVar(&run_build.ForceRemoveIntermediate, "force-rm", false, "Always remove intermediate containers, even upon failure")
	buildCmd.Flags().BoolVar(&run_build.Compress, "compress", false, "Compress the build context sent to the daemon")
	buildCmd.Flags().BoolVar(&run_build.Load, "load", false, "No-op compatibility flag (image is always loaded into the local daemon)")
	buildCmd.Flags().StringVar(&run_build.Target, "target", "", "Set the target build stage to build")
	buildCmd.Flags().StringVar(&run_build.Platform, "platform", "", "Set platform if supported by the daemon")
	buildCmd.Flags().StringVar(&run_build.Progress, "progress", "auto", "Set type of progress output (auto|plain|tty)")
}
