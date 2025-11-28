// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/26 01:22
// Original filename: src/cmd/blacklistCommands.go

package cmd

import (
	"dtools2/blacklist"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

// blacklistCmd  groups resources blacklist-related subcommands.
var blCmd = &cobra.Command{
	Use:     "blacklist",
	Aliases: []string{"bl"},
	Example: "dtools2 blacklist {list | add | rm}",
	Short:   "Resource blacklist management",
	Long: "A resource blacklist is a way to protect from removal a specific resource, " +
		"such as volume, network, container or image.",
}

var blListCmd = &cobra.Command{
	Use:     "ls [flags]",
	Example: "dtools2 blacklist ls [{ -a | { volume | network | container | image } }]",
	Short:   "Lists the black listed resources",
	Run: func(cmd *cobra.Command, args []string) {
		//var err error
		if blacklist.AllBlackLists {
			//return blacklist.ListAllFromFile()
			blacklist.ListAllFromFile()
		} else {
			if len(args) > 0 {
				a := strings.ToLower(args[0])
				if !slices.Contains(blacklist.ResourceNamesList, a) {
					fmt.Println("Resource name not recognized")
				} else {
					//return blacklist.ListFromFile(args[0])
					blacklist.ListFromFile(strings.ToLower(args[0]))
				}
			}
		}
	},
}

var blAddCmd = &cobra.Command{
	Use:     "add resource_name resource1..resourceN",
	Example: "dtools2 blacklist add resource_name resource...",
	Short:   "Add one or more resource to resource_name",
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		blacklist.AddResource(args[0], args[1:])
	},
}

func init() {
	rootCmd.AddCommand(blCmd)
	blCmd.AddCommand(blListCmd, blAddCmd)

	blListCmd.Flags().BoolVarP(&blacklist.AllBlackLists, "all", "a", false, "List all resources")
	//containersCmd.AddCommand(containersListCmd)
	//
	//containersListCmd.Flags().BoolVarP(&containers.OnlyRunningContainers, "running", "r", false, "List only the running containers")
	//containersListCmd.Flags().BoolVarP(&containers.ExtendedContainerInfo, "extended", "x", false, "Show extended container info")
}
