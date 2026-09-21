// dtools2
// src/cmd/backend_commands.go

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// notSupportedByBackend reports that the current backend doesn't implement a
// capability (containerd has no networks/volumes/run/build/cp of its own).
func notSupportedByBackend(op string) {
	fmt.Printf("%q is not supported by the %q backend\n", op, activeBackend.Name())
}

// backendCmd groups backend-related subcommands.
var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "Inspect the container runtime backend dtools is using",
}

var backendActiveCmd = &cobra.Command{
	Use:   "active",
	Short: "Show the currently selected backend",
	Run: func(cmd *cobra.Command, args []string) {
		if activeBackend == nil {
			fmt.Println("Backend not initialized")
			return
		}
		fmt.Println(activeBackend.Name())
	},
}

func init() {
	rootCmd.AddCommand(backendCmd)
	backendCmd.AddCommand(backendActiveCmd)
}
