// dtools2
// src/cmd/root.go

package cmd

import (
	"dtools2/extras"
	"dtools2/rest"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "dtools2",
	SilenceUsage: true,
	Short:        "Docker / Podman client",
	Version:      "0.90.00 (2026.01.03)",
	Long: `dtools2 is a lightweight Docker/Podman client that talks directly
to the daemon's REST API (local Unix socket or remote TCP, with optional TLS).`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if restClient != nil {
			return
		}

		cfg := rest.Config{
			Host:               ConnectURI,
			APIVersion:         APIVersion,
			UseTLS:             UseTLS,
			CACertPath:         TLSCACert,
			CertPath:           TLSCert,
			KeyPath:            TLSKey,
			InsecureSkipVerify: TLSSkipVerify,
		}

		client, err := rest.NewClient(cfg)
		if err != nil {
			fmt.Println("Failed to initialize the REST client: ", err.Error())
			return
		}

		// If user did not force an API version, negotiate it with /version.
		if APIVersion == "" {
			v, err := rest.NegotiateAPIVersion(cmd.Context(), client)
			if err != nil {
				fmt.Println("Failed to negotiate API version: ", err.Error())
				return
			}
			client.SetAPIVersion(v)
			if extras.Debug {
				fmt.Fprintf(os.Stderr, "Negotiated API version: v%s\n", v)
			}
		}

		restClient = client
		return
	},
}

var execCmd = &cobra.Command{
	Use:     "exec [flags] CONTAINER COMMAND [ARG...]",
	Short:   "Run a command in a running container",
	Example: "dtools2 exec -it mycontainer /bin/sh",
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Fprintln(os.Stderr, "REST client not initialized")
			os.Exit(1)
		}
		rest.Context = cmd.Context()

		container := args[0]
		command := args[1:]

		exitCode, cerr := extras.Run(restClient, container, command)
		if cerr != nil {
			fmt.Fprintln(os.Stderr, cerr)
			os.Exit(1)
		}
		os.Exit(exitCode)
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

	rootCmd.AddCommand(completionCmd, execCmd)

	// Global flags.
	rootCmd.PersistentFlags().BoolVarP(&extras.Debug, "debug", "D", false, "Enable debug output on stderr")
	rootCmd.PersistentFlags().BoolVarP(&rest.QuietOutput, "quiet", "q", false, "Quiet output")
	rootCmd.PersistentFlags().StringVarP(&ConnectURI, "host", "H", "", "Docker daemon host (e.g. unix:///var/run/docker.sock, tcp://host:2376)")
	rootCmd.PersistentFlags().StringVarP(&APIVersion, "api-version", "A", "", "Docker API version (e.g. 1.43); if empty, auto-negotiate with the daemon")
	rootCmd.PersistentFlags().BoolVarP(&UseTLS, "tls", "T", false, "Use TLS when connecting to the daemon (for tcp:// hosts)")

	execCmd.Flags().BoolVarP(&extras.Interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
	execCmd.Flags().BoolVarP(&extras.AllocateTTY, "tty", "t", false, "Allocate a pseudo-TTY")
	execCmd.Flags().StringVarP(&extras.User, "user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
}
