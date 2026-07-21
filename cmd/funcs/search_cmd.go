package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/funcs/internal/alias"
	"github.com/funcs/internal/config"
	"github.com/funcs/internal/domain"
	"github.com/funcs/internal/flags"
	"github.com/funcs/internal/ui"
)

type searchOptions struct {
	configPath  string
	interactive bool
}

func newSearchCmd() *cobra.Command {
	opts := &searchOptions{}
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search functions and aliases",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()
			cfg := config.Load(cwd, opts.configPath)

			aliasManager := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

			if opts.interactive || len(args) == 0 {
				return runInteractive(cmd, &rootOptions{configPath: opts.configPath}, cfg)
			}

			query := strings.ToLower(args[0])
			aliases, err := aliasManager.Load()
			if err != nil {
				return err
			}

			var matches []domain.Alias
			for _, item := range aliases {
				if strings.Contains(strings.ToLower(item.Name), query) ||
					strings.Contains(strings.ToLower(item.Description), query) ||
					strings.Contains(strings.ToLower(item.Value), query) ||
					strings.Contains(strings.ToLower(string(item.Type)), query) {
					matches = append(matches, item)
				}
			}

			if len(matches) == 0 {
				cmd.Printf("No functions or aliases matching '%s' found\n", args[0])
				return nil
			}

			theme := ui.ThemeFromConfig(cfg)
			tagStyle := lipgloss.NewStyle().Foreground(theme.Tags).Bold(true)
			nameStyle := lipgloss.NewStyle().Foreground(theme.Headings).Bold(true)
			descStyle := lipgloss.NewStyle().Foreground(theme.Muted)

			for _, item := range matches {
				fnType := string(item.Type)
				if fnType == "" {
					fnType = "alias"
				}
				typeTag := tagStyle.Render(fmt.Sprintf("[%s]", fnType))
				line := fmt.Sprintf("%-10s %s", typeTag, nameStyle.Render(item.Name))
				if item.Description != "" {
					line += " " + descStyle.Render("# "+item.Description)
				}
				if item.SourceFile != "" {
					line += " " + descStyle.Render("("+item.SourceFile+")")
				}
				cmd.Println(line)
			}

			return nil
		},
	}

	flags.Set(cmd, &opts.configPath, "config", "c", "config file path")
	flags.Set(cmd, &opts.interactive, "interactive", "i", "launch interactive TUI search")

	return cmd
}
