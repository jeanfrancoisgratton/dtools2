// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 21:06
// Original filename: src/cmd/authCommands.go

package cmd

import "github.com/spf13/cobra"

// authCmd groups registry authentication related subcommands.
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage registry authentication",
	Long:  "Manage authentication data for container registries (e.g. docker login equivalent).",
}

func init() {
	rootCmd.AddCommand(authCmd)
}
