// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/11/18 21:08
// Original filename: src/cmd/loginCommands.go

package cmd

import (
	"dtools2/auth"
	"dtools2/extras"
	"dtools2/rest"
	"fmt"
	"os"

	hf "github.com/jeanfrancoisgratton/helperFunctions/v4"
	hftx "github.com/jeanfrancoisgratton/helperFunctions/v4/terminalfx"
	"github.com/spf13/cobra"
)

// loginCmd implements `dtools2 auth login`, wiring through to auth.Login().
var loginCmd = &cobra.Command{
	Use:   "login REGISTRY",
	Short: "Log in to a container registry",
	Long: `Log in to a container registry and update ~/.docker/config.json.
This is similar to 'docker login': credentials are verified against the
registry's /v2/ endpoint and, on success, stored in the Docker config file.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		registry := args[0]

		if loginUsername == "" {
			if l, e := auth.GetUser(); e == nil {
				loginUsername = l
			} else {
				fmt.Println(e.Error())
				os.Exit(1)
			}
		}
		if loginPassword == "" {
			loginPassword = hf.GetPassword("Please enter the password: ", extras.Debug)
		}
		// check again, to ensure that the call to GetPassword() did return something
		if loginPassword == "" {
			fmt.Println("There is no password to login.")
			os.Exit(1)
		}

		opts := auth.LoginOptions{
			Registry:   registry,
			Username:   loginUsername,
			Password:   loginPassword,
			Insecure:   loginInsecure,
			CACertPath: loginCACertPath,
			// Timeout left as zero => default inside auth.Login
		}
		rest.Context = cmd.Context()
		if err := auth.Login(opts); err != nil {
			fmt.Println("Login failed: ", err.Error())
			os.Exit(1)
		}

		fmt.Println(hftx.EnabledSign(fmt.Sprintf("Loggin succeeded on registry %s", registry)))
		return
	},
}

func init() {
	// Attach as `dtools2 auth login`.
	rootCmd.AddCommand(loginCmd)

	loginCmd.Flags().StringVarP(&loginUsername, "username", "u", "", "Username for the registry")
	loginCmd.Flags().StringVarP(&loginPassword, "password", "p", "", "Password for the registry")
	loginCmd.Flags().BoolVar(&loginInsecure, "insecure", false, "Do not verify TLS certificate when using HTTPS")
	loginCmd.Flags().StringVar(&loginCACertPath, "ca-cert", "", "Path to a custom CA certificate file for the registry")
}
