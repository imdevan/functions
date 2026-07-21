package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/funcs/internal/alias"
	"github.com/funcs/internal/config"
	"github.com/funcs/internal/domain"
	"github.com/funcs/internal/flags"
	"github.com/funcs/internal/ui"
)

type editOptions struct {
	configPath string
}

func newEditCmd() *cobra.Command {
	opts := &editOptions{}
	cmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Edit a function, script, or alias in your editor",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			cfg := config.Load(cwd, opts.configPath)
			return runEditCommand(cmd, args, opts, cfg)
		},
	}

	flags.Set(cmd, &opts.configPath, "config", "c", "config file path")

	return cmd
}

func runEditCommand(cmd *cobra.Command, args []string, opts *editOptions, cfg domain.Config) error {
	aliasManager := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

	// If no function/alias name provided, open the main aliases file or functions directory
	if len(args) == 0 {
		return openEditor(cfg.Editor, cfg.ResolvedAliasFile(), 0)
	}

	name := args[0]

	// Check if function/alias exists
	exists, err := aliasManager.Exists(name)
	if err != nil {
		return err
	}

	if exists {
		al, err := aliasManager.Get(name)
		if err == nil && al.SourceFile != "" {
			return openEditor(cfg.Editor, al.SourceFile, 0)
		}
	}

	// If not found as exact source file, open form workflow or create new
	al := domain.Alias{
		Name: name,
		Type: domain.TypeGlobal,
	}

	theme := ui.ThemeFromConfig(cfg)
	m := ui.NewAliasFormModelEdit(theme, al).WithTitle(fmt.Sprintf("'%s' Not Found, Add Function", name))

	return runAliasFormWorkflow(cmd, cfg, aliasManager, m, al, false, true, "Edit")
}
