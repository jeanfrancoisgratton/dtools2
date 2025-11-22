// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 22:01
// Original filename: src/cmd/imagesCommands.go

package cmd

import (
	"dtools2/images"
	"fmt"

	"github.com/spf13/cobra"
)

// imagesCmd groups image-related subcommands.
var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage container images",
	Long:  "Manage container images via the Docker/Podman API (pull, list, etc.).",
}

// imagesPullCmd implements `dtools2 images pull`, wiring through to images.ImagePull().
// cmd/images.go

var imagesPullCmd = &cobra.Command{
	Use:   "pull IMAGE",
	Short: "Pull an image from a registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if restClient == nil {
			return fmt.Errorf("REST client not initialized")
		}

		imageRef := args[0]
		return images.ImagePull(cmd.Context(), restClient, imageRef)
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd, imagesPullCmd)
	imagesCmd.AddCommand(imagesPullCmd)

	imagesPullCmd.Flags().StringVarP(&imagePullRegistry, "registry", "r", "", "Registry hostname to use for auth (e.g. registry.example.com:5000); empty for anonymous")
}
