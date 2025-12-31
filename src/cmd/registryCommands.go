// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 22:01
// Original filename: src/cmd/imgCommands.go

package cmd

import (
	"dtools2/registry"
	"fmt"

	hf "github.com/jeanfrancoisgratton/helperFunctions/v4"
	"github.com/spf13/cobra"
)

var registryCmd = &cobra.Command{
	Use:     "registry",
	Aliases: []string{"reg", "repo"},
	Short:   "Manage default registry handling",
	//Long:  "Manage docker/podman images via the Docker/Podman API (pull, list, etc.).",
}

var registryRemoveCmd = &cobra.Command{
	Use:     "registry remove",
	Example: "dtools2 registry remove",
	Aliases: []string{"rm"},
	Short:   "Remove the default registry entry and leave a blank entry instead",
	Run: func(cmd *cobra.Command, args []string) {

		re := registry.RegistryEntry{}
		if err := re.RemoveReg(); err != nil {
			fmt.Println(err)
		}
		return
	},
}

var registryAddCmd = &cobra.Command{
	Use:     "registry add",
	Example: "dtools2 registry add REGISTRY_URL [-c comments] [-u username] [-p password]",
	Short:   "Add a default registry entry",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		p := ""
		if registry.RegEntryPassword != "" {
			p = hf.EncodeString(registry.RegEntryPassword, "")
		}
		re := registry.RegistryEntry{RegistryName: args[0], Comments: registry.RegEntryComment,
			Username:      registry.RegEntryUsername,
			EncodedPasswd: p}

		if err := re.AddReg(); err != nil {
			fmt.Println(err)
		}
		return
	},
}

func init() {
	rootCmd.AddCommand(registryCmd)
	registryCmd.AddCommand(registryRemoveCmd, registryAddCmd)

	registryCmd.Flags().StringVarP(&registry.RegConfigFile, "registryfile", "r", "", "registry config file")
	registryAddCmd.Flags().StringVarP(&registry.RegEntryComment, "comment", "c", "", "registry entry comments")
	registryAddCmd.Flags().StringVarP(&registry.RegEntryUsername, "user", "u", "", "registry entry username --> currently unused")
	registryAddCmd.Flags().StringVarP(&registry.RegEntryPassword, "passwd", "p", "", "registry entry (encoded) password --> currently unused")

}
