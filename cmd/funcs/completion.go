package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

/*
newCompletionCmd creates the completion command for generating shell completion scripts.

The completion command generates shell completion scripts for various shells.
This enables tab-completion for funcs commands and funcs.

Supported shells:
  - bash
  - zsh
  - fish
  - powershell

Examples:

	# Generate bash completion
	funcs completion bash > /etc/bash_completion.d/funcs

	# Generate zsh completion
	funcs completion zsh > ~/.zsh/completion/_funcs

	# Generate fish completion
	funcs completion fish > ~/.config/fish/completions/funcs.fish

	# Generate powershell completion
	funcs completion powershell > funcs.ps1
*/
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long:  completionHelp(),
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCompletion(cmd, args)
		},
	}
	return cmd
}

func runCompletion(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("shell is required (bash, zsh, fish, powershell)")
	}
	shell := strings.ToLower(strings.TrimSpace(args[0]))
	switch shell {
	case "bash":
		return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
	case "zsh":
		return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
	case "fish":
		return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
	case "powershell":
		return cmd.Root().GenPowerShellCompletion(cmd.OutOrStdout())
	default:
		return fmt.Errorf("unsupported shell %q", shell)
	}
}

func completionHelp() string {
	return strings.Join([]string{
		"Examples:",
		"  funcs completion bash > /etc/bash_completion.d/funcs",
		"  funcs completion zsh > ~/.zsh/completion/_funcs",
		"  funcs completion fish > ~/.config/fish/completions/funcs.fish",
		"  funcs completion powershell > funcs.ps1",
	}, "\n")
}
