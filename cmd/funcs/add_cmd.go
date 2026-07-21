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

type addOptions struct {
	configPath  string
	fnType      string
	value       string
	aliasName   string
	description string
	yes         bool
}

func newAddCmd() *cobra.Command {
	opts := &addOptions{}
	cmd := &cobra.Command{
		Use:   "add [name] [type] [description]",
		Short: "Add a new function or alias (types: global, lib, scripts, alias)",
		Args:  cobra.MaximumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAddInteractive(cmd, args, opts)
		},
	}

	flags.Set(cmd, &opts.configPath, "config", "c", "config file path")
	flags.Set(cmd, &opts.fnType, "type", "t", "function type: global, lib, scripts, alias")
	flags.Set(cmd, &opts.value, "value", "v", "alias value or command")
	flags.Set(cmd, &opts.aliasName, "alias", "a", "alias name for the function")
	flags.Set(cmd, &opts.description, "description", "d", "function description")
	flags.Set(cmd, &opts.yes, "yes", "y", "skip editor, save directly")

	return cmd
}

func runAddInteractive(cmd *cobra.Command, args []string, opts *addOptions) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	cfg := config.Load(cwd, opts.configPath)
	aliasManager := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

	var name, arg1, description string
	if len(args) >= 1 {
		name = args[0]
	}
	if len(args) >= 2 {
		arg1 = args[1]
	}
	if len(args) >= 3 {
		description = args[2]
	}

	if opts.description != "" {
		description = opts.description
	}

	fnType := domain.TypeAlias
	value := opts.value

	if opts.fnType != "" {
		fnType = domain.FunctionType(opts.fnType)
		if value == "" && arg1 != string(fnType) {
			value = arg1
		}
	} else if arg1 == "global" || arg1 == "lib" || arg1 == "scripts" || arg1 == "alias" {
		fnType = domain.FunctionType(arg1)
	} else if arg1 != "" {
		value = arg1
		fnType = domain.TypeAlias
	}

	if fnType == "" {
		fnType = domain.TypeGlobal
	}

	// -y: save directly without opening editor
	if opts.yes {
		if name == "" {
			return fmt.Errorf("function name is required")
		}
		alItem := domain.Alias{
			Name:        name,
			Type:        fnType,
			Value:       value,
			AliasName:   opts.aliasName,
			Description: description,
		}
		exists, err := aliasManager.Exists(name)
		if err != nil {
			return err
		}
		if err := aliasManager.Add(alItem); err != nil {
			return err
		}
		action := "created"
		if exists {
			action = "updated"
		}
		printSuccess(cfg, action, name, value)
		return nil
	}

	theme := ui.ThemeFromConfig(cfg)
	var initialAlias domain.Alias
	var isPrefilled bool
	if name != "" || arg1 != "" || description != "" || opts.value != "" {
		initialAlias = domain.Alias{
			Name:        name,
			Type:        fnType,
			Value:       opts.value,
			AliasName:   opts.aliasName,
			Description: description,
		}
		isPrefilled = true
	}

	exists, err := aliasManager.Exists(initialAlias.Name)
	if err != nil {
		return err
	}

	var m ui.AliasFormModel
	if isPrefilled {
		m = ui.NewAliasFormModelEdit(theme, initialAlias).WithTitle("Add Function / Alias")
	} else {
		m = ui.NewAliasFormModel(theme, "", "").WithTitle("Add Function / Alias")
	}

	if err := runAliasFormWorkflow(cmd, cfg, aliasManager, m, initialAlias, exists, false, "Add"); err != nil {
		return err
	}

	// Open created/edited function file in editor unless user suppressed it
	if name != "" {
		created, err := aliasManager.Get(name)
		if err == nil && created.SourceFile != "" && created.Type != domain.TypeAlias {
			_ = openEditor(cfg.Editor, created.SourceFile, 0)
		}
	}

	return nil
}
