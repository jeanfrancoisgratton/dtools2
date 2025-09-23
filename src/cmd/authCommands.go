// dtools2
// src/cmd/authCommands.go
// CLI glue for registry authentication (wires to auth subpackage)

package cmd

import (
	"context"
	"fmt"
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
		ctx := context.Background()
		mode, key, err := auth.CentralizedLogin(ctx, auth.LoginOptions{
			Registry:           loginRegistry, // accept host[:port] or full URL
			Username:           loginUsername,
			Password:           loginPassword,
			AllowHTTP:          loginAllowHTTP,
			CAFile:             "", // wire your flags if/when you add them
			ClientCertFile:     "",
			ClientKeyFile:      "",
			InsecureSkipVerify: loginTLSSkipVerify,
			Timeout:            15 * time.Second,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Login successful (%s). Credentials stored for %s\n", mode, key)
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
		ok, err := auth.Logout(logoutRegistry)
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
	authLoginCmd.Flags().StringVarP(&loginUsername, "username", "", "", "Registry username")
	authLoginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Registry password / PAT (can be empty)")
	authLoginCmd.Flags().BoolVarP(&loginAllowHTTP, "allow-http", "u", false, "Allow HTTP when no scheme specified (dangerous on untrusted networks)")
	authLoginCmd.Flags().BoolVarP(&loginTLSSkipVerify, "tls-skip-verify", "s", false, "Skip TLS certificate verification (dangerous; for lab/self-signed)")

	// auth logout
	authCmd.AddCommand(authLogoutCmd)
	authLogoutCmd.Flags().StringVarP(&logoutRegistry, "registry", "r", "", "Registry hostname[:port] or full URL")

	// auth whoami
	authCmd.AddCommand(authWhoAmICmd)
	authWhoAmICmd.Flags().StringVarP(&whoamiRegistry, "registry", "r", "", "Registry hostname[:port] or full URL")
}
