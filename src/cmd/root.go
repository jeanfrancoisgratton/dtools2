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
	Use:          "dtools",
	SilenceUsage: true,
	Short:        "Docker / Podman client",
	Version:      "2.00.00 (2026.01.03)",
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
	Example: "dtools exec -it mycontainer /bin/sh",
	Args:    cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			os.Exit(1)
		}
		rest.Context = cmd.Context()

		container := args[0]
		command := args[1:]

		exitCode, cerr := extras.Run(restClient, container, command)
		if cerr != nil {
			fmt.Println(cerr)
			os.Exit(1)
		}
		os.Exit(exitCode)
	},
}

var logsCmd = &cobra.Command{
	Use:     "logs [flags] CONTAINER",
	Aliases: []string{"log"},
	Short:   "Fetch the logs of a container",
	Example: "dtools logs -t -n 200 -f mycontainer",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()

		if cerr := extras.Logs(restClient, args[0]); cerr != nil {
			fmt.Println(cerr)
			return
		}
	},
}

var runCmd = &cobra.Command{
	Use:     "run [flags] IMAGE [COMMAND] [ARG...]",
	Short:   "Run a command in a new container",
	Example: "dtools run -it --rm alpine:latest /bin/sh",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if restClient == nil {
			fmt.Println("REST client not initialized")
			return
		}
		rest.Context = cmd.Context()

		image := args[0]
		command := []string{}
		if len(args) > 1 {
			command = args[1:]
		}

		_, id, cerr := extras.RunContainer(restClient, image, command)
		if cerr != nil {
			fmt.Println(cerr)
			return
		}

		if extras.RunDetach {
			if id != "" {
				fmt.Println(id)
			}
			return
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

	rootCmd.AddCommand(completionCmd, execCmd, logsCmd, runCmd)

	// Global flags.
	rootCmd.PersistentFlags().BoolVarP(&extras.Debug, "debug", "D", false, "Enable debug output on stderr")
	rootCmd.PersistentFlags().BoolVarP(&rest.QuietOutput, "quiet", "q", false, "Quiet output")
	rootCmd.PersistentFlags().StringVarP(&ConnectURI, "host", "H", "", "Docker daemon host (e.g. unix:///var/run/docker.sock, tcp://host:2376)")
	rootCmd.PersistentFlags().StringVarP(&APIVersion, "api-version", "A", "", "Docker API version (e.g. 1.43); if empty, auto-negotiate with the daemon")
	rootCmd.PersistentFlags().BoolVarP(&UseTLS, "tls", "T", false, "Use TLS when connecting to the daemon (for tcp:// hosts)")
	execCmd.Flags().BoolVarP(&extras.Interactive, "interactive", "i", false, "Keep STDIN open even if not attached")
	execCmd.Flags().BoolVarP(&extras.AllocateTTY, "tty", "t", false, "Allocate a pseudo-TTY")
	execCmd.Flags().StringVarP(&extras.User, "user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	logsCmd.Flags().BoolVarP(&extras.LogTimestamps, "timestamps", "t", false, "Show timestamps")
	logsCmd.Flags().IntVarP(&extras.LogTail, "tail", "n", -1, "Number of lines to show from the end of the logs (-1 means all)")
	logsCmd.Flags().BoolVarP(&extras.LogFollow, "follow", "f", false, "Follow log output")
	runCmd.Flags().BoolVarP(&extras.RunDetach, "detach", "d", false, "Run container in background and print container ID")
	runCmd.Flags().BoolVarP(&extras.RunInteractive, "interactive", "i", false, "Keep STDIN open even if not attached")
	runCmd.Flags().BoolVarP(&extras.RunTTY, "tty", "t", false, "Allocate a pseudo-TTY")
	runCmd.Flags().BoolVar(&extras.RunRemove, "rm", false, "Automatically remove the container when it exits")
	runCmd.Flags().StringVar(&extras.RunName, "name", "", "Assign a name to the container")
	runCmd.Flags().StringVarP(&extras.RunUser, "user", "u", "", "Username or UID (format: <name|uid>[:<group|gid>])")
	runCmd.Flags().StringVarP(&extras.RunWorkdir, "workdir", "w", "", "Working directory inside the container")
	runCmd.Flags().StringArrayVarP(&extras.RunEnv, "env", "e", nil, "Set environment variables")
	runCmd.Flags().StringArrayVarP(&extras.RunPublish, "publish", "p", nil, "Publish a container's port(s) to the host")
	runCmd.Flags().StringArrayVarP(&extras.RunVolume, "volume", "v", nil, "Bind mount a volume")
	runCmd.Flags().StringVar(&extras.RunNetwork, "network", "", "Connect a container to a network")
	runCmd.Flags().StringVar(&extras.RunEntrypoint, "entrypoint", "", "Overwrite the default ENTRYPOINT of the image")
	runCmd.Flags().StringVarP(&extras.RunHostname, "hostname", "h", "", "Container host name")
}
