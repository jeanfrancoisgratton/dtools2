// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/28 13:10
// Original filename: src/cmd/authCommands.go

package cmd

import (
	"context"
	"dtools2/auth"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	registry string
	username string
	token    string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage registry credentials in ~/.docker/config.json",
}

var authPingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Ping the Docker daemon",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Ping(ctx); err != nil {
			return err
		}
		fmt.Println("OK")
		return nil
	},
}

var authVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show daemon /version",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		resp, err := client.Do(ctx, "GET", []string{"version"}, nil, nil)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var m map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
			return err
		}
		out, _ := json.MarshalIndent(m, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

var authLoginCmd = &cobra.Command{
	Use:   "login USERNAME [PASSWORD]",
	Short: "Store username/password for a registry",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		username := args[0]
		var password string
		if len(args) == 2 {
			password = args[1]
		} else {
			fmt.Printf("Password: ")
			b, err := term.ReadPassword(0)
			fmt.Println()
			if err != nil {
				return err
			}
			password = string(b)
		}

		cfg, err := auth.Load()
		if err != nil {
			return err
		}
		// auth.NormalizeRegistry("") => Docker Hub default
		auth.SetUserPass(cfg, auth.NormalizeRegistry(registry), username, password)
		if err := auth.Save(cfg); err != nil {
			return err
		}
		fmt.Println("Saved credentials")
		return nil
	},
}

// token: positional args only.
// Usage: dtools2 auth token REGISTRY TOKEN
var authTokenCmd = &cobra.Command{
	Use:   "token REGISTRY TOKEN",
	Short: "Store identity token for a registry",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		registry := args[0]
		tok := args[1]

		cfg, err := auth.Load()
		if err != nil {
			return err
		}
		auth.SetToken(cfg, registry, tok)
		if err := auth.Save(cfg); err != nil {
			return err
		}
		fmt.Println("Saved token")
		return nil
	},
}

// logout: positional arg optional.
// Usage: dtools2 auth logout [REGISTRY]
var authLogoutCmd = &cobra.Command{
	Use:   "logout [REGISTRY]",
	Short: "Remove credentials for a registry",
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		registry := ""
		if len(args) == 1 {
			registry = args[0]
		}
		cfg, err := auth.Load()
		if err != nil {
			return err
		}
		if auth.Logout(cfg, registry) {
			if err := auth.Save(cfg); err != nil {
				return err
			}
			fmt.Println("Removed")
			return nil
		}
		fmt.Println("No entry")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authLogoutCmd, authTokenCmd, authPingCmd, authVersionCmd)
	authLoginCmd.Flags().StringVarP(&registry, "registry", "s", "", "Registry server (default: Docker Hub)")
	authLoginCmd.Flags().StringVarP(&username, "username", "u", "", "Registry username")
	authLoginCmd.Flags().StringVarP(&token, "token", "t", "", "Registry identity token (skips username/password)")
	authLogoutCmd.Flags().StringVarP(&registry, "registry", "s", "", "Registry server (default: Docker Hub)")
}
