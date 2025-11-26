// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 22:01
// Original filename: src/cmd/imgCommands.go

package cmd

import (
	"dtools2/images"
	"dtools2/rest"
	"fmt"

	"github.com/spf13/cobra"
)

// imgCmd groups image-related subcommands.
var imgCmd = &cobra.Command{
	Use:   "image",
	Short: "Manage docker images",
	Long:  "Manage docker images via the Docker/Podman API (pull, list, etc.).",
}

// imagePullCmd implements `dtools2 images pull`, wiring through to images.ImagePull().
// cmd/images.go

var imagePullCmd = &cobra.Command{
	Use:   "pull IMAGE",
	Short: "Pull an image from a registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if restClient == nil {
			return fmt.Errorf("REST client not initialized")
		}

		imageRef := args[0]
		rest.Context = cmd.Context()
		return images.ImagePull(restClient, imageRef)
	},
}

// imagePullCmd implements `dtools2 images pull`, wiring through to images.ImagePull().
// cmd/images.go

var imagePushCmd = &cobra.Command{
	Use:   "push IMAGE",
	Short: "Push an image to a registry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if restClient == nil {
			return fmt.Errorf("REST client not initialized")
		}

		imageRef := args[0]
		rest.Context = cmd.Context()
		return images.ImagePush(cmd.Context(), restClient, imageRef)
	},
}

func init() {
	rootCmd.AddCommand(imgCmd, imagePullCmd, imagePushCmd)
	imgCmd.AddCommand(imagePullCmd, imagePushCmd)

	imagePullCmd.Flags().StringVarP(&imagePullRegistry, "registry", "r", "", "Registry hostname to use for auth (e.g. registry.example.com:5000); empty for anonymous")
}
