// dtools2
// src/cmd/root.go

package cmd

import (
	"context"
	"dtools2/rest"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

var Debug = false
var ConnectURI string

var (
	hostFlag    string
	forceAPIVer string
	client      *rest.Client
)

var rootCmd = &cobra.Command{
	Use:          "dtools2",
	SilenceUsage: true,
	Short:        "Docker / Podman client",
	Version:      "1.00.00-0 (2025.09.16)",
	Long: `This software is intended to be a full drop-in replacemenat to the current docker and podman clients.
It relies on the REST APIs of both platforms as the SDK tend to change too much, and to frequently to ensure stability.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize REST client once
		if client != nil {
			return nil
		}
		c, err := rest.New(hostFlag, forceAPIVer)
		if err != nil {
			return err
		}
		// Negotiate early by ping to provide fast fail
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := c.Ping(ctx); err != nil {
			return err
		}
		client = c
		return nil
	},
}

// Shows changelog
var clCmd = &cobra.Command{
	Use:     "changelog",
	Aliases: []string{"cl"},
	Short:   "Shows the Changelog",
	Run: func(cmd *cobra.Command, args []string) {
		changeLog()
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.DisableAutoGenTag = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	rootCmd.AddCommand(clCmd, completionCmd)
	rootCmd.AddCommand(authPingCmd)
	rootCmd.AddCommand(authVersionCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.PersistentFlags().StringVarP(&ConnectURI, "host", "H", "unix:///var/run/docker.sock", "Remote host:port to connect to")
	rootCmd.PersistentFlags().BoolVarP(&Debug, "debug", "D", false, "Debug mode")
	rootCmd.PersistentFlags().StringVarP(&forceAPIVer, "force-api", "F", "", "Force Docker API version, e.g. v1.45 or 1.45 (disables negotiation)")

}

func changeLog() {
	//fmt.Printf("\x1b[2J")
	fmt.Printf("\x1bc")

	fmt.Println("CHANGELOG")
	fmt.Println("=========")

	fmt.Print(`
VERSION			DATE			COMMENT
-------			----			-------
0.10.00			2025.09.16		Initial release
`)
}
