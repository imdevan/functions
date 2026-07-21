package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/functions/internal/alias"
	"github.com/functions/internal/config"
	"github.com/functions/internal/domain"
	"github.com/functions/internal/flags"
)

func newImportCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "import <path>",
		Short: "Import aliases from a file or folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			importPath := args[0]

			cwd, _ := os.Getwd()
			cfg := config.Load(cwd, configPath)

			// Clean import path
			importPath = filepath.Clean(importPath)
			if !filepath.IsAbs(importPath) {
				importPath = filepath.Join(cwd, importPath)
			}

			// Check if import path exists
			info, err := os.Stat(importPath)
			if err != nil {
				return fmt.Errorf("import path %q not found: %w", importPath, err)
			}

			var importFiles []string
			if info.IsDir() {
				err = filepath.Walk(importPath, func(path string, fileInfo os.FileInfo, walkErr error) error {
					if walkErr != nil {
						return walkErr
					}
					if !fileInfo.IsDir() {
						importFiles = append(importFiles, path)
					}
					return nil
				})
				if err != nil {
					return fmt.Errorf("failed to scan folder: %w", err)
				}
			} else {
				importFiles = []string{importPath}
			}

			defaultAliasFile := cfg.ResolvedAliasFile()
			defaultManager := alias.NewManager(defaultAliasFile, cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, nil)

			// Load existing aliases
			existing, err := defaultManager.Load()
			if err != nil {
				return fmt.Errorf("failed to load existing aliases: %w", err)
			}

			aliasMap := make(map[string]domain.Alias)
			var order []string

			for _, al := range existing {
				aliasMap[al.Name] = al
				order = append(order, al.Name)
			}

			importedCount := 0
			overwrittenCount := 0

			// Load and parse aliases from each file to import
			for _, file := range importFiles {
				if file == defaultAliasFile {
					continue
				}

				fileManager := alias.NewManager(file, cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, nil)
				importedAliases, err := fileManager.Load()
				if err != nil {
					continue
				}

				for _, al := range importedAliases {
					if _, exists := aliasMap[al.Name]; exists {
						overwrittenCount++
					} else {
						order = append(order, al.Name)
						importedCount++
					}
					
					aliasMap[al.Name] = domain.Alias{
						Name:        al.Name,
						Value:       al.Value,
						Description: al.Description,
						SourceFile:  defaultAliasFile,
					}
				}
			}

			// Build final merged list
			var merged []domain.Alias
			for _, name := range order {
				merged = append(merged, aliasMap[name])
			}

			// Save back to default file
			if err := defaultManager.Save(merged); err != nil {
				return fmt.Errorf("failed to save imported aliases: %w", err)
			}

			cmd.Printf("Imported %d new aliases, overwritten %d duplicates.\n", importedCount, overwrittenCount)
			return nil
		},
	}

	flags.Set(cmd, &configPath, "config", "c", "config file path")

	return cmd
}
