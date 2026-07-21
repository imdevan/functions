package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/functions/internal/adapters/editor"
	"github.com/functions/internal/adapters/icon"
	"github.com/functions/internal/adapters/tty"
	"github.com/functions/internal/alias"
	"github.com/functions/internal/config"
	"github.com/functions/internal/domain"
	"github.com/functions/internal/flags"
	pkg "github.com/functions/internal/package"
	"github.com/functions/internal/ui"
)

// Metadata loaded from package.toml at build time
var (
	version = pkg.Version()
	name    = pkg.Name()
	short   = pkg.Short()
)

type rootOptions struct {
	configPath  string
	showVersion bool
	interactive bool
	add         bool
	yes         bool
}

var rootCmd = newRootCmd()

// Execute is the CLI entrypoint.
func Execute() error {
	return rootCmd.Execute()
}

func newRootCmd() *cobra.Command {
	opts := &rootOptions{}
	cmd := &cobra.Command{
		Use:   name + " [name]",
		Short: short,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.showVersion {
				ver := resolvedVersion()
				cmd.Printf("%s\n", ver)
				return nil
			}

			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}

			cfg := config.Load(cwd, opts.configPath)

			// Interactive mode: if explicit -i flag, or if no args/commands are provided
			isNoArgsOrCommands := len(args) == 0 && !opts.add && !opts.yes

			if opts.interactive || isNoArgsOrCommands {
				return runInteractive(cmd, opts, cfg)
			}

			// Interactive add form
			if opts.add {
				return runAddForm(cmd, opts, cfg, cwd)
			}

			// Add alias mode
			return runAddAlias(cmd, args, opts, cfg, cwd)
		},
	}

	flags.SetPersistent(cmd, &opts.configPath, "config", "c", "config file path")

	flags.Set(cmd, &opts.interactive, "interactive", "i", "interactive alias browser")
	flags.Set(cmd, &opts.add, "add", "a", "interactive add alias form")
	flags.Set(cmd, &opts.yes, "yes", "y", "skip confirmation, and interactive prompts")

	flags.Set(cmd, &opts.showVersion, "version", "v", "print version information")

	cmd.AddCommand(newAddCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newEditCmd())
	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newCompletionCmd())

	return cmd
}

func resolvedVersion() string {
	ver := version
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ver
	}
	if ver == "dev" && strings.TrimSpace(info.Main.Version) != "" && info.Main.Version != "(devel)" {
		ver = info.Main.Version
	}
	return ver
}

func runAddAlias(cmd *cobra.Command, args []string, opts *rootOptions, cfg domain.Config, cwd string) error {
	aliasManager := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)

	nameArg := args[0]
	// Default value for a new alias is cd to current directory
	defaultValue := "cd " + cwd

	exists, err := aliasManager.Exists(nameArg)
	if err != nil {
		return err
	}

	if exists && !opts.yes && !confirmOverwrite(cmd, aliasManager, nameArg, cfg) {
		theme := ui.ThemeFromConfig(cfg)
		cmd.Println(ui.CanceledMessage(theme, "Overwrite"))
		return nil
	}

	al := domain.Alias{
		Name:  nameArg,
		Value: defaultValue,
	}

	if err := aliasManager.Add(al); err != nil {
		return err
	}

	action := "created"
	if exists {
		action = "updated"
	}
	printSuccess(cfg, action, nameArg, defaultValue)
	return nil
}

func confirmOverwrite(cmd *cobra.Command, aliasManager *alias.Manager, nameArg string, cfg domain.Config) bool {
	existing, _ := aliasManager.Get(nameArg)
	theme := ui.ThemeFromConfig(cfg)
	confirmModel := ui.NewConfirmationModel(
		"Overwrite Alias",
		fmt.Sprintf("Alias '%s → %s' already exists. Overwrite?", nameArg, existing.Value),
		theme,
	)

	p := tea.NewProgram(confirmModel, tea.WithoutSignalHandler())
	result, err := p.Run()
	if err != nil {
		return false
	}

	if confirmResult, ok := result.(ui.ConfirmationModel); ok {
		return confirmResult.ChoiceValue()
	}
	return false
}

func printSuccess(cfg domain.Config, action, aliasName, value string) {
	theme := ui.ThemeFromConfig(cfg)
	inline := action == "deleted"
	var body string
	if value != "" {
		body = fmt.Sprintf("%s → %s", aliasName, value)
	} else {
		body = aliasName
	}
	fmt.Println(ui.SuccessMessage(theme, action, body, inline))
}

func openEditor(editorName, filePath string, line int) error {
	if editorName == "" {
		return fmt.Errorf("no editor configured")
	}

	editorAdapter := editor.New(editorName)
	if line > 0 {
		return editorAdapter.OpenAtLine(filePath, line)
	}
	return editorAdapter.Open(filePath)
}

func runAddForm(cmd *cobra.Command, opts *rootOptions, cfg domain.Config, cwd string) error {
	aliasManager := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)
	defaultName := alias.GenerateAlias(cwd, cfg.AutoAliasSeparator, cfg.AutoAliasLowercase, cfg.DefaultAliasPartLength)

	theme := ui.ThemeFromConfig(cfg)
	m := ui.NewAliasFormModel(theme, defaultName, "cd "+cwd)

	initialAlias := domain.Alias{
		Name:  defaultName,
		Value: "cd " + cwd,
	}

	exists, err := aliasManager.Exists(defaultName)
	if err != nil {
		return err
	}

	return runAliasFormWorkflow(cmd, cfg, aliasManager, m, initialAlias, exists, false, "Add")
}

func runAliasFormWorkflow(
	cmd *cobra.Command,
	cfg domain.Config,
	aliasManager *alias.Manager,
	m ui.AliasFormModel,
	initialAlias domain.Alias,
	exists bool,
	isEdit bool,
	cancelAction string,
) error {
	progOpts := tty.GetProgramOptions(tea.WithoutSignalHandler())
	p := tea.NewProgram(m, progOpts...)
	result, err := p.Run()
	if err != nil {
		return err
	}

	fm, ok := result.(ui.AliasFormModel)
	if !ok || !fm.IsCompleted() {
		theme := ui.ThemeFromConfig(cfg)
		cmd.Println(ui.CanceledMessage(theme, cancelAction))
		return nil
	}

	fName, fValue, fDesc := fm.Values()

	// If the name changed and we are editing an existing one, delete the old one
	if exists && isEdit && fName != initialAlias.Name {
		if err := aliasManager.Delete(initialAlias.Name); err != nil {
			return err
		}
	}

	alreadyExists, err := aliasManager.Exists(fName)
	if err != nil {
		return err
	}

	newAl := domain.Alias{
		Name:        fName,
		Value:       fValue,
		Description: fDesc,
		SourceFile:  initialAlias.SourceFile,
	}

	if err := aliasManager.Add(newAl); err != nil {
		return err
	}

	action := "created"
	if alreadyExists {
		action = "updated"
	}
	printSuccess(cfg, action, fName, fValue)
	return nil
}

func runInteractive(cmd *cobra.Command, opts *rootOptions, cfg domain.Config) error {
	aliasManager := alias.NewManager(cfg.ResolvedAliasFile(), cfg.Shell, cfg.FunctionAlias, cfg.InteractiveAlias, cfg.IndexFolders)
	aliases, err := aliasManager.Load()
	if err != nil {
		return err
	}

	if len(aliases) == 0 {
		cmd.Println("No aliases found. Add one with: aliases [name]")
		return nil
	}

	return runAliasListing(aliases, cfg, aliasManager)
}

func sortAliases(aliases []domain.Alias, sortBy string) {
	switch sortBy {
	case "a-z", "A to Z":
		for i := 0; i < len(aliases)-1; i++ {
			for j := i + 1; j < len(aliases); j++ {
				if strings.ToLower(aliases[i].Name) > strings.ToLower(aliases[j].Name) {
					aliases[i], aliases[j] = aliases[j], aliases[i]
				}
			}
		}
	case "z-a", "Z to A":
		for i := 0; i < len(aliases)-1; i++ {
			for j := i + 1; j < len(aliases); j++ {
				if strings.ToLower(aliases[i].Name) < strings.ToLower(aliases[j].Name) {
					aliases[i], aliases[j] = aliases[j], aliases[i]
				}
			}
		}
	default:
		// Default is "newest" / natural order (no-op)
	}
}

func runAliasListing(aliases []domain.Alias, cfg domain.Config, aliasManager *alias.Manager) error {
	sortAliases(aliases, cfg.DefaultSortBy)

	items := make([]list.Item, 0, len(aliases))
	for _, al := range aliases {
		items = append(items, aliasItem{Alias: al, Config: cfg})
	}

	theme := ui.ThemeFromConfig(cfg)
	delegate := ui.NewListDelegate(theme, ui.ListDelegateOptions{
		Spacing:        cfg.ListSpacing,
		ShowMetadata:   false,
		MetadataIndent: 1,
	})

	listModel := ui.NewListModel(items, delegate, 80, 20, theme)
	listModel.Title = fmt.Sprintf("%s Aliases (%d)", icon.Bookmarks.String(), len(items))
	listModel.SetShowStatusBar(false)
	listModel.SetFilteringEnabled(true)

	model := aliasListModel{
		list:       listModel,
		theme:      theme,
		responsive: ui.NewResponsiveManager(80),
		manager:    aliasManager,
		config:     cfg,
	}

	model.list.AdditionalShortHelpKeys = model.getShortHelpKeys
	model.list.AdditionalFullHelpKeys = model.allHelpKeys

	opts := tty.GetProgramOptions(tea.WithoutSignalHandler())

	p := tea.NewProgram(model, opts...)
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run interactive list: %w", err)
	}

	return nil
}

type aliasItem struct {
	Alias  domain.Alias
	Config domain.Config
}

func (a aliasItem) Title() string {
	title := a.Alias.Name
	if a.Alias.Description != "" {
		mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(a.Config.Muted))
		title += " " + mutedStyle.Render("• "+a.Alias.Description)
	}
	return title
}

func (a aliasItem) Description() string {
	return a.Alias.Value
}

func (a aliasItem) FilterValue() string {
	return a.Alias.Name + " " + a.Alias.Value + " " + a.Alias.Description
}

type aliasListModel struct {
	list            list.Model
	theme           ui.Theme
	responsive      *ui.ResponsiveManager
	manager         *alias.Manager
	config          domain.Config
	message         string
	confirmMode     bool
	confirmModel    *ui.ConfirmationModel
	addMode         bool
	addModel        *ui.AliasFormModel
	editMode        bool
	editModel       *ui.AliasFormModel
	editingAlias    string
	pendingAction   string
	pendingItem     aliasItem
	pendingBookmark *domain.Alias
	screenW         int
	screenH         int
}

func (m *aliasListModel) updateTitle() {
	visibleCount := len(m.list.VisibleItems())
	totalCount := len(m.list.Items())

	if m.list.FilterState() == list.Filtering || m.list.FilterState() == list.FilterApplied {
		if visibleCount != totalCount {
			m.list.Title = fmt.Sprintf("%s Aliases (%d/%d)", icon.Bookmarks.String(), visibleCount, totalCount)
		} else {
			m.list.Title = fmt.Sprintf("%s Aliases (%d)", icon.Bookmarks.String(), totalCount)
		}
	} else {
		m.list.Title = fmt.Sprintf("%s Aliases (%d)", icon.Bookmarks.String(), totalCount)
	}
}

func (m aliasListModel) allHelpKeys() []key.Binding {
	if m.list.FilterState() == list.Filtering {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("enter"),
				key.WithHelp("enter", "run alias"),
			),
		}
	}

	return []key.Binding{
		key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "run alias"),
		),
		key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add alias"),
		),
		key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit alias"),
		),
		key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete alias"),
		),
		key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "force delete"),
		),
	}
}

func (m aliasListModel) getShortHelpKeys() []key.Binding {
	allKeys := m.allHelpKeys()

	var splitAt int
	switch m.responsive.Breakpoint() {
	case ui.BreakpointXL:
		splitAt = 2
	case ui.BreakpointLG:
		splitAt = 1
	default:
		splitAt = 1
	}

	return allKeys[:splitAt]
}

func (m aliasListModel) getFullHelpKeys() []key.Binding {
	allKeys := m.allHelpKeys()

	var splitAt int
	switch m.responsive.Breakpoint() {
	case ui.BreakpointXS:
		splitAt = 1
	case ui.BreakpointSM:
		splitAt = 1
	case ui.BreakpointMD:
		splitAt = 2
	default:
		return []key.Binding{}
	}

	return allKeys[splitAt:]
}

func (m aliasListModel) Init() tea.Cmd {
	return nil
}

func (m aliasListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.editMode && m.editModel != nil {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.screenW = msg.Width
			m.screenH = msg.Height
			m.responsive.SetWidth(msg.Width)
			width, height := m.responsive.GetListDimensions(msg.Width, msg.Height)
			m.list.SetSize(width, height)
			mw, mh := modalDimensions(msg.Width, msg.Height)
			updatedForm, _ := m.editModel.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
			if fm, ok := updatedForm.(ui.AliasFormModel); ok {
				m.editModel = &fm
			}
			return m, nil
		}

		var cmd tea.Cmd
		var updatedModel tea.Model
		updatedModel, cmd = m.editModel.Update(msg)
		if updatedForm, ok := updatedModel.(ui.AliasFormModel); ok {
			m.editModel = &updatedForm
		}

		if m.editModel.IsCompleted() {
			m.editMode = false
			nameInput, valueInput, descInput := m.editModel.Values()

			// Load old alias to preserve fields if needed
			var oldAl domain.Alias
			var loadErr error
			oldAl, loadErr = m.manager.Get(m.editingAlias)

			// If the name changed, delete the old one
			if m.editingAlias != nameInput {
				if err := m.manager.Delete(m.editingAlias); err != nil {
					m.message = fmt.Sprintf("✗ Failed to delete old alias: %s", err)
					m.editModel = nil
					return m, nil
				}
			}

			al := domain.Alias{
				Name:        nameInput,
				Value:       valueInput,
				Description: descInput,
			}
			if loadErr == nil {
				al.SourceFile = oldAl.SourceFile
			}

			if err := m.manager.Add(al); err != nil {
				m.message = fmt.Sprintf("✗ Failed to update: %s", err)
			} else {
				m.reloadList()
			}
			m.editModel = nil
			m.updateTitle()
			return m, nil
		} else if m.editModel.IsCancelled() {
			m.editMode = false
			m.editModel = nil
			return m, nil
		}

		return m, cmd
	}

	if m.addMode && m.addModel != nil {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			m.screenW = msg.Width
			m.screenH = msg.Height
			m.responsive.SetWidth(msg.Width)
			width, height := m.responsive.GetListDimensions(msg.Width, msg.Height)
			m.list.SetSize(width, height)
			mw, mh := modalDimensions(msg.Width, msg.Height)
			updatedForm, _ := m.addModel.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
			if fm, ok := updatedForm.(ui.AliasFormModel); ok {
				m.addModel = &fm
			}
			return m, nil
		}

		var cmd tea.Cmd
		var updatedModel tea.Model
		updatedModel, cmd = m.addModel.Update(msg)
		if updatedForm, ok := updatedModel.(ui.AliasFormModel); ok {
			m.addModel = &updatedForm
		}

		if m.addModel.IsCompleted() {
			m.addMode = false
			nameInput, valueInput, descInput := m.addModel.Values()
			al := domain.Alias{
				Name:        nameInput,
				Value:       valueInput,
				Description: descInput,
			}
			m.addModel = nil

			// Check for existing alias — show overwrite confirmation if needed
			exists, _ := m.manager.Exists(nameInput)
			if exists {
				m.pendingAction = "Overwrite"
				m.pendingBookmark = &al
				existing, _ := m.manager.Get(nameInput)
				confirmModel := ui.NewConfirmationModel(
					"Overwrite Alias",
					fmt.Sprintf("'%s → %s' exists. Overwrite?", nameInput, existing.Value),
					m.theme,
				)
				m.confirmModel = &confirmModel
				m.confirmMode = true
				return m, confirmModel.Init()
			}

			if err := m.manager.Add(al); err != nil {
				m.message = fmt.Sprintf("✗ Failed to add: %s", err)
			} else {
				m.reloadList()
			}
			m.updateTitle()
			return m, nil
		} else if m.addModel.IsCancelled() {
			m.addMode = false
			m.addModel = nil
			return m, nil
		}

		return m, cmd
	}

	if m.confirmMode && m.confirmModel != nil {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			updated, cmd := m.confirmModel.Update(msg)
			if updatedConfirm, ok := updated.(ui.ConfirmationModel); ok {
				m.confirmModel = &updatedConfirm
				if cmd != nil {
					if _, isQuit := cmd().(tea.QuitMsg); isQuit {
						confirmed := m.confirmModel.ChoiceValue()
						m.confirmMode = false
						if confirmed {
							return m.executeAction()
						} else {
							if m.pendingAction != "Delete" {
								m.message = fmt.Sprintf("%s cancelled", m.pendingAction)
							}
							m.pendingAction = ""
							return m, nil
						}
					}
				}
			}
			return m, cmd
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.screenW = msg.Width
		m.screenH = msg.Height
		m.responsive.SetWidth(msg.Width)
		width, height := m.responsive.GetListDimensions(msg.Width, msg.Height)
		m.list.SetSize(width, height)
		m.list.AdditionalShortHelpKeys = m.getShortHelpKeys
		return m, nil

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			switch msg.String() {
			case "q", "esc", "ctrl+c":
				return m, tea.Quit
			case "enter":
				if item, ok := m.list.SelectedItem().(aliasItem); ok {
					fmt.Println(item.Alias.Value)
					return m, tea.Quit
				}
			case "e", "n", "d", "D", "a":
				var cmd tea.Cmd
				m.list, cmd = m.list.Update(msg)
				return m, cmd
			}
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(aliasItem); ok {
				fmt.Println(item.Alias.Value)
				return m, tea.Quit
			}
		case "a":
			cwd, err := os.Getwd()
			if err != nil {
				m.message = "✗ Failed to get current directory"
				return m, nil
			}
			defaultName := alias.GenerateAlias(cwd, m.config.AutoAliasSeparator, m.config.AutoAliasLowercase, m.config.DefaultAliasPartLength)
			formModel := ui.NewAliasFormModel(m.theme, defaultName, "cd "+cwd)
			m.addModel = &formModel
			m.addMode = true

			if m.screenW > 0 {
				mw, mh := modalDimensions(m.screenW, m.screenH)
				updatedForm, _ := m.addModel.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
				if fm, ok := updatedForm.(ui.AliasFormModel); ok {
					m.addModel = &fm
				}
			}

			return m, formModel.Init()
		case "e":
			if item, ok := m.list.SelectedItem().(aliasItem); ok {
				formModel := ui.NewAliasFormModelEdit(m.theme, item.Alias)
				m.editModel = &formModel
				m.editMode = true
				m.editingAlias = item.Alias.Name

				if m.screenW > 0 {
					mw, mh := modalDimensions(m.screenW, m.screenH)
					updatedForm, _ := m.editModel.Update(tea.WindowSizeMsg{Width: mw, Height: mh})
					if fm, ok := updatedForm.(ui.AliasFormModel); ok {
						m.editModel = &fm
					}
				}

				return m, formModel.Init()
			}
		case "d":
			if item, ok := m.list.SelectedItem().(aliasItem); ok {
				if !m.config.ConfirmDelete {
					if err := m.manager.Delete(item.Alias.Name); err != nil {
						m.message = fmt.Sprintf("✗ Failed to delete: %s", err)
					} else {
						m.reloadList()
					}
					return m, nil
				}
				m.pendingAction = "Delete"
				m.pendingItem = item
				confirmModel := ui.NewConfirmationModel(
					"Delete Alias",
					fmt.Sprintf("Delete alias '%s'?", item.Alias.Name),
					m.theme,
				).WithTitleColor(m.theme.Error)
				m.confirmModel = &confirmModel
				m.confirmMode = true
				return m, confirmModel.Init()
			}
		case "D":
			if item, ok := m.list.SelectedItem().(aliasItem); ok {
				if err := m.manager.Delete(item.Alias.Name); err != nil {
					m.message = fmt.Sprintf("✗ Failed to delete: %s", err)
				} else {
					items := m.list.Items()
					filtered := make([]list.Item, 0, len(items))
					for _, listItem := range items {
						if aItem, ok := listItem.(aliasItem); ok {
							if aItem.Alias.Name != item.Alias.Name {
								filtered = append(filtered, aItem)
							}
						}
					}
					m.list.SetItems(filtered)
					m.updateTitle()
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	m.updateTitle()
	return m, cmd
}

func (m aliasListModel) View() string {
	listView := m.responsive.FullWidthFrameStyle(m.theme).Render(m.list.View())

	if m.message != "" {
		listView = listView + "\n\n" + m.message
	}

	if m.addMode && m.addModel != nil {
		return ui.Center(listView, m.addModel.View(), m.screenW, m.screenH)
	}

	if m.editMode && m.editModel != nil {
		return ui.Center(listView, m.editModel.View(), m.screenW, m.screenH)
	}

	if m.confirmMode && m.confirmModel != nil {
		return ui.Center(listView, m.confirmModel.View(), m.screenW, m.screenH)
	}

	return listView
}

func (m *aliasListModel) reloadList() {
	aliases, err := m.manager.Load()
	if err == nil {
		sortAliases(aliases, m.config.DefaultSortBy)
		items := make([]list.Item, 0, len(aliases))
		for _, b := range aliases {
			items = append(items, aliasItem{Alias: b, Config: m.config})
		}
		m.list.SetItems(items)
	}
}

func modalDimensions(screenW, screenH int) (width, height int) {
	const minW, minH = 80, 20
	width = screenW * 75 / 100
	if width < minW {
		width = minW
	}
	height = screenH * 80 / 100
	if height < minH {
		height = minH
	}
	return
}

func (m aliasListModel) executeAction() (tea.Model, tea.Cmd) {
	switch m.pendingAction {
	case "Delete":
		if err := m.manager.Delete(m.pendingItem.Alias.Name); err != nil {
			m.message = fmt.Sprintf("✗ Failed to delete: %s", err)
		} else {
			items := m.list.Items()
			filtered := make([]list.Item, 0, len(items))
			for _, item := range items {
				if aItem, ok := item.(aliasItem); ok {
					if aItem.Alias.Name != m.pendingItem.Alias.Name {
						filtered = append(filtered, item)
					}
				}
			}
			m.list.SetItems(filtered)
			m.updateTitle()
		}
	case "Overwrite":
		if m.pendingBookmark != nil {
			if err := m.manager.Add(*m.pendingBookmark); err != nil {
				m.message = fmt.Sprintf("✗ Failed to overwrite: %s", err)
			} else {
				m.message = fmt.Sprintf("✓ Overwritten: %s", m.pendingBookmark.Name)
				m.reloadList()
				m.updateTitle()
			}
			m.pendingBookmark = nil
		}
	}
	m.pendingAction = ""
	return m, nil
}
