package alias

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/functions/internal/adapters/shell"
	"github.com/functions/internal/domain"
	"github.com/functions/internal/utils"
)

var (
	ErrAliasNotFound = errors.New("alias not found")
	ErrInvalidAlias  = errors.New("invalid alias: must contain only alphanumeric characters, hyphens, and underscores")
	ErrReservedAlias = errors.New("invalid alias: cannot use shell reserved keywords")
)

var reservedKeywords = map[string]bool{
	"if":       true,
	"then":     true,
	"else":     true,
	"elif":     true,
	"fi":       true,
	"case":     true,
	"esac":     true,
	"for":      true,
	"select":   true,
	"while":    true,
	"until":    true,
	"do":       true,
	"done":     true,
	"in":       true,
	"function": true,
	"time":     true,
	"switch":   true,
	"begin":    true,
	"end":      true,
	"and":      true,
	"or":       true,
	"not":      true,
	"match":    true,
	"loop":     true,
	"def":      true,
	"alias":    true,
	"export":   true,
	"use":      true,
	"let":      true,
	"mut":      true,
	"const":    true,
}

func IsReservedKeyword(alias string) bool {
	return reservedKeywords[strings.ToLower(alias)]
}

func IsValidAlias(alias string) bool {
	if len(alias) == 0 {
		return false
	}
	for _, r := range alias {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// Manager handles alias CRUD operations.
type Manager struct {
	filePath         string
	shell            string
	functionAlias    string
	interactiveAlias string
	indexFolders     []string
	shellAdapter     *shell.Adapter
}

// NewManager creates a new alias manager.
func NewManager(filePath string, shellType string, functionAlias string, interactiveAlias string, indexFolders []string) *Manager {
	return &Manager{
		filePath:         expandPath(filePath),
		shell:            shellType,
		functionAlias:    functionAlias,
		interactiveAlias: interactiveAlias,
		indexFolders:     indexFolders,
		shellAdapter:     shell.New(shellType),
	}
}

// Load reads all aliases from the default alias file and any index folders.
func (m *Manager) Load() ([]domain.Alias, error) {
	var allAliases []domain.Alias

	// 1. Load from default alias file
	if _, err := os.Stat(m.filePath); err == nil {
		data, err := os.ReadFile(m.filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read default alias file: %w", err)
		}
		aliases, err := m.parseShellScript(string(data))
		if err != nil {
			return nil, err
		}
		allAliases = append(allAliases, aliases...)
	}

	// 2. Load from index folders
	for _, pattern := range m.indexFolders {
		matches, err := filepath.Glob(expandPath(pattern))
		if err != nil {
			continue
		}
		for _, match := range matches {
			if match == m.filePath {
				continue
			}
			info, err := os.Stat(match)
			if err != nil || info.IsDir() {
				continue
			}
			data, err := os.ReadFile(match)
			if err != nil {
				continue
			}
			fileManager := &Manager{filePath: match, shell: m.shell}
			aliases, err := fileManager.parseShellScript(string(data))
			if err == nil {
				allAliases = append(allAliases, aliases...)
			}
		}
	}

	return allAliases, nil
}

// parseShellScript extracts aliases from shell script.
func (m *Manager) parseShellScript(content string) ([]domain.Alias, error) {
	var aliases []domain.Alias
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		name, val, desc, ok := parseAliasLine(line, m.shell)
		if ok {
			aliases = append(aliases, domain.Alias{
				Name:        name,
				Value:       val,
				Description: desc,
				SourceFile:  m.filePath,
			})
		}
	}
	return aliases, scanner.Err()
}

func parseAliasLine(line, shellType string) (name, value, description string, ok bool) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "alias ") {
		return "", "", "", false
	}

	content := strings.TrimPrefix(line, "alias ")
	content = strings.TrimSpace(content)

	parts := strings.SplitN(content, "=", 2)
	if len(parts) != 2 {
		return "", "", "", false
	}

	name = strings.TrimSpace(parts[0])
	rightSide := strings.TrimSpace(parts[1])

	var val, desc string
	if len(rightSide) > 0 && (rightSide[0] == '\'' || rightSide[0] == '"') {
		quoteChar := rightSide[0]
		escaped := false
		closingIdx := -1
		for i := 1; i < len(rightSide); i++ {
			if escaped {
				escaped = false
				continue
			}
			if rightSide[i] == '\\' {
				escaped = true
				continue
			}
			if rightSide[i] == quoteChar {
				closingIdx = i
				break
			}
		}

		if closingIdx != -1 {
			rawVal := rightSide[1:closingIdx]
			val = strings.ReplaceAll(rawVal, "\\"+string(quoteChar), string(quoteChar))

			afterQuote := rightSide[closingIdx+1:]
			if hashIdx := strings.Index(afterQuote, "#"); hashIdx != -1 {
				desc = strings.TrimSpace(afterQuote[hashIdx+1:])
			}
		} else {
			val = rightSide
		}
	} else {
		if hashIdx := strings.Index(rightSide, "#"); hashIdx != -1 {
			val = strings.TrimSpace(rightSide[:hashIdx])
			desc = strings.TrimSpace(rightSide[hashIdx+1:])
		} else {
			val = rightSide
		}
	}

	return name, val, desc, true
}

// Save writes the given list of aliases to the default alias file.
func (m *Manager) Save(aliases []domain.Alias) error {
	if err := os.MkdirAll(filepath.Dir(m.filePath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return m.generateShellScript(aliases)
}

func (m *Manager) generateShellScript(aliases []domain.Alias) error {
	var script strings.Builder

	script.WriteString("# Generated by alias manager\n")
	script.WriteString("# Do not edit manually - changes will be overwritten\n\n")

	if m.functionAlias != "" && m.functionAlias != "false" {
		functionName := "aliases"
		if m.functionAlias != "true" {
			functionName = m.functionAlias
		}
		script.WriteString(m.generateFunctionWrapper(functionName))
		script.WriteString("\n")
	}

	if m.interactiveAlias != "" && m.interactiveAlias != "false" {
		script.WriteString(m.generateInteractiveWrapper(m.interactiveAlias))
		script.WriteString("\n")
	}

	for _, al := range aliases {
		script.WriteString(m.formatSingleAlias(al))
	}

	return os.WriteFile(m.filePath, []byte(script.String()), 0o644)
}

func (m *Manager) formatSingleAlias(al domain.Alias) string {
	escapedVal := utils.EscapeAliasValue(al.Value)
	switch m.shell {
	case "nu", "nushell":
		if al.Description != "" {
			return fmt.Sprintf("alias %s = \"%s\" # %s\n", al.Name, escapedVal, al.Description)
		}
		return fmt.Sprintf("alias %s = \"%s\"\n", al.Name, escapedVal)
	default:
		if al.Description != "" {
			return fmt.Sprintf("alias %s=\"%s\" # %s\n", al.Name, escapedVal, al.Description)
		}
		return fmt.Sprintf("alias %s=\"%s\"\n", al.Name, escapedVal)
	}
}

//go:embed templates/function_wrapper.tmpl
var functionWrapperTemplate string

//go:embed templates/interactive_wrapper.tmpl
var interactiveWrapperTemplate string

func (m *Manager) generateFunctionWrapper(functionName string) string {
	tmpl, err := template.New("function_wrapper").Parse(functionWrapperTemplate)
	if err != nil {
		panic(err)
	}
	var builder strings.Builder
	data := struct {
		Shell        string
		FunctionName string
		FilePath     string
	}{
		Shell:        m.shell,
		FunctionName: functionName,
		FilePath:     m.filePath,
	}
	if err := tmpl.Execute(&builder, data); err != nil {
		panic(err)
	}
	return builder.String()
}

func (m *Manager) generateInteractiveWrapper(functionName string) string {
	tmpl, err := template.New("interactive_wrapper").Parse(interactiveWrapperTemplate)
	if err != nil {
		panic(err)
	}
	var builder strings.Builder

	targetFunction := "aliases"
	if m.functionAlias != "" && m.functionAlias != "false" && m.functionAlias != "true" {
		targetFunction = m.functionAlias
	}

	data := struct {
		Shell                 string
		FunctionName          string
		TargetFunction        string
		IsMainFunctionEnabled bool
		FilePath              string
	}{
		Shell:                 m.shell,
		FunctionName:          functionName,
		TargetFunction:        targetFunction,
		IsMainFunctionEnabled: m.functionAlias != "" && m.functionAlias != "false",
		FilePath:              m.filePath,
	}
	if err := tmpl.Execute(&builder, data); err != nil {
		panic(err)
	}
	return builder.String()
}

// Add creates or updates an alias in its SourceFile.
func (m *Manager) Add(al domain.Alias) error {
	if !IsValidAlias(al.Name) {
		return ErrInvalidAlias
	}
	if IsReservedKeyword(al.Name) {
		return ErrReservedAlias
	}

	existing, err := m.Get(al.Name)
	targetFile := m.filePath
	if err == nil && existing.SourceFile != "" {
		targetFile = existing.SourceFile
	}
	al.SourceFile = targetFile

	return m.writeAliasToFile(targetFile, al, false)
}

// Get retrieves an alias by name.
func (m *Manager) Get(name string) (domain.Alias, error) {
	aliases, err := m.Load()
	if err != nil {
		return domain.Alias{}, err
	}

	for _, a := range aliases {
		if a.Name == name {
			return a, nil
		}
	}

	return domain.Alias{}, ErrAliasNotFound
}

// Delete removes an alias.
func (m *Manager) Delete(name string) error {
	existing, err := m.Get(name)
	if err != nil {
		return err
	}
	targetFile := m.filePath
	if existing.SourceFile != "" {
		targetFile = existing.SourceFile
	}
	return m.writeAliasToFile(targetFile, existing, true)
}

// Exists checks if an alias exists.
func (m *Manager) Exists(name string) (bool, error) {
	_, err := m.Get(name)
	if err == ErrAliasNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (m *Manager) writeAliasToFile(targetFile string, al domain.Alias, isDelete bool) error {
	targetFile = expandPath(targetFile)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(targetFile), 0o755); err != nil {
		return err
	}

	if targetFile == m.filePath {
		aliases, err := m.Load()
		if err != nil {
			return err
		}
		var fileAliases []domain.Alias
		found := false
		for i, a := range aliases {
			if a.SourceFile == m.filePath || a.SourceFile == "" {
				if a.Name == al.Name {
					if isDelete {
						continue
					}
					aliases[i] = al
					found = true
				}
				fileAliases = append(fileAliases, aliases[i])
			}
		}
		if !found && !isDelete {
			fileAliases = append(fileAliases, al)
		}
		return m.generateShellScript(fileAliases)
	}

	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		if isDelete {
			return nil
		}
		line := m.formatSingleAlias(al)
		return os.WriteFile(targetFile, []byte(line), 0o644)
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		name, _, _, ok := parseAliasLine(line, m.shell)
		if ok && name == al.Name {
			if isDelete {
				lines = append(lines[:i], lines[i+1:]...)
			} else {
				lines[i] = strings.TrimSuffix(m.formatSingleAlias(al), "\n")
			}
			found = true
			break
		}
	}

	if !found && !isDelete {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, strings.TrimSuffix(m.formatSingleAlias(al), "\n"))
	}

	return os.WriteFile(targetFile, []byte(strings.Join(lines, "\n")), 0o644)
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

func GenerateAlias(path string, separator string, lowercase bool, partLength int) string {
	return utils.GenerateAlias(path, separator, lowercase, partLength)
}
