// dtools2
// src/cmd/authCommands.go
// CLI glue for registry authentication (wires to auth subpackage)

package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"time"

	"dtools2/auth"
	"github.com/spf13/cobra"
)

var (
	// Shared flags (login)
	loginRegistry      string
	loginUsername      string
	loginPassword      string
	loginAllowHTTP     bool
	loginTLSSkipVerify bool

	// logout flags
	logoutRegistry string

	// whoami flags
	whoamiRegistry string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication helpers (login, logout, whoami)",
	// Avoid running anything when called without a subcommand.
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

// --- auth login ---

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate to a Docker/OCI registry and store credentials in config.json",
	Example: `  dtools2 auth login -r myreg:3281 -u bob -p q1w2e3
  dtools2 auth login -r http://myreg:3281 -u bob -p q1w2e3         # force HTTP explicitly
  dtools2 auth login -r myreg:3281 -u bob -p q1w2e3 --allow-http   # no scheme -> choose HTTP
  dtools2 auth login -r myreg:3281 -u bob -p q1w2e3 --tls-skip-verify`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginRegistry == "" {
			return fmt.Errorf("--registry/-r is required")
		}
		if loginUsername == "" {
			return fmt.Errorf("--username/-u is required")
		}

		// Decide scheme:
		reg := loginRegistry
		if !strings.Contains(reg, "://") {
			if loginAllowHTTP {
				reg = "http://" + reg
			} else {
				reg = "https://" + reg
			}
		}

		// HTTP client: timeout + optional TLS skip verify
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: loginTLSSkipVerify}, //nolint:gosec // intentionally user-controlled
		}
		client := &http.Client{
			Timeout:   15 * time.Second,
			Transport: transport,
		}

		ctx := context.Background()
		if err := auth.LoginAndStoreSmartWithClient(ctx, client, reg, loginUsername, loginPassword); err != nil {
			return err
		}

		fmt.Printf("Login successful. Credentials stored for %s\n", auth.NormalizeRegistry(reg))
		return nil
	},
}

// --- auth logout ---

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials for a registry from config.json",
	Example: `  dtools2 auth logout -r myreg:3281
  dtools2 auth logout -r https://index.docker.io/v1/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if logoutRegistry == "" {
			return fmt.Errorf("--registry/-r is required")
		}
		ok, err := auth.RemoveDockerConfigAuth(logoutRegistry)
		if err != nil {
			return err
		}
		if ok {
			fmt.Printf("Removed credentials for %s\n", auth.NormalizeRegistry(logoutRegistry))
		} else {
			fmt.Printf("No credentials found for %s\n", auth.NormalizeRegistry(logoutRegistry))
		}
		return nil
	},
}

// --- auth whoami ---

var authWhoAmICmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show how you are authenticated to a registry (basic/token/helper/missing)",
	Example: `  dtools2 auth whoami -r myreg:3281
  dtools2 auth whoami -r docker.io`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if whoamiRegistry == "" {
			return fmt.Errorf("--registry/-r is required")
		}
		info, err := auth.WhoAmI(whoamiRegistry)
		if err != nil {
			return err
		}

		switch info.Mode {
		case "basic":
			// Do not print the password; just show username.
			fmt.Printf("[%s] mode=basic user=%s\n", info.Registry, info.Username)
		case "token":
			// Only show a short, non-sensitive preview.
			fmt.Printf("[%s] mode=token token=%s\n", info.Registry, info.TokenPreview)
		case "helper":
			fmt.Printf("[%s] mode=helper (credential helper configured)\n", info.Registry)
		case "missing":
			fmt.Printf("[%s] mode=missing (no stored credentials)\n", info.Registry)
		default:
			fmt.Printf("[%s] mode=unknown\n", info.Registry)
		}
		return nil
	},
}

func init() {
	// Attach `auth` under root.
	rootCmd.AddCommand(authCmd)

	// auth login
	authCmd.AddCommand(authLoginCmd)
	authLoginCmd.Flags().StringVarP(&loginRegistry, "registry", "r", "", "Registry hostname[:port] or full URL (e.g., myreg:3281 or https://myreg:3281)")
	authLoginCmd.Flags().StringVarP(&loginUsername, "username", "u", "", "Registry username")
	authLoginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Registry password / PAT (can be empty)")
	authLoginCmd.Flags().BoolVar(&loginAllowHTTP, "allow-http", false, "Allow HTTP when no scheme specified (dangerous on untrusted networks)")
	authLoginCmd.Flags().BoolVar(&loginTLSSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification (dangerous; for lab/self-signed)")

	// auth logout
	authCmd.AddCommand(authLogoutCmd)
	authLogoutCmd.Flags().StringVarP(&logoutRegistry, "registry", "r", "", "Registry hostname[:port] or full URL")

	// auth whoami
	authCmd.AddCommand(authWhoAmICmd)
	authWhoAmICmd.Flags().StringVarP(&whoamiRegistry, "registry", "r", "", "Registry hostname[:port] or full URL")
}
