// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 22:01
// Original filename: src/cmd/imagesCommands.go

package cmd

import (
	"dtools2/env"
	"dtools2/rest"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	hf "github.com/jeanfrancoisgratton/helperFunctions/v4"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:     "env",
	Aliases: []string{"environment"},
	Short:   "Manage default registry handling",
	//Long:  "Manage docker/podman images via the Docker/Podman API (pull, list, etc.).",
}

var envRemoveCmd = &cobra.Command{
	Use:     "remove",
	Example: "dtools2 env remove",
	Aliases: []string{"rm"},
	Short:   "Remove the default registry entry and leave a blank entry instead",
	Run: func(cmd *cobra.Command, args []string) {

		if env.RegConfigFile == "" {
			env.RegConfigFile = filepath.Join(os.Getenv("HOME"), ".config", "JFG", "dtools2", "defaultRegistry.json")
		}
		re := env.RegistryEntry{}
		if err := re.RemoveReg(); err != nil {
			fmt.Println(err)
			return
		}
		if !rest.QuietOutput {
			fmt.Println(hftx.EnabledSign("Default registry removed"))
		}
	},
}

var envAddCmd = &cobra.Command{
	Use:     "add",
	Example: "dtools2 env add REGISTRY_URL [-c comments] [-u username] [-p password]",
	Short:   "Add a default registry entry",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		if env.RegConfigFile == "" {
			env.RegConfigFile = filepath.Join(os.Getenv("HOME"), ".config", "JFG", "dtools2", "defaultRegistry.json")
		}
		p := ""
		if env.RegEntryPassword != "" {
			p = hf.EncodeString(env.RegEntryPassword, "")
		}
		re := env.RegistryEntry{RegistryName: strings.TrimSuffix(args[0], "/"),
			Comments: env.RegEntryComment, Username: env.RegEntryUsername, EncodedPasswd: p}

		if err := re.AddReg(); err != nil {
			fmt.Println(err)
			return
		}
		if !rest.QuietOutput {
			fmt.Println(hftx.EnabledSign("Default registry set to " + args[0] + " in " + env.RegConfigFile))
		}
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.AddCommand(envRemoveCmd, envAddCmd)
	envRemoveCmd.Flags().StringVarP(&env.RegConfigFile, "registryfile", "r", "", "registry config file")
	envAddCmd.Flags().StringVarP(&env.RegConfigFile, "registryfile", "r", "", "registry config file")
	envAddCmd.Flags().StringVarP(&env.RegEntryComment, "comment", "c", "", "registry entry comments")
	envAddCmd.Flags().StringVarP(&env.RegEntryUsername, "user", "u", "", "registry entry username --> currently unused")
	envAddCmd.Flags().StringVarP(&env.RegEntryPassword, "passwd", "p", "", "registry entry (encoded) password --> currently unused")

}
