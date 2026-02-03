// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 22:01
// Original filename: src/cmd/imagesCommands.go

package cmd

import (
	"dtools2/env"
	"dtools2/system"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetch image information",
}

var getCatalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "fetches the registry's full catalog, in JSON format",
	Run: func(cmd *cobra.Command, args []string) {
		if env.RegConfigFile == "" {
			env.RegConfigFile = filepath.Join(os.Getenv("HOME"), ".config", "JFG", "dtools", "defaultRegistry.json")
		}

		if err := system.GetCatalog(); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var getTagsCmd = &cobra.Command{
	Use:   "tags IMAGE_NAME",
	Short: "fetches all of the tags for a given image",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if env.RegConfigFile == "" {
			env.RegConfigFile = filepath.Join(os.Getenv("HOME"), ".config", "JFG", "dtools", "defaultRegistry.json")
		}
		if err := system.GetTags(args[0]); err != nil {
			fmt.Println(err)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
	getCmd.AddCommand(getCatalogCmd, getTagsCmd)
	getCatalogCmd.Flags().StringVarP(&env.RegConfigFile, "registryfile", "r", "", "registry config file")
	getCatalogCmd.Flags().StringVarP(&system.JSONoutputfile, "output", "o", "", "send output to file")
	getTagsCmd.Flags().StringVarP(&env.RegConfigFile, "registryfile", "r", "", "registry config file")
	getTagsCmd.Flags().StringVarP(&system.JSONoutputfile, "file", "f", "", "send output to file")

}
