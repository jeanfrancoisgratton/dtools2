// dtools2
// src/cmd/root.go

package cmd

import (
	"dtools2/extras"
	"dtools2/rest"
	"dtools2/system"
	"fmt"
	"os"
	"runtime"
	"strings"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "dtools",
	SilenceUsage: true,
	Short:        "Docker / Podman client",
	Version:      "feature/2.30.00 (2026.01.11), Go version = " + runtime.Version(),
	Long: `dtools is a lightweight Docker/Podman client that talks directly
to the daemon's REST API (local Unix socket or remote TCP, with optional TLS).`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if restClient != nil {
			return
		}

		cfg := rest.Config{
			Host:               rest.ConnectURI,
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
				fmt.Printf("Negotiated API version: v%s\n", v)
			}
		}
		restClient = client
		return
	},
}

var copyCmd = &cobra.Command{
	Use:     "cp",
	Aliases: []string{"copy"},
	Short:   "Copy a file to or from a container",
	Example: "dtools cp { container_name:path host_path | host_path container:path }",
	Args:    cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		nContainers := 0
		if strings.Contains(args[0], ":") {
			nContainers++
		}
		if strings.Contains(args[1], ":") {
			nContainers++
		}

		if nContainers == 0 || nContainers == 2 {
			fmt.Println(hftx.ErrorSign("You must specify a container name:path in one of the arguments"))
			return
		}

		rest.Context = cmd.Context()
		if err := system.CopyFile(restClient, args[0], args[1]); err != nil {
			fmt.Println(err)
		}
		return
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

	rootCmd.AddCommand(copyCmd, completionCmd)

	// Override Cobra's default version shorthand (-v) to free it for future use.
	// Cobra will not register its own version flag if it already exists.
	rootCmd.Flags().BoolP("version", "V", false, "Show version and exit")

	// Global flags.
	rootCmd.PersistentFlags().BoolVarP(&extras.Debug, "debug", "D", false, "Enable debug output on stderr")
	rootCmd.PersistentFlags().BoolVar(&extras.OutputJSON, "json", false, "Output JSON instead of formatted tables")
	rootCmd.PersistentFlags().BoolVarP(&rest.QuietOutput, "quiet", "q", false, "Quiet output")
	rootCmd.PersistentFlags().StringVarP(&rest.ConnectURI, "host", "H", "", "Docker daemon host (e.g. unix:///var/run/docker.sock, tcp://host:2376)")
	rootCmd.PersistentFlags().StringVarP(&APIVersion, "api-version", "A", "", "Docker API version (e.g. 1.43); if empty, auto-negotiate with the daemon")
	rootCmd.PersistentFlags().BoolVarP(&UseTLS, "tls", "T", false, "Use TLS when connecting to the daemon (for tcp:// hosts)")

}
