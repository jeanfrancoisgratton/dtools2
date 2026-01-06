// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 22:01
// Original filename: src/cmd/imagesCommands.go

package cmd

import (
	"dtools2/env"
	"dtools2/rest"
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

// *DO NOT* confuse rm and rmc
// rm removes all non-running/non-paused containers, while rmc removes all targeted container(s)
var systemRmCmd = &cobra.Command{
	Use:   "rms",
	Short: "Remove all exited or created (NOT running/paused) containers",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := system.RmContainers(restClient); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

// Clean : remove unused images, volumes and networks
var systemCleanCmd = &cobra.Command{
	Use:   "clean [flags]",
	Short: "Remove all unused images, volumes and networks",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := system.Clean(restClient); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

var sysInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show daemon system information (server section only)",
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()
		if errCode := system.Info(restClient); errCode != nil {
			fmt.Println(errCode)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(sysCmd, systemRmCmd, systemCleanCmd)
	sysCmd.AddCommand(sysGetCatalogCmd, sysGetTagsCmd, systemRmCmd, systemCleanCmd, sysInfoCmd)

	sysGetCatalogCmd.Flags().StringVarP(&env.RegConfigFile, "registryfile", "r", "", "registry config file")
	sysGetCatalogCmd.Flags().StringVarP(&system.JSONoutputfile, "output", "o", "", "send output to file")
	sysGetTagsCmd.Flags().StringVarP(&env.RegConfigFile, "registryfile", "r", "", "registry config file")
	sysGetTagsCmd.Flags().StringVarP(&system.JSONoutputfile, "output", "o", "", "send output to file")
	systemRmCmd.Flags().BoolVarP(&system.ForceRemove, "force", "f", false, "force removal of container")
	systemRmCmd.Flags().BoolVarP(&system.RemoveUnamedVolumes, "remove-vols", "r", true, "remove non-named volume")
	systemRmCmd.Flags().BoolVarP(&system.RemoveBlacklisted, "blacklist", "B", false, "remove container even if blacklisted")
	systemCleanCmd.Flags().BoolVarP(&system.RemoveBlacklisted, "blacklist", "B", false, "remove container even if blacklisted")
	systemCleanCmd.Flags().BoolVarP(&system.ForceRemove, "force", "f", false, "force removal of container")

}
