// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/29 09:14
// Original filename: src/cmd/imageCommands.go

package cmd

import (
	"context"
	"dtools2/image"

	"github.com/spf13/cobra"
)

var imageCmd = &cobra.Command{
	Use:     "image",
	Aliases: []string{"img"},
	Short:   "Manage images. Available commands are { pull, push, list, remove }",
}

var imagePullCmd = &cobra.Command{
	Use:   "pull image [image2...imageN]",
	Short: "Pull an image",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		return image.Pull(ctx, client, args)
	},
}

func init() {
	rootCmd.AddCommand(imageCmd)
	imageCmd.AddCommand(imagePullCmd)

}
