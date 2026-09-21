// dtools2
// src/cmd/root.go

package cmd

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"dtools2/backend"
	"dtools2/backend/containerdbackend"
	"dtools2/backend/restbackend"
	"dtools2/extras"
	"dtools2/rest"

	hftx "github.com/jeanfrancoisgratton/helperFunctions/v5/terminalfx"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "dtools",
	SilenceUsage: true,
	Short:        "Docker / Podman client",
	Long: `dtools is a lightweight Docker/Podman client that talks directly
to the daemon's REST API (local Unix socket or remote TCP, with optional TLS).`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if activeBackend != nil {
			return
		}

		switch Runtime {
		case "", "docker", "podman", "containerd":
			// valid
		default:
			fmt.Printf("Unknown --runtime %q: must be one of \"\" (auto), \"docker\", \"podman\", \"containerd\"\n", Runtime)
			return
		}

		if Runtime == "containerd" && cmd.Flags().Changed("host") {
			fmt.Println("--host/-H is not supported with --runtime=containerd: containerd is local-only")
			return
		}

		// Auto-detect only makes sense when the user hasn't pinned down where
		// to connect: an explicit -H means "talk to this REST endpoint", and
		// falling back to a local containerd if that's unreachable would
		// silently switch to an unrelated daemon instead of surfacing the
		// real connection error.
		autoDetect := Runtime == "" && !cmd.Flags().Changed("host")

		var (
			b      backend.Backend
			err    error
			picked string
		)

		switch {
		case Runtime == "containerd":
			b, err = containerdbackend.New(cmd.Context(), ContainerdSocket, ContainerdNamespace, AllNamespaces)

		case autoDetect:
			// Probe with a short, throwaway client first: FastFailTimeout also
			// becomes the transport's ResponseHeaderTimeout for every request
			// a client ever makes, not just this reachability check. Reusing
			// the probe's client for the whole session would leave every
			// later request (pull, delete, ...) with only a few seconds to
			// receive response headers. So on a successful probe we build a
			// fresh client with the normal (possibly user-configured) timeout
			// instead of keeping the probe's client around.
			probeCfg := restConfigFromFlags()
			if !cmd.Flags().Changed("fast-fail") {
				probeCfg.FastFailTimeout = autoDetectProbeTimeout
			}
			if _, probeErr := restbackend.New(cmd.Context(), probeCfg); probeErr == nil {
				b, err = restbackend.New(cmd.Context(), restConfigFromFlags())
			} else {
				var cdErr error
				b, cdErr = containerdbackend.New(cmd.Context(), ContainerdSocket, ContainerdNamespace, AllNamespaces)
				if cdErr != nil {
					err = fmt.Errorf("no container runtime found:\n  docker/podman: %s\n  containerd: %s", probeErr, cdErr)
				} else {
					picked = " (docker/podman unreachable, fell back to containerd)"
				}
			}

		default: // Runtime == "docker" or "podman"
			b, err = restbackend.New(cmd.Context(), restConfigFromFlags())
		}

		if err != nil {
			fmt.Println("Failed to initialize the backend: ", err.Error())
			return
		}
		if autoDetect && !rest.QuietOutput {
			fmt.Fprintf(os.Stderr, "Using backend: %s%s\n", b.Name(), picked)
		} else if extras.Debug {
			fmt.Printf("Using backend: %s\n", b.Name())
		}
		activeBackend = b
		return
	},
}

const autoDetectProbeTimeout = 3 * time.Second

func restConfigFromFlags() rest.Config {
	return rest.Config{
		Host:               rest.ConnectURI,
		APIVersion:         APIVersion,
		UseTLS:             UseTLS,
		CACertPath:         TLSCACert,
		CertPath:           TLSCert,
		KeyPath:            TLSKey,
		InsecureSkipVerify: TLSSkipVerify,
		FastFailTimeout:    time.Duration(rest.FastFailTimeoutSeconds) * time.Second,
		SessionTimeout:     time.Duration(rest.SessionTimeoutMinutes) * time.Minute,
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Shows the software version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(hftx.White("dtools v" + buildVersion + " (" + buildDate + "), Go version = v" + strings.TrimPrefix(runtime.Version(), "go") + " (" + runtime.GOARCH + ")"))
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

	rootCmd.AddCommand(completionCmd, blListCmd, versionCmd)

	// Global flags.
	rootCmd.PersistentFlags().BoolVarP(&extras.Debug, "debug", "D", false, "Enable debug output on stderr")
	rootCmd.PersistentFlags().BoolVar(&extras.OutputJSON, "json", false, "Output JSON instead of formatted tables")
	rootCmd.PersistentFlags().BoolVarP(&rest.QuietOutput, "quiet", "q", false, "Quiet output")
	rootCmd.PersistentFlags().StringVarP(&rest.ConnectURI, "host", "H", "", "Docker daemon host (e.g. unix:///var/run_build/docker.sock, tcp://host:2376)")
	rootCmd.PersistentFlags().StringVarP(&APIVersion, "api-version", "V", "", "Docker API version (e.g. 1.43); if empty, auto-negotiate with the daemon")
	rootCmd.PersistentFlags().BoolVarP(&UseTLS, "tls", "T", false, "Use TLS when connecting to the daemon (for tcp:// hosts)")
	rootCmd.PersistentFlags().IntVar(&rest.FastFailTimeoutSeconds, "fast-fail", rest.FastFailTimeoutSeconds, "HTTP fast-fail timeout in seconds (dial/TLS handshake/headers)")
	rootCmd.PersistentFlags().IntVar(&rest.SessionTimeoutMinutes, "session-timeout", rest.SessionTimeoutMinutes, "HTTP session timeout in minutes for finite long operations (pull/build/cp/save/load); 0 disables")
	rootCmd.PersistentFlags().StringVar(&Runtime, "runtime", "", "Container runtime backend to use: \"\", \"docker\" or \"podman\" (REST API) or \"containerd\"")
	rootCmd.PersistentFlags().StringVar(&ContainerdSocket, "containerd-socket", containerdbackend.DefaultSocket, "containerd gRPC socket path (only used with --runtime=containerd)")
	rootCmd.PersistentFlags().StringVarP(&ContainerdNamespace, "namespace", "N", containerdbackend.DefaultNamespace, "containerd namespace to operate in (only used with --runtime=containerd)")
	rootCmd.PersistentFlags().BoolVarP(&AllNamespaces, "all-namespaces", "A", false, "containerd: list across all namespaces instead of just --namespace (list operations only; lifecycle/prune stay scoped to --namespace)")

}
