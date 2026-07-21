package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/funcs/internal/alias"
	"github.com/funcs/internal/config"
	"github.com/funcs/internal/flags"
	"github.com/funcs/internal/ui"
)

func newDeleteCmd() *cobra.Command {
	var configPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete an alias",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]

			cwd, _ := os.Getwd()
			cfg := config.Load(cwd, configPath)

			aliasManager := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

			// Check if alias exists
			al, err := aliasManager.Get(name)
			if err == alias.ErrAliasNotFound {
				return fmt.Errorf("alias '%s' not found", name)
			}
			if err != nil {
				return err
			}

			// Confirm deletion unless --force or confirm_delete is false
			if !force && cfg.ConfirmDelete {
				theme := ui.ThemeFromConfig(cfg)
				confirmModel := ui.NewConfirmationModel(
					"Delete Alias",
					fmt.Sprintf("Delete alias '%s → %s'?", al.Name, al.Value),
					theme,
				).WithTitleColor(theme.Error)

				p := tea.NewProgram(confirmModel, tea.WithoutSignalHandler())
				result, err := p.Run()
				if err != nil {
					return err
				}

				if confirmResult, ok := result.(ui.ConfirmationModel); ok {
					if !confirmResult.ChoiceValue() {
						cmd.Println(ui.CanceledMessage(theme, "Delete"))
						return nil
					}
				}
			}

			if err := aliasManager.Delete(name); err != nil {
				return err
			}

			printSuccess(cfg, "deleted", name, "")
			return nil
		},
	}

	flags.Set(cmd, &configPath, "config", "c", "config file path")
	flags.Set(cmd, &force, "force", "f", "skip confirmation")

	return cmd
}
