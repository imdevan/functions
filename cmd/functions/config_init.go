package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	"github.com/functions/internal/adapters/editor"
	"github.com/functions/internal/config"
	"github.com/functions/internal/domain"
	"github.com/functions/internal/flags"
	"github.com/functions/internal/utils"
)

type configInitOptions struct {
	force        bool
	openInEditor bool
}

// @docs-command:
//
// name: config init
// description:
//
//	The config init command creates a new configuration file with default values
//	at the standard XDG config location ($XDG_CONFIG_HOME/bookmark/config.toml).
//
// example:
//
//	```bash
//	# Generate default config
//	bookmark config init
//
//	# Overwrite existing config
//	bookmark config init --force
//
//	# Generate and open in editor
//	bookmark config init --editor
//	```
//
// note:
//
//	The generated config file includes commented examples for all available options.
func newConfigInitCmd() *cobra.Command {
	opts := &configInitOptions{}
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a default config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigInit(cmd, opts)
		},
	}
	flags.Set(cmd, &opts.force, "force", "f", "overwrite existing config")
	flags.Set(cmd, &opts.openInEditor, "editor", "e", "open config in editor after creation")
	return cmd
}

func runConfigInit(cmd *cobra.Command, opts *configInitOptions) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	exists, err := config.NewManager(cwd).Exists()
	if err != nil {
		return err
	}
	if exists && !opts.force {
		return fmt.Errorf("config already exists at %s (use --force to overwrite)", utils.ConfigPathGlobal())
	}
	cfg := domain.DefaultConfig()
	path := utils.ConfigPathGlobal()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := renderConfigTemplate(cfg)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	if opts.openInEditor {
		editorAdapter := editor.New(cfg.Editor)
		if err := editorAdapter.Open(path); err != nil {
			return err
		}
	}
	cmd.Printf("Wrote config to %s\n", utils.ConfigPathGlobal())
	return nil
}

//go:embed templates/config.toml.tmpl
var configTemplateContent string

func renderConfigTemplate(cfg domain.Config) string {
	tmpl, err := template.New("config").Parse(configTemplateContent)
	if err != nil {
		panic(err)
	}
	var builder strings.Builder
	if err := tmpl.Execute(&builder, cfg); err != nil {
		panic(err)
	}
	return builder.String()
}
