package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/funcs/internal/alias"
	"github.com/funcs/internal/config"
	"github.com/funcs/internal/flags"
)

type openOptions struct {
	configPath string
}

func newOpenCmd() *cobra.Command {
	opts := &openOptions{}
	cmd := &cobra.Command{
		Use:   "open [name]",
		Short: "Open a function, script, or alias file in your editor",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			cfg := config.Load(cwd, opts.configPath)
			aliasManager := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

			if len(args) == 0 {
				return openEditor(cfg.Editor, cfg.ResolvedAliasFile(), 0)
			}

			name := args[0]
			al, err := aliasManager.Get(name)
			if err != nil {
				return fmt.Errorf("function or script '%s' not found: %w", name, err)
			}

			if al.SourceFile == "" {
				return fmt.Errorf("no source file recorded for '%s'", name)
			}

			return openEditor(cfg.Editor, al.SourceFile, 0)
		},
	}

	flags.Set(cmd, &opts.configPath, "config", "c", "config file path")
	return cmd
}
