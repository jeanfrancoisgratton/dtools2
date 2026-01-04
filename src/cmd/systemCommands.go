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

var sysCmd = &cobra.Command{
	Use:     "system",
	Aliases: []string{"sys"},
	Short:   "System and extra commands",
	Long:    "Some system commands and extra features not found in the official clients",
}

var sysGetCatalogCmd = &cobra.Command{
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

var sysGetTagsCmd = &cobra.Command{
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
	rootCmd.AddCommand(sysCmd)
	sysCmd.AddCommand(sysGetCatalogCmd, sysGetTagsCmd)

	sysGetCatalogCmd.Flags().StringVarP(&env.RegConfigFile, "registryfile", "r", "", "registry config file")
	sysGetCatalogCmd.Flags().StringVarP(&system.JSONoutputfile, "output", "o", "", "send output to file")
	sysGetTagsCmd.Flags().StringVarP(&env.RegConfigFile, "registryfile", "r", "", "registry config file")
	sysGetTagsCmd.Flags().StringVarP(&system.JSONoutputfile, "output", "o", "", "send output to file")

}
