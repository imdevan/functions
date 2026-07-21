package alias

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/funcs/internal/adapters/shell"
	"github.com/funcs/internal/domain"
	"github.com/funcs/internal/utils"
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

// Manager handles alias and function CRUD operations.
type Manager struct {
	functionsDir     string
	filePath         string
	shell            string
	functionAlias    string
	interactiveAlias string
	indexFolders     []string
	shellAdapter     *shell.Adapter
}

// NewManager creates a new alias/function manager.
func NewManager(filePath string, shellType string, functionAlias string, interactiveAlias string, indexFolders []string) *Manager {
	expPath := expandPath(filePath)
	parent := filepath.Dir(expPath)
	grandParent := filepath.Dir(parent)

	functionsDir := grandParent
	baseParent := filepath.Base(parent)
	if baseParent != "scripts" && baseParent != "global" && baseParent != "lib" {
		functionsDir = parent
	}
	return &Manager{
		functionsDir:     functionsDir,
		filePath:         expPath,
		shell:            shellType,
		functionAlias:    functionAlias,
		interactiveAlias: interactiveAlias,
		indexFolders:     indexFolders,
		shellAdapter:     shell.New(shellType),
	}
}

// NewManagerWithDir creates a manager with explicit functions directory and alias file.
func NewManagerWithDir(functionsDir string, filePath string, shellType string, functionAlias string, interactiveAlias string, indexFolders []string) *Manager {
	expDir := expandPath(functionsDir)
	if expDir == "" {
		home, _ := os.UserHomeDir()
		expDir = filepath.Join(home, ".functions")
	}
	expPath := expandPath(filePath)
	if expPath == "" {
		expPath = filepath.Join(expDir, "scripts", domain.GetAliasFileName(shellType))
	}
	return &Manager{
		functionsDir:     expDir,
		filePath:         expPath,
		shell:            shellType,
		functionAlias:    functionAlias,
		interactiveAlias: interactiveAlias,
		indexFolders:     indexFolders,
		shellAdapter:     shell.New(shellType),
	}
}

// FunctionsDir returns the functions directory.
func (m *Manager) FunctionsDir() string {
	return m.functionsDir
}

// Load reads all aliases and functions from .functions directory and any index folders.
func (m *Manager) Load() ([]domain.Alias, error) {
	var allAliases []domain.Alias
	seen := make(map[string]bool)

	// 1. Load functions from ~/.functions/global, lib, scripts
	globals := m.loadDirectoryFunctions(filepath.Join(m.functionsDir, "global"), domain.TypeGlobal)
	for _, g := range globals {
		if !seen[g.Name] {
			allAliases = append(allAliases, g)
			seen[g.Name] = true
		}
	}

	libs := m.loadDirectoryFunctions(filepath.Join(m.functionsDir, "lib"), domain.TypeLib)
	for _, l := range libs {
		if !seen[l.Name] {
			allAliases = append(allAliases, l)
			seen[l.Name] = true
		}
	}

	scripts := m.loadDirectoryFunctions(filepath.Join(m.functionsDir, "scripts"), domain.TypeScript)
	for _, s := range scripts {
		if !seen[s.Name] {
			allAliases = append(allAliases, s)
			seen[s.Name] = true
		}
	}

	// 2. Load from default alias file
	if _, err := os.Stat(m.filePath); err == nil {
		data, err := os.ReadFile(m.filePath)
		if err == nil {
			aliases, _ := m.parseShellScript(string(data))
			for _, al := range aliases {
				if !seen[al.Name] {
					al.Type = domain.TypeAlias
					allAliases = append(allAliases, al)
					seen[al.Name] = true
				}
			}
		}
	}

	// 3. Load from index folders
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
			fileManager := &Manager{filePath: match, shell: m.shell, functionsDir: m.functionsDir}
			aliases, err := fileManager.parseShellScript(string(data))
			if err == nil {
				for _, al := range aliases {
					if !seen[al.Name] {
						al.Type = domain.TypeAlias
						allAliases = append(allAliases, al)
						seen[al.Name] = true
					}
				}
			}
		}
	}

	return allAliases, nil
}

func (m *Manager) loadDirectoryFunctions(dirPath string, fnType domain.FunctionType) []domain.Alias {
	var results []domain.Alias
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return results
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if fnType == domain.TypeScript && strings.HasPrefix(name, "aliases.") {
			continue
		}
		filePath := filepath.Join(dirPath, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}
		item := m.parseFunctionFile(name, string(data), fnType, filePath)
		results = append(results, item)
	}
	return results
}

func (m *Manager) parseFunctionFile(filename string, content string, fnType domain.FunctionType, filePath string) domain.Alias {
	ext := filepath.Ext(filename)
	funcName := strings.TrimSuffix(filename, ext)
	description := ""
	aliasName := ""

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "# Description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "# Description:"))
		} else if strings.HasPrefix(line, "# Function:") {
			fnVal := strings.TrimSpace(strings.TrimPrefix(line, "# Function:"))
			if fnVal != "" {
				funcName = fnVal
			}
		} else if strings.HasPrefix(line, "# Script:") {
			scriptVal := strings.TrimSpace(strings.TrimPrefix(line, "# Script:"))
			if scriptVal != "" {
				funcName = scriptVal
			}
		} else if strings.HasPrefix(line, "# Alias:") {
			aliasName = strings.TrimSpace(strings.TrimPrefix(line, "# Alias:"))
		}
	}

	return domain.Alias{
		Name:        funcName,
		Value:       filePath,
		Description: description,
		SourceFile:  filePath,
		Type:        fnType,
		AliasName:   aliasName,
	}
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
				Type:        domain.TypeAlias,
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

	script.WriteString("# Generated by function manager\n")
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

//go:embed templates/global.tmpl
var globalFunctionTemplate string

//go:embed templates/lib.tmpl
var libFunctionTemplate string

//go:embed templates/scripts.tmpl
var scriptsFunctionTemplate string

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

// Add creates or updates an alias or function in its appropriate location.
func (m *Manager) Add(al domain.Alias) error {
	if !IsValidAlias(al.Name) {
		return ErrInvalidAlias
	}
	if IsReservedKeyword(al.Name) {
		return ErrReservedAlias
	}

	ext := shell.GetFileExtension(m.shell)

	switch al.Type {
	case domain.TypeGlobal:
		targetDir := filepath.Join(m.functionsDir, "global")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return err
		}
		targetFile := filepath.Join(targetDir, al.Name+ext)
		al.SourceFile = targetFile
		content := m.renderTemplate(globalFunctionTemplate, al)
		if err := os.WriteFile(targetFile, []byte(content), 0o755); err != nil {
			return err
		}
		if al.AliasName != "" {
			aliasItem := domain.Alias{
				Name:        al.AliasName,
				Value:       al.Name,
				Description: fmt.Sprintf("Alias for global function %s", al.Name),
				Type:        domain.TypeAlias,
			}
			_ = m.writeAliasToFile(m.filePath, aliasItem, false)
		}
		return nil

	case domain.TypeLib:
		targetDir := filepath.Join(m.functionsDir, "lib")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return err
		}
		targetFile := filepath.Join(targetDir, al.Name+ext)
		al.SourceFile = targetFile
		content := m.renderTemplate(libFunctionTemplate, al)
		return os.WriteFile(targetFile, []byte(content), 0o755)

	case domain.TypeScript:
		targetDir := filepath.Join(m.functionsDir, "scripts")
		if err := os.MkdirAll(targetDir, 0o755); err != nil {
			return err
		}
		targetFile := filepath.Join(targetDir, al.Name+ext)
		al.SourceFile = targetFile
		content := m.renderTemplate(scriptsFunctionTemplate, al)
		if err := os.WriteFile(targetFile, []byte(content), 0o755); err != nil {
			return err
		}

		aliasName := al.AliasName
		if aliasName == "" {
			aliasName = utils.GenerateAlias(al.Name, "", true, 1)
		}
		if aliasName != "" {
			aliasItem := domain.Alias{
				Name:        aliasName,
				Value:       targetFile,
				Description: fmt.Sprintf("Script alias for %s", al.Name),
				Type:        domain.TypeAlias,
			}
			_ = m.writeAliasToFile(m.filePath, aliasItem, false)
		}
		return nil

	default: // domain.TypeAlias or unspecified
		existing, err := m.Get(al.Name)
		targetFile := m.filePath
		if err == nil && existing.SourceFile != "" && (existing.Type == domain.TypeAlias || existing.Type == "") {
			targetFile = existing.SourceFile
		}
		al.SourceFile = targetFile
		return m.writeAliasToFile(targetFile, al, false)
	}
}

func (m *Manager) renderTemplate(tmplContent string, al domain.Alias) string {
	tmpl, err := template.New("function").Parse(tmplContent)
	if err != nil {
		return ""
	}
	var b strings.Builder
	upperName := strings.ToUpper(al.Name)
	re := regexp.MustCompile(`[^A-Z0-9_]`)
	upperName = re.ReplaceAllString(upperName, "_")

	data := struct {
		Name        string
		UpperName   string
		Description string
		AliasName   string
	}{
		Name:        al.Name,
		UpperName:   upperName,
		Description: al.Description,
		AliasName:   al.AliasName,
	}
	_ = tmpl.Execute(&b, data)
	return b.String()
}

// Get retrieves an alias or function by name.
func (m *Manager) Get(name string) (domain.Alias, error) {
	aliases, err := m.Load()
	if err != nil {
		return domain.Alias{}, err
	}

	for _, a := range aliases {
		if a.Name == name || (a.AliasName != "" && a.AliasName == name) {
			return a, nil
		}
	}

	return domain.Alias{}, ErrAliasNotFound
}

// Delete removes an alias or function.
func (m *Manager) Delete(name string) error {
	existing, err := m.Get(name)
	if err != nil {
		return err
	}

	if existing.SourceFile != "" && (existing.Type == domain.TypeGlobal || existing.Type == domain.TypeLib || existing.Type == domain.TypeScript) {
		_ = os.Remove(existing.SourceFile)
		if existing.AliasName != "" {
			_ = m.writeAliasToFile(m.filePath, domain.Alias{Name: existing.AliasName}, true)
		}
		return nil
	}

	targetFile := m.filePath
	if existing.SourceFile != "" {
		targetFile = existing.SourceFile
	}
	return m.writeAliasToFile(targetFile, existing, true)
}

// RunScript executes a script by name with given arguments.
func (m *Manager) RunScript(scriptName string, args []string) error {
	aliases, err := m.Load()
	if err == nil {
		for _, a := range aliases {
			if a.Type == domain.TypeScript && (a.Name == scriptName || a.AliasName == scriptName) {
				cmd := exec.Command(a.SourceFile, args...)
				cmd.Stdin = os.Stdin
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				return cmd.Run()
			}
		}
	}

	scriptsDir := filepath.Join(m.functionsDir, "scripts")
	matches, _ := filepath.Glob(filepath.Join(scriptsDir, scriptName+"*"))
	for _, match := range matches {
		base := filepath.Base(match)
		if strings.HasPrefix(base, "aliases.") {
			continue
		}
		if base == scriptName || strings.TrimSuffix(base, filepath.Ext(base)) == scriptName {
			cmd := exec.Command(match, args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		}
	}

	return fmt.Errorf("script '%s' not found", scriptName)
}

// Exists checks if an alias or function exists.
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
