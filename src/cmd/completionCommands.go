// dtools2
// Written by J.F. Gratton <jean-francois@famillegratton.net>
// Original timestamp: 2025/09/15 08:35
// Original filename: src/cmd/completionCommands.go
// Bash/Zsh completion scripts via Cobra.

package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion scripts",
	Long: `Generate completion scripts for your shell.

Bash:
  $ source <(dtools2 completion bash)
  # To persist:
  $ dtools2 completion bash | sudo tee /etc/bash_completion.d/dtools2 > /dev/null

Zsh:
  $ dtools2 completion zsh > ~/.zsh[.completion.d]/_dtools2
  $ echo 'fpath=($HOME/.zsh $fpath)' >> ~/.zshrc
  $ echo 'autoload -Uz compinit && compinit' >> ~/.zshrc
  # Or, for current session:
  $ source <(dtools2 completion zsh)
`,
}

var completionBashCmd = &cobra.Command{
	Use:   "bash",
	Short: "Generate a Bash completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		// V2 is recommended; writes to stdout
		return rootCmd.GenBashCompletionV2(os.Stdout, true)
	},
}

var completionZshCmd = &cobra.Command{
	Use:   "zsh",
	Short: "Generate a Zsh completion script",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure the script is zsh-compatible
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

func init() {
	completionCmd.AddCommand(completionBashCmd, completionZshCmd)
}
