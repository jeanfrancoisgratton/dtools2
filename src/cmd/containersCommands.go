// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/24 20:20
// Original filename: src/cmd/containersCommands.go

package cmd

import (
	"dtools2/containers"
	"fmt"

	"github.com/spf13/cobra"
)

	Use:   "container",
	Short: "Manage containers",
	Long:  "Manage containers via the Docker/Podman API (pull, list, etc.).",
}

	Use:     "ls [flags]",
	Aliases: []string{"lsc"},
	Short:   "Lists the containers",
	RunE: func(cmd *cobra.Command, args []string) error {
		if restClient == nil {
			return fmt.Errorf("REST client not initialized")
		}
		return errCode
	},
}

func init() {

}
