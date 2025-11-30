// dtools2
// src/cmd/root.go

package cmd

import (
	"dtools2/rest"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "dtools2",
	SilenceUsage: true,
	Short:        "Docker / Podman client",
	Version:      "0.20.00 (2025.11.24)",
	Long: `dtools2 is a lightweight Docker/Podman client that talks directly
to the daemon's REST API (local Unix socket or remote TCP, with optional TLS).`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if restClient != nil {
			return nil
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
			return fmt.Errorf("failed to initialize REST client: %w", err)
		}

		// If user did not force an API version, negotiate it with /version.
		if APIVersion == "" {
			v, err := rest.NegotiateAPIVersion(cmd.Context(), client)
			if err != nil {
				return fmt.Errorf("failed to negotiate API version: %w", err)
			}
			client.SetAPIVersion(v)
			if Debug {
				fmt.Fprintf(os.Stderr, "Negotiated API version: %s\n", v)
			}
		}

		restClient = client
		return nil
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

	rootCmd.AddCommand(completionCmd)

	// Global flags.
	rootCmd.PersistentFlags().BoolVarP(&Debug, "debug", "D", false, "Enable debug output on stderr")
	rootCmd.PersistentFlags().BoolVarP(&rest.QuietOutput, "quiet", "q", false, "Quiet output")
	rootCmd.PersistentFlags().StringVarP(&ConnectURI, "host", "H", "", "Docker daemon host (e.g. unix:///var/run/docker.sock, tcp://host:2376)")
	rootCmd.PersistentFlags().StringVarP(&APIVersion, "api-version", "A", "", "Docker API version (e.g. 1.43); if empty, auto-negotiate with the daemon")
	rootCmd.PersistentFlags().BoolVarP(&UseTLS, "tls", "t", false, "Use TLS when connecting to the Docker daemon (for tcp:// hosts)")
}
