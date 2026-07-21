package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/funcs/internal/alias"
	"github.com/funcs/internal/config"
	"github.com/funcs/internal/flags"
)

func newListCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all aliases",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := cmd.Flags().GetString("cwd")
			if err != nil {
				cwd = "."
			}

			cfg := config.Load(cwd, configPath)

			aliasManager := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)
			aliases, err := aliasManager.Load()
			if err != nil {
				return err
			}

			if len(aliases) == 0 {
				cmd.Println("No aliases found")
				return nil
			}

			// Find max name length for alignment (including description)
			maxName := 0
			for _, al := range aliases {
				nameLen := len(al.Name)
				if al.Description != "" {
					nameLen += len(" # " + al.Description)
				}
				if nameLen > maxName {
					maxName = nameLen
				}
			}

			for _, al := range aliases {
				name := al.Name
				if al.Description != "" {
					name += " # " + al.Description
				}
				line := fmt.Sprintf("%-*s  %s", maxName, name, al.Value)
				cmd.Println(line)
			}

			return nil
		},
	}

	flags.Set(cmd, &configPath, "config", "c", "config file path")

	return cmd
}
