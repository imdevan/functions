package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/functions/internal/config"
	"github.com/functions/internal/domain"
)

// TestEnv holds test environment setup.
type TestEnv struct {
	TempDir    string
	ConfigPath string
	Cleanup    func()
}

// SetupTestEnv creates a complete test environment with config and temp directories.
func SetupTestEnv(t *testing.T) *TestEnv {
	t.Helper()

	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config")
	bookmarkDir := filepath.Join(tempDir, ".bookmarks")

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.MkdirAll(bookmarkDir, 0o755); err != nil {
		t.Fatalf("failed to create bookmark dir: %v", err)
	}

	// Create test config file
	configPath := filepath.Join(configDir, "config.toml")

	cfg := domain.Config{
		AliasFile:          filepath.Join(bookmarkDir, "aliases.sh"),
		Shell:              "bash",
		Editor:             "vim",
		AutoAliasLowercase: true,
		AutoAliasSeparator: "",
	}

	// Save config using toml
	data, err := toml.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal test config: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("failed to save test config: %v", err)
	}

	return &TestEnv{
		TempDir:    tempDir,
		ConfigPath: configPath,
		Cleanup:    func() {},
	}
}

// LoadTestConfig loads a config from the given path.
func LoadTestConfig(t *testing.T, configPath string) domain.Config {
	t.Helper()

	mgr := config.NewManager(filepath.Dir(configPath))
	cfg, err := mgr.LoadWithOverride(configPath)
	if err != nil {
		t.Fatalf("failed to load test config: %v", err)
	}

	return cfg
}
