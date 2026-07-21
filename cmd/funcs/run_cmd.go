package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/funcs/internal/alias"
	"github.com/funcs/internal/config"
	"github.com/funcs/internal/flags"
)

type runOptions struct {
	configPath string
}

func newRunCmd() *cobra.Command {
	opts := &runOptions{}
	cmd := &cobra.Command{
		Use:   "run <script_name> [args...]",
		Short: "Search for script name and execute it",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			cfg := config.Load(cwd, opts.configPath)

			aliasManager := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

			scriptName := args[0]
			scriptArgs := args[1:]

			if err := aliasManager.RunScript(scriptName, scriptArgs); err != nil {
				return fmt.Errorf("failed to run script '%s': %w", scriptName, err)
			}
			return nil
		},
	}

	flags.Set(cmd, &opts.configPath, "config", "c", "config file path")
	return cmd
}
