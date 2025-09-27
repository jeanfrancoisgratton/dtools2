// dtools2
// src/cmd/authCommands.go
// CLI glue for registry authentication (wires to auth subpackage)

package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"dtools2/auth"

	hf "github.com/jeanfrancoisgratton/helperFunctions/v3"
	hft "github.com/jeanfrancoisgratton/helperFunctions/v3/terminalfx"

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
	Use:   "login [REGISTRY] [USERNAME] [PASSWORD]",
	Short: "Authenticate to a Docker/OCI registry and store credentials in config.json",
	Args:  cobra.RangeArgs(2, 3), // require REGISTRY and USERNAME; PASSWORD optional
	Example: `  dtools2 auth login myreg:3281 bob q1w2e3
  dtools2 auth login http://myreg:3281 bob q1w2e3         # force HTTP explicitly
  dtools2 auth login myreg:3281 bob --allow-http          # no scheme -> choose HTTP; prompt for password
  dtools2 auth login myreg:3281 bob --tls-skip-verify     # prompt for password`,
	Run: func(cmd *cobra.Command, args []string) {
		// Positional args take precedence; fall back to flags if provided
		reg := loginRegistry
		user := loginUsername
		pass := loginPassword

		if len(args) >= 1 {
			reg = args[0]
		}
		if len(args) >= 2 {
			user = args[1]
		}
		if len(args) >= 3 {
			pass = args[2]
		}

		if reg == "" {
			fmt.Println("REGISTRY is required (positional arg 1)")
			os.Exit(1)
		}
		if user == "" {
			fmt.Println("USERNAME is required (positional arg 2)")
			os.Exit(1)
		}

		// If password still empty, prompt (plain text as requested)
		if pass == "" {
			pass = hf.GetPassword("Please enter the password: ", Debug)
			if pass == "" {
				fmt.Println("Password is required")
				os.Exit(1)
			}
		}

		// Note: CentralizedLogin handles allow-http + tls-skip-verify via options.
		ctx := context.Background()
		mode, key, err := auth.CentralizedLogin(ctx, auth.LoginOptions{
			Registry:           reg,
			Username:           user,
			Password:           pass,
			AllowHTTP:          loginAllowHTTP,
			InsecureSkipVerify: loginTLSSkipVerify,
			Timeout:            15 * time.Second,
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s (%s). %s %s\n", hft.Green("Login successful"), mode,
			hft.Green("Credentials stored for"), key)
		os.Exit(0)
	},
}

// --- auth logout ---

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove stored credentials for a registry from config.json",
	Example: `  dtools2 auth logout -r myreg:3281
  dtools2 auth logout -r https://index.docker.io/v1/`,
	Run: func(cmd *cobra.Command, args []string) {
		if logoutRegistry == "" {
			fmt.Println("--registry/-r is required")
			os.Exit(1)
		}
		ok, err := auth.Logout(logoutRegistry)
		if err != nil {
			fmt.Println(err.Error())
		}
		if ok {
			fmt.Printf("Removed credentials for %s\n", auth.NormalizeRegistry(logoutRegistry))
		} else {
			fmt.Printf("No credentials found for %s\n", auth.NormalizeRegistry(logoutRegistry))
		}
	},
}

// --- auth whoami ---

var authWhoAmICmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show how you are authenticated to a registry (basic/token/helper/missing)",
	Example: `  dtools2 auth whoami -r myreg:3281
  dtools2 auth whoami -r docker.io`,
	Run: func(cmd *cobra.Command, args []string) {
		if whoamiRegistry == "" {
			fmt.Println("--registry/-r is required")
			os.Exit(1)
		}
		info, err := auth.WhoAmI(whoamiRegistry)
		if err != nil {
			fmt.Println(err.Error())
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
	},
}

func init() {
	// auth login
	authCmd.AddCommand(authLoginCmd)
	authLoginCmd.Flags().StringVarP(&loginRegistry, "registry", "r", "", "Registry hostname[:port] or full URL (deprecated; use positional REGISTRY)")
	authLoginCmd.Flags().StringVarP(&loginUsername, "username", "u", "", "Registry username (deprecated; use positional USERNAME)")
	authLoginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Registry password / PAT (optional; will prompt if omitted)")
	authLoginCmd.Flags().BoolVar(&loginAllowHTTP, "allow-http", false, "Allow HTTP when no scheme specified")
	authLoginCmd.Flags().BoolVar(&loginTLSSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification")

	_ = authLoginCmd.Flags().MarkDeprecated("registry", "use positional REGISTRY instead")
	_ = authLoginCmd.Flags().MarkDeprecated("username", "use positional USERNAME instead")

	// auth logout
	authCmd.AddCommand(authLogoutCmd)
	authLogoutCmd.Flags().StringVarP(&logoutRegistry, "registry", "r", "", "Registry hostname[:port] or full URL")

	// auth whoami
	authCmd.AddCommand(authWhoAmICmd)
	authWhoAmICmd.Flags().StringVarP(&whoamiRegistry, "registry", "r", "", "Registry hostname[:port] or full URL")
}
