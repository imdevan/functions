package alias

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/functions/internal/domain"
)

func TestGenerateAlias(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		separator  string
		lowercase  bool
		partLength int
		want       string
	}{
		{
			name:       "simple path - first letter",
			path:       "/home/user/my-cool-project",
			separator:  "",
			lowercase:  true,
			partLength: 1,
			want:       "mcp",
		},
		{
			name:       "simple path - two letters",
			path:       "/home/user/my-cool-project",
			separator:  "",
			lowercase:  true,
			partLength: 2,
			want:       "mycopr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateAlias(tt.path, tt.separator, tt.lowercase, tt.partLength)
			if got != tt.want {
				t.Errorf("GenerateAlias() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestManager_AddAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "aliases.sh")

	m := NewManager(filePath, "bash", "true", "al", nil)

	al := domain.Alias{
		Name:        "test",
		Value:       "echo hello",
		Description: "Test alias",
	}

	if err := m.Add(al); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	got, err := m.Get("test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Name != al.Name || got.Value != al.Value || got.Description != al.Description {
		t.Errorf("Get() = %v, want %v", got, al)
	}
}

func TestManager_IndexFolders(t *testing.T) {
	tmpDir := t.TempDir()
	defaultFile := filepath.Join(tmpDir, "aliases.sh")
	
	// Create another file in a folder to scan
	scanDir := filepath.Join(tmpDir, "dotfiles")
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		t.Fatalf("failed to create scan dir: %v", err)
	}
	
	customFile := filepath.Join(scanDir, "custom.sh")
	content := "alias custom='echo scan' # custom description\n"
	if err := os.WriteFile(customFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write custom file: %v", err)
	}
	
	m := NewManager(defaultFile, "bash", "true", "al", []string{filepath.Join(scanDir, "*.sh")})
	
	aliases, err := m.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	
	if len(aliases) != 1 {
		t.Fatalf("expected 1 alias, got %d", len(aliases))
	}
	
	if aliases[0].Name != "custom" || aliases[0].Value != "echo scan" || aliases[0].Description != "custom description" {
		t.Errorf("loaded alias is not correct: %+v", aliases[0])
	}
	
	// Try updating the custom alias and check if it writes back to the correct file
	aliases[0].Value = "echo updated"
	if err := m.Add(aliases[0]); err != nil {
		t.Fatalf("Add() update error = %v", err)
	}
	
	// Read custom file to verify it was updated
	data, err := os.ReadFile(customFile)
	if err != nil {
		t.Fatalf("failed to read custom file: %v", err)
	}
	
	expectedContent := "alias custom=\"echo updated\" # custom description\n"
	if string(data) != expectedContent {
		t.Errorf("custom file content = %q, want %q", string(data), expectedContent)
	}
}

func TestManager_FormatSingleAlias(t *testing.T) {
	tests := []struct {
		name       string
		shell      string
		alias      domain.Alias
		wantFormat string
	}{
		{
			name:  "bash simple no quotes",
			shell: "bash",
			alias: domain.Alias{Name: "t1", Value: "echo hi"},
			wantFormat: "alias t1=\"echo hi\"\n",
		},
		{
			name:  "bash with double quotes in value",
			shell: "bash",
			alias: domain.Alias{Name: "t1", Value: `echo "hi"`},
			wantFormat: "alias t1=\"echo 'hi'\"\n",
		},
		{
			name:  "bash with single quotes in value",
			shell: "bash",
			alias: domain.Alias{Name: "t1", Value: "echo 'hi'"},
			wantFormat: "alias t1=\"echo 'hi'\"\n",
		},
		{
			name:  "bash with description",
			shell: "bash",
			alias: domain.Alias{Name: "t1", Value: "echo hi", Description: "simple test"},
			wantFormat: "alias t1=\"echo hi\" # simple test\n",
		},
		{
			name:  "nushell with double quotes in value",
			shell: "nushell",
			alias: domain.Alias{Name: "t1", Value: `echo "hi"`, Description: "nu test"},
			wantFormat: "alias t1 = \"echo 'hi'\" # nu test\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "aliases.sh")
			m := NewManager(filePath, tt.shell, "false", "false", nil)
			got := m.formatSingleAlias(tt.alias)
			if got != tt.wantFormat {
				t.Errorf("formatSingleAlias() = %q, want %q", got, tt.wantFormat)
			}
		})
	}
}

