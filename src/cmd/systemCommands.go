// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 22:01
// Original filename: src/cmd/imagesCommands.go

package cmd

import (
	"dtools2/registry"
	"dtools2/rest"
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
	Use:   "getcatalog",
	Short: "fetches the registry's full catalog, in JSON format",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		drf := ""
		if len(args) == 0 {
			drf = registry.RegConfigFile
		}

		rest.Context = cmd.Context()
		if err := system.GetCatalog(restClient, drf); err != nil {
			fmt.Println(err)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(sysCmd)
	sysCmd.AddCommand(sysGetCatalogCmd)

	sysCmd.Flags().StringVarP(&registry.RegConfigFile, "registryfile", "r", "", "registry config file")
}
