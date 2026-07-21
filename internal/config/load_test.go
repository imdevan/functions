package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/functions/internal/utils"
)

func TestLoad_returnsDefaultWhenNoConfig(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))

	cfg := Load(cwd, "")

	if cfg.Editor == "" {
		t.Fatal("expected default editor")
	}
}

func TestLoad_loadsFromGlobalConfig(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))

	configPath := utils.ConfigPathGlobal()
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("editor = \"vim\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := Load(cwd, "")

	if cfg.Editor != "vim" {
		t.Fatalf("expected editor \"vim\", got %q", cfg.Editor)
	}
}

func TestLoad_usesOverridePath(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	overridePath := filepath.Join(root, "custom.toml")
	if err := os.WriteFile(overridePath, []byte("editor = \"emacs\"\n"), 0o644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	cfg := Load(cwd, overridePath)

	if cfg.Editor != "emacs" {
		t.Fatalf("expected editor \"emacs\", got %q", cfg.Editor)
	}
}

func TestLoad_returnsDefaultOnBadOverridePath(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "project")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := Load(cwd, "/nonexistent/path/config.toml")

	if cfg.Editor == "" {
		t.Fatal("expected default config on error")
	}
}
