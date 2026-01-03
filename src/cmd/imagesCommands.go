// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 22:01
// Original filename: src/cmd/imagesCommands.go

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
	Short: "Manage docker/podman images",
	Long:  "Manage docker/podman images via the Docker/Podman API (pull, list, etc.).",
}

// imagePullCmd implements `dtools2 images pull`, wiring through to images.ImagePull().
// cmd/images.go

var imagePullCmd = &cobra.Command{
	Use:   "pull IMAGE",
	Short: "Pull an image from a registry",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		imageRef := args[0]
		rest.Context = cmd.Context()
		if err := images.ImagePull(restClient, imageRef); err != nil {
			fmt.Println(err)
		}
		return
	},
}

// imagePullCmd implements `dtools2 images pull`, wiring through to images.ImagePull().
// cmd/images.go

var imagePushCmd = &cobra.Command{
	Use:   "push IMAGE",
	Short: "Push an image to a registry",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		imageRef := args[0]
		rest.Context = cmd.Context()
		if err := images.ImagePush(restClient, imageRef); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var imageListCmd = &cobra.Command{
	Use:   "lsi",
	Short: "List images",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		if err := images.ImagesList(restClient); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var imageTagCmd = &cobra.Command{
	Use:   "tag IMAGE:TAG IMAGE:NEWTAG",
	Short: "Tag image",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		if err := images.TagImage(restClient, args[0], args[1]); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var imageRemoveCmd = &cobra.Command{
	Use:     "rmi IMAGE_NAME",
	Example: "dtools2 rmi [-B] [-f] image_name1 [image_name2..image_nameN]",
	Short:   "Remove image",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}

		rest.Context = cmd.Context()
		if err := images.RemoveImage(restClient, args); err != nil {
			fmt.Println(err)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(imgCmd, imagePullCmd, imagePushCmd, imageListCmd, imageTagCmd, imageRemoveCmd)
	imgCmd.AddCommand(imagePullCmd, imagePushCmd, imageListCmd, imageTagCmd, imageRemoveCmd)

	imagePullCmd.Flags().StringVarP(&imagePullRegistry, "registry", "r", "", "registry hostname to use for auth (e.g. registry.example.com:5000); empty for anonymous")
	imageRemoveCmd.Flags().BoolVarP(&images.ForceRemove, "force", "f", false, "Force remove image")
	imageRemoveCmd.Flags().BoolVarP(&images.RemoveBlacklisted, "blacklist", "B", false, "remove image even if blacklisted")

}
