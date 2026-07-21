package domain

type FunctionType string

const (
	TypeGlobal  FunctionType = "global"
	TypeLib     FunctionType = "lib"
	TypeScript  FunctionType = "scripts"
	TypeAlias   FunctionType = "alias"
)

// Alias represents a command alias or function with metadata.
type Alias struct {
	Name        string       `toml:"name"`
	Value       string       `toml:"value"`
	Description string       `toml:"description,omitempty"`
	SourceFile  string       `toml:"source_file,omitempty"`
	Type        FunctionType `toml:"type,omitempty"`
	AliasName   string       `toml:"alias_name,omitempty"`
}

