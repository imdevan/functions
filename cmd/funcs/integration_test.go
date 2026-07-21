package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/funcs/internal/alias"
	"github.com/funcs/internal/domain"
	"github.com/funcs/internal/testutil"
)

func TestIntegration_AliasCurrentDirectory(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Create test directory
	testDir := filepath.Join(env.TempDir, "my-cool-project")
	if err := os.MkdirAll(testDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Change current working directory to testDir so the default alias creation uses it
	origDir, err := os.Getwd()
	if err == nil {
		defer os.Chdir(origDir)
		os.Chdir(testDir)
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{"-c", env.ConfigPath, "mcp"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to create alias: %v", err)
	}

	cfg := testutil.LoadTestConfig(t, env.ConfigPath)
	mgr := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

	aliases, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load aliases: %v", err)
	}

	if len(aliases) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(aliases))
	}

	al := aliases[0]
	if al.Name != "mcp" {
		t.Errorf("expected name 'mcp', got %q", al.Name)
	}
	if al.Value != "cd "+testDir {
		t.Errorf("expected value 'cd %s', got %q", testDir, al.Value)
	}
}

func TestIntegration_CustomAlias(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	cmd := newRootCmd()
	cmd.SetArgs([]string{"add", "sayhello", "echo hello", "say hello description", "-c", env.ConfigPath, "-y"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to add alias: %v", err)
	}

	cfg := testutil.LoadTestConfig(t, env.ConfigPath)
	mgr := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

	aliases, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load aliases: %v", err)
	}

	if len(aliases) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(aliases))
	}

	al := aliases[0]
	if al.Name != "sayhello" {
		t.Errorf("expected name 'sayhello', got %q", al.Name)
	}
	if al.Value != "echo hello" {
		t.Errorf("expected value 'echo hello', got %q", al.Value)
	}
	if al.Description != "say hello description" {
		t.Errorf("expected description 'say hello description', got %q", al.Description)
	}
}

func TestIntegration_DeleteAlias(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Add alias first
	cmd := newRootCmd()
	cmd.SetArgs([]string{"add", "todelete", "echo delete", "to delete description", "-c", env.ConfigPath, "-y"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to add alias: %v", err)
	}

	cfg := testutil.LoadTestConfig(t, env.ConfigPath)
	mgr := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

	// Verify it exists
	aliases, _ := mgr.Load()
	if len(aliases) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(aliases))
	}

	// Delete it
	cmd2 := newRootCmd()
	cmd2.SetArgs([]string{"delete", "todelete", "-c", env.ConfigPath, "-f"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("failed to delete alias: %v", err)
	}

	// Verify it's gone
	aliases, _ = mgr.Load()
	if len(aliases) != 0 {
		t.Errorf("expected 0 aliases, got %d", len(aliases))
	}
}

func TestIntegration_Import(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Add an existing alias to the default file first
	cmd := newRootCmd()
	cmd.SetArgs([]string{"add", "foo", "echo existing", "existing desc", "-c", env.ConfigPath, "-y"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to add initial alias: %v", err)
	}

	// Create a temporary directory and files to import
	importDir := filepath.Join(env.TempDir, "import_src")
	if err := os.MkdirAll(importDir, 0o755); err != nil {
		t.Fatal(err)
	}

	importFile1 := filepath.Join(importDir, "aliases1.sh")
	content1 := "alias foo='echo overridden' # new desc\nalias bar='echo bar'\n"
	if err := os.WriteFile(importFile1, []byte(content1), 0o644); err != nil {
		t.Fatal(err)
	}

	importFile2 := filepath.Join(importDir, "aliases2.sh")
	content2 := "alias baz='echo baz' # baz desc\n"
	if err := os.WriteFile(importFile2, []byte(content2), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run import on the directory
	cmd2 := newRootCmd()
	cmd2.SetArgs([]string{"import", importDir, "-c", env.ConfigPath})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("failed to import folder: %v", err)
	}

	cfg := testutil.LoadTestConfig(t, env.ConfigPath)
	mgr := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

	aliases, err := mgr.Load()
	if err != nil {
		t.Fatalf("failed to load aliases: %v", err)
	}

	// Map them for easy checking
	aliasMap := make(map[string]domain.Alias)
	for _, al := range aliases {
		aliasMap[al.Name] = al
	}

	// Check if all three exist
	if len(aliases) != 3 {
		t.Fatalf("expected 3 aliases after import, got %d", len(aliases))
	}

	// 'foo' should be overridden by the last one (imported file)
	fooAl, exists := aliasMap["foo"]
	if !exists {
		t.Fatal("expected 'foo' to exist")
	}
	if fooAl.Value != "echo overridden" || fooAl.Description != "new desc" {
		t.Errorf("foo alias not correctly overridden: %+v", fooAl)
	}

	// 'bar' and 'baz' should exist
	if _, exists := aliasMap["bar"]; !exists {
		t.Error("expected 'bar' to exist")
	}
	if _, exists := aliasMap["baz"]; !exists {
		t.Error("expected 'baz' to exist")
	}
}

func TestIntegration_AddScriptWithEditor(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Note: set editor to true/echo so it doesn't fail trying to launch editor
	cmd := newRootCmd()
	cmd.SetArgs([]string{"add", "testscript", "scripts", "-c", env.ConfigPath, "-y"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to add script: %v", err)
	}

	cfg := testutil.LoadTestConfig(t, env.ConfigPath)
	mgr := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

	created, err := mgr.Get("testscript")
	if err != nil {
		t.Fatalf("failed to get created script: %v", err)
	}

	if created.Type != domain.TypeScript {
		t.Errorf("expected type 'scripts', got %q", created.Type)
	}

	if created.SourceFile == "" || !strings.HasSuffix(created.SourceFile, "testscript.sh") {
		t.Errorf("expected SourceFile to point to script file, got %q", created.SourceFile)
	}
}

func TestIntegration_OpenCommand(t *testing.T) {
	env := testutil.SetupTestEnv(t)

	// Add script first
	cmd := newRootCmd()
	cmd.SetArgs([]string{"add", "openscript", "scripts", "-c", env.ConfigPath, "-y"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("failed to add script: %v", err)
	}

	// Test open command with non-existent name returns error
	cmd2 := newRootCmd()
	cmd2.SetArgs([]string{"open", "nonexistent", "-c", env.ConfigPath})
	if err := cmd2.Execute(); err == nil {
		t.Errorf("expected error opening nonexistent script, got nil")
	}
}



