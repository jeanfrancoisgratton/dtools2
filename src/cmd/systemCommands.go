// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 22:01
// Original filename: src/cmd/imagesCommands.go

package cmd

import (
	"dtools2/env"
	"dtools2/system"
	"fmt"

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
			fmt.Println("No registry config file specified (see the -r flag), or file is missing")
			return
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

		if err := system.GetTags(args[0]); err != nil {
			fmt.Println(err)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(sysCmd)
	sysCmd.AddCommand(sysGetCatalogCmd, sysGetTagsCmd)

	sysCmd.Flags().StringVarP(&env.RegConfigFile, "registryfile", "r", "", "registry config file")
	sysCmd.Flags().StringVarP(&system.JSONoutputfile, "output", "o", "", "send output to file")
}
