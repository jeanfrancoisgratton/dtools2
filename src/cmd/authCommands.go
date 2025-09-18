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
	loginRegistry      string
	loginUsername      string
	loginPassword      string
	loginAllowHTTP     bool
	loginTLSSkipVerify bool
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication helpers (login, etc.)",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate to a Docker/OCI registry and store credentials in config.json",
	Example: `  dtools2 auth login -r myreg:3281 -u bob -p q1w2e3
  dtools2 auth login -r http://myreg:3281 -u bob -p q1w2e3         # force HTTP explicitly
  dtools2 auth login -r myreg:3281 -u bob -p q1w2e3 --allow-http   # no scheme provided -> choose HTTP
  dtools2 auth login -r myreg:3281 -u bob -p q1w2e3 --tls-skip-verify`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginRegistry == "" {
			return fmt.Errorf("--registry/-r is required")
		}
		if loginUsername == "" {
			return fmt.Errorf("--username/-u is required")
		}
		// NOTE: empty password can be valid (PAT-as-password or token exchange)

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

func init() {
	// Attach `auth` root under main root, and `login` under `auth`.
	authCmd.AddCommand(authLoginCmd)

	authLoginCmd.Flags().StringVarP(&loginRegistry, "registry", "r", "", "Registry hostname[:port] or full URL (e.g., myreg:3281 or https://myreg:3281)")
	authLoginCmd.Flags().StringVarP(&loginUsername, "username", "u", "", "Registry username")
	authLoginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Registry password / PAT (can be empty)")
	authLoginCmd.Flags().BoolVar(&loginAllowHTTP, "allow-http", false, "Allow HTTP when no scheme specified (dangerous on untrusted networks)")
	authLoginCmd.Flags().BoolVar(&loginTLSSkipVerify, "tls-skip-verify", false, "Skip TLS certificate verification (dangerous; for lab/self-signed)")
}
