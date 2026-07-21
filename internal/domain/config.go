package domain

import (
	"os"
	"path/filepath"
	"strings"

	shelladapter "github.com/functions/internal/adapters/shell"
)

// Config describes the resolved configuration.
type Config struct {
	Editor               string `toml:"editor"`
	Primary              string `toml:"primary"`
	Secondary            string `toml:"secondary"`
	Headings             string `toml:"headings"`
	Text                 string `toml:"text"`
	TextHighlight        string `toml:"text_highlight"`
	DescriptionHighlight string `toml:"description_highlight"`
	Tags                 string `toml:"tags"`
	Flags                string `toml:"flags"`
	Muted                string `toml:"muted"`
	Accent               string `toml:"accent"`
	Border               string `toml:"border"`
	Error                string `toml:"error"`
	Success              string `toml:"success"`
	InteractiveDefault   bool   `toml:"interactive_default"`
	PlainText            bool   `toml:"plain_text"`
	ConfirmDelete        bool   `toml:"confirm_delete"`
	ListSpacing          string `toml:"list_spacing"`
	
	// Alias settings
	AliasFile          string            `toml:"alias_file"`
	Shell              string            `toml:"shell"`
	IndexFolders       []string          `toml:"index_folders"`
	CacheInterval      int               `toml:"cache_interval"`
	ScriptIcons        map[string]string `toml:"script_icons"`
	AutoAliasSeparator     string `toml:"auto_alias_separator"`
	AutoAliasLowercase     bool   `toml:"auto_alias_lowercase"`
	DefaultAliasPartLength int    `toml:"default_alias_part_length"`
	HomeIcon               string `toml:"home_icon"`
	DefaultSortBy          string `toml:"default_sort_by"`
	FunctionAlias          string `toml:"function_alias"`
	InteractiveAlias       string `toml:"interactive_alias"`
}

// DefaultConfig returns the default configuration values.
func DefaultConfig() Config {
	home, _ := os.UserHomeDir()
	detectedShell := shelladapter.DetectShell()
	aliasFile := filepath.Join(home, ".aliases", GetAliasFileName(detectedShell))
	
	return Config{
		Editor:               "nvim",
		Headings:             "15",
		Primary:              "02",
		Secondary:            "06",
		Text:                 "07",
		TextHighlight:        "06",
		DescriptionHighlight: "05",
		Tags:                 "13",
		Flags:                "12",
		Muted:                "08",
		Accent:               "13",
		Border:               "08",
		Error:                "01",
		Success:              "02",
		InteractiveDefault:   false,
		PlainText:            false,
		ConfirmDelete:        true,
		ListSpacing:          "space",
		AliasFile:            aliasFile,
		Shell:                detectedShell,
		IndexFolders:         []string{},
		CacheInterval:        300,
		ScriptIcons:          map[string]string{},
		AutoAliasSeparator:     "",
		AutoAliasLowercase:     true,
		DefaultAliasPartLength: 1, // Take 1 character from each part by default
		HomeIcon:               "~",
		DefaultSortBy:        "newest",
		FunctionAlias:        "true",
		InteractiveAlias:     "al",
	}
}

// GetAliasFileName returns the appropriate alias filename for the shell.
func GetAliasFileName(shell string) string {
	switch shell {
	case "fish":
		return "aliases.fish"
	case "nu", "nushell":
		return "aliases.nu"
	case "zsh":
		return "aliases.zsh"
	default: // bash, sh
		return "aliases.sh"
	}
}

// ResolvedAliasFile returns the expanded path to the alias file.
func (c Config) ResolvedAliasFile() string {
	return expandPath(c.AliasFile)
}

func expandPath(value string) string {
	expanded := os.ExpandEnv(value)
	if expanded == "" {
		return expanded
	}
	if expanded == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return expanded
	}
	if strings.HasPrefix(expanded, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(expanded, "~/"))
		}
	}
	return expanded
}

func xdgHome(envKey, fallbackSuffix string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, fallbackSuffix)
}
