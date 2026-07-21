package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	al "github.com/funcs/internal/alias"
	"github.com/funcs/internal/domain"
)

const (
	formName = iota
	formValue
	formDesc
)

var fieldTitles = []string{
	"Name",
	"Value",
	"Description (optional)",
}

var fieldDescs = []string{
	"Short name for the alias",
	"Command or script that the alias executes",
	"",
}

// AliasFormModel is a form for creating/editing an alias.
type AliasFormModel struct {
	inputs           []textinput.Model
	focused          int
	theme            Theme
	responsive       *ResponsiveManager
	completed        bool
	cancelled        bool
	title            string
	validationErrors map[int]string
}

// WithTitle sets a custom title for the form model.
func (m AliasFormModel) WithTitle(title string) AliasFormModel {
	m.title = title
	return m
}

// NewAliasFormModel creates a new alias form with optional default values.
func NewAliasFormModel(theme Theme, defaultName, defaultValue string) AliasFormModel {
	inputs := make([]textinput.Model, 3)

	inputs[formName] = textinput.New()
	inputs[formName].Placeholder = defaultName
	inputs[formName].Focus()
	inputs[formName].Prompt = ""

	inputs[formValue] = textinput.New()
	inputs[formValue].Placeholder = defaultValue
	inputs[formValue].Prompt = ""

	inputs[formDesc] = textinput.New()
	inputs[formDesc].Placeholder = "Optional description"
	inputs[formDesc].Prompt = ""

	return AliasFormModel{
		inputs:           inputs,
		focused:          0,
		theme:            theme,
		responsive:       NewResponsiveManager(80),
		title:            "Add Alias",
		validationErrors: make(map[int]string),
	}
}

// NewAliasFormModelEdit creates a new alias form prefilled with existing values.
func NewAliasFormModelEdit(theme Theme, alias domain.Alias) AliasFormModel {
	inputs := make([]textinput.Model, 3)

	inputs[formName] = textinput.New()
	inputs[formName].Placeholder = alias.Name
	inputs[formName].SetValue(alias.Name)
	inputs[formName].Focus()
	inputs[formName].Prompt = ""

	inputs[formValue] = textinput.New()
	inputs[formValue].Placeholder = alias.Value
	inputs[formValue].SetValue(alias.Value)
	inputs[formValue].Prompt = ""

	inputs[formDesc] = textinput.New()
	inputs[formDesc].Placeholder = "Optional description"
	inputs[formDesc].SetValue(alias.Description)
	inputs[formDesc].Prompt = ""

	return AliasFormModel{
		inputs:           inputs,
		focused:          0,
		theme:            theme,
		responsive:       NewResponsiveManager(80),
		title:            "Edit Alias",
		validationErrors: make(map[int]string),
	}
}

// Init initializes the form.
func (m AliasFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *AliasFormModel) validateField(index int) bool {
	delete(m.validationErrors, index)

	val := m.inputs[index].Value()
	if val == "" {
		val = m.inputs[index].Placeholder
	}
	val = strings.TrimSpace(val)

	switch index {
	case formName:
		if val == "" {
			m.validationErrors[formName] = "* Name cannot be empty"
			return false
		}
		if !al.IsValidAlias(val) {
			m.validationErrors[formName] = "* Name must contain only alphanumeric characters, hyphens, and underscores"
			return false
		}
		if al.IsReservedKeyword(val) {
			m.validationErrors[formName] = "* Name cannot be a shell reserved keyword"
			return false
		}
	case formValue:
		if val == "" {
			m.validationErrors[formValue] = "* Value cannot be empty"
			return false
		}
	}
	return true
}

func (m *AliasFormModel) validateAll() bool {
	valid := true
	for i := range m.inputs {
		if !m.validateField(i) {
			valid = false
		}
	}
	return valid
}

// Update handles messages for the form.
func (m AliasFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.responsive.SetWidth(msg.Width)
		inputWidth := m.responsive.MaxContentWidth() - 6
		if inputWidth < 20 {
			inputWidth = 20
		}
		for i := range m.inputs {
			m.inputs[i].Width = inputWidth
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit

		case "enter":
			if !m.validateField(m.focused) {
				return m, nil
			}
			if m.focused == len(m.inputs)-1 {
				if m.validateAll() {
					m.completed = true
					return m, tea.Quit
				}
				for i := range m.inputs {
					if _, exists := m.validationErrors[i]; exists {
						m.inputs[m.focused].Blur()
						m.focused = i
						m.inputs[m.focused].Focus()
						break
					}
				}
				return m, nil
			}
			m.inputs[m.focused].Blur()
			m.focused++
			m.inputs[m.focused].Focus()
			return m, nil

		case "alt+enter":
			if m.validateAll() {
				m.completed = true
				return m, tea.Quit
			}
			for i := range m.inputs {
				if _, exists := m.validationErrors[i]; exists {
					m.inputs[m.focused].Blur()
					m.focused = i
					m.inputs[m.focused].Focus()
					break
				}
			}
			return m, nil

		case "shift+tab", "up":
			if m.focused > 0 {
				m.validateField(m.focused)
				m.inputs[m.focused].Blur()
				m.focused--
				m.inputs[m.focused].Focus()
			}
			return m, nil

		case "tab", "down":
			if m.focused < len(m.inputs)-1 {
				m.validateField(m.focused)
				m.inputs[m.focused].Blur()
				m.focused++
				m.inputs[m.focused].Focus()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.inputs[m.focused], cmd = m.inputs[m.focused].Update(msg)
	return m, cmd
}

// View renders the form.
func (m AliasFormModel) View() string {
	if m.completed || m.cancelled {
		return ""
	}

	titleText := m.title
	if titleText == "" {
		titleText = "Create New Alias"
	}
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Headings).
		Render(titleText)

	help := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Render("↑/↓ or tab/shift+tab: navigate • enter: next • alt+enter: submit • esc: cancel")

	var items []string
	for i := range m.inputs {
		var itemLines []string

		titleStyle := lipgloss.NewStyle().Bold(true)
		if i == m.focused {
			titleStyle = titleStyle.Foreground(m.theme.Secondary)
		} else {
			titleStyle = titleStyle.Foreground(m.theme.Text)
		}
		itemLines = append(itemLines, titleStyle.Render(fieldTitles[i]))

		if errMsg, exists := m.validationErrors[i]; exists && errMsg != "" {
			errStyle := lipgloss.NewStyle().
				Foreground(m.theme.Error).
				Bold(true)
			itemLines = append(itemLines, errStyle.Render(errMsg))
		}

		if fieldDescs[i] != "" {
			descStyle := lipgloss.NewStyle().Foreground(m.theme.Muted)
			itemLines = append(itemLines, descStyle.Render(fieldDescs[i]))
		}

		borderWidth := m.responsive.MaxContentWidth() - 2
		if borderWidth < 22 {
			borderWidth = 22
		}
		inputStyle := lipgloss.NewStyle().
			Padding(0, 1).
			Width(borderWidth).
			BorderStyle(lipgloss.RoundedBorder())

		if _, exists := m.validationErrors[i]; exists {
			inputStyle = inputStyle.BorderForeground(m.theme.Error)
		} else if i == m.focused {
			inputStyle = inputStyle.BorderForeground(m.theme.Secondary)
		} else {
			inputStyle = inputStyle.BorderForeground(m.theme.Border)
		}

		inputBox := inputStyle.Render(m.inputs[i].View())
		itemLines = append(itemLines, inputBox)

		items = append(items, strings.Join(itemLines, "\n"))
	}

	content := strings.Join([]string{
		title,
		"",
		strings.Join(items, "\n\n"),
		"",
		help,
	}, "\n")

	return m.responsive.AdaptiveFrameStyle(m.theme).Render(content)
}

// Values returns the form values.
func (m AliasFormModel) Values() (name, value, desc string) {
	name = m.inputs[formName].Value()
	if name == "" {
		name = m.inputs[formName].Placeholder
	}
	value = m.inputs[formValue].Value()
	if value == "" {
		value = m.inputs[formValue].Placeholder
	}
	desc = m.inputs[formDesc].Value()

	return strings.TrimSpace(name),
		strings.TrimSpace(value),
		strings.TrimSpace(desc)
}

// IsCompleted returns true if the form was completed successfully.
func (m AliasFormModel) IsCompleted() bool {
	return m.completed && !m.cancelled
}

// IsCancelled returns true if the form was cancelled.
func (m AliasFormModel) IsCancelled() bool {
	return m.cancelled
}
