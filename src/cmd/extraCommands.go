// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2026/01/03 23:34
// Original filename: src/cmd/extraCommands.go

package cmd

import (
	"dtools2/extras"
	"dtools2/rest"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:     "exec [flags] CONTAINER COMMAND [ARG...]",
	Short:   "Run a command in a running container",
	Example: "dtools exec -it mycontainer /bin/sh",
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			os.Exit(1)
		}
		rest.Context = cmd.Context()

		container := args[0]
		command := args[1:]

		exitCode, cerr := extras.Run(restClient, container, command)
		if cerr != nil {
			fmt.Println(cerr)
			os.Exit(1)
		}
		os.Exit(exitCode)
	},
}

var logsCmd = &cobra.Command{
	Use:     "logs [flags] CONTAINER",
	Aliases: []string{"log"},
	Short:   "Fetch the logs of a container",
	Example: "dtools logs -t -n 200 -f mycontainer",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()

		if cerr := extras.Logs(restClient, args[0]); cerr != nil {
			fmt.Println(cerr)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(execCmd, logsCmd, runCmd)

	execCmd.Flags().BoolVarP(&extras.Interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
	execCmd.Flags().BoolVarP(&extras.AllocateTTY, "tty", "t", false, "Allocate a pseudo-TTY")
	execCmd.Flags().StringVarP(&extras.User, "user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	logsCmd.Flags().BoolVarP(&extras.LogTimestamps, "timestamps", "t", false, "Show timestamps")
	logsCmd.Flags().IntVarP(&extras.LogTail, "tail", "n", -1, "Number of lines to show from the end of the logs (-1 means all)")
	logsCmd.Flags().BoolVarP(&extras.LogFollow, "follow", "f", false, "Follow log output")

}
