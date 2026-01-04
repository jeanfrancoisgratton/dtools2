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

	ce "github.com/jeanfrancoisgratton/customError/v3"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
	"github.com/spf13/cobra"
)

// blacklistCmd  groups resources blacklist-related subcommands.
var blCmd = &cobra.Command{
	Use:     "blacklist",
	Aliases: []string{"bl"},
	Example: "dtools blacklist {list | add | rm}",
	Short:   "Resource blacklist management",
	Long: "A resource blacklist is a way to protect from removal a specific resource, " +
		"such as volume, network, container or image.",
}

var blListCmd = &cobra.Command{
	Use:     "lsb [flags]",
	Example: "dtools blacklist lsb [{ -a | { volume | network | container | image } }]",
	Short:   "Lists the black listed resources",
	Run: func(cmd *cobra.Command, args []string) {
		var err *ce.CustomError
		if blacklist.AllBlackLists {
			//return blacklist.ListAllFromFile()
			err = blacklist.ListAllFromFile()
		} else {
			if len(args) > 0 {
				a := strings.ToLower(args[0])
				if !slices.Contains(blacklist.ResourceNamesList, a) {
					fmt.Println("Resource name not recognized")
				} else {
					//return blacklist.ListFromFile(args[0])
					err = blacklist.ListFromFile(strings.ToLower(args[0]))
				}
			} else {
				fmt.Println(hftx.WarningSign(" resourceType is empty"))
				return
			}
		}
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	},
}

var blAddCmd = &cobra.Command{
	Use:     "add RESOURCE_TYPE RESOURCE_NAME",
	Example: "dtools blacklist add resource_type resource_name1 [resource_name2..resource_nameN]",
	Short:   "Add one or more resource_name to resource_type",
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := blacklist.AddResource(args[0], args[1:]); err != nil {
			fmt.Println(err.Error())
			return
		}
	},
}

var blRemoveCmd = &cobra.Command{
	Use:     "rmb RESOURCE_TYPE RESOURCE_NAME",
	Example: "dtools blacklist rmb resource_type resource_name1 [resource_name2..resource_nameN]",
	Short:   "Remove one or more resource from resource_type",
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if _, err := blacklist.DeleteResource(args[0], args[1:]); err != nil {
			fmt.Println(err.Error())
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(blCmd)
	blCmd.AddCommand(blListCmd, blAddCmd, blRemoveCmd)

	blListCmd.Flags().BoolVarP(&blacklist.AllBlackLists, "all", "a", false, "List all resources")
}
