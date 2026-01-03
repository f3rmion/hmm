package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/f3rmion/hmm/internal/anki"
	"github.com/f3rmion/hmm/internal/config"
	"github.com/f3rmion/hmm/internal/decomp"
	"github.com/f3rmion/hmm/internal/llm"
	"github.com/f3rmion/hmm/internal/pinyin"
	"github.com/f3rmion/hmm/internal/prompt"
	"github.com/f3rmion/hmm/internal/tui/views"
)

// ViewType represents the current active view
type ViewType int

const (
	ViewLookup ViewType = iota
	ViewBrowse
	ViewLearn
	ViewFilePicker
	ViewSettings
)

// MenuItem represents a sidebar menu entry
type MenuItem struct {
	Label    string
	Icon     string
	View     ViewType
	Shortcut string
}

// ViewSwitchMsg requests a view change
type ViewSwitchMsg struct {
	View ViewType
}

// FileSelectedMsg is sent when a file is selected in the file picker
type FileSelectedMsg struct {
	Path string
}

// PackageLoadedMsg is sent when an Anki package is loaded
type PackageLoadedMsg struct {
	Package *anki.Package
	Path    string
	Err     error
}

// AppModel is the main unified TUI model
type AppModel struct {
	// Core dependencies
	dict      *decomp.Dictionary
	config    *config.Config
	llmClient *llm.Client
	parser    *pinyin.Parser
	generator *prompt.Generator

	// Layout state
	width        int
	height       int
	sidebarWidth int
	ready        bool

	// Navigation
	currentView   ViewType
	menuItems     []MenuItem
	selectedMenu  int
	sidebarActive bool

	// Sub-models (views)
	lookupView     views.LookupModel
	browseView     views.BrowseModel
	learnView      views.LearnModel
	filePickerView views.FilePickerModel
	settingsView   views.SettingsModel

	// Loaded Anki package
	ankiPackage *anki.Package
	ankiPath    string

	// Help overlay
	showHelp bool
}

// NewApp creates a new unified TUI application
func NewApp(dict *decomp.Dictionary, cfg *config.Config) AppModel {
	var gen *prompt.Generator
	if cfg != nil {
		gen = prompt.NewGenerator(cfg.Actors, cfg.Sets, cfg.Props)
	} else {
		gen = prompt.NewGenerator(nil, nil, nil)
	}

	llmClient, _ := llm.NewClient()

	menuItems := []MenuItem{
		{Label: "Lookup", Icon: "字", View: ViewLookup, Shortcut: "1"},
		{Label: "Browse", Icon: "卡", View: ViewBrowse, Shortcut: "2"},
		{Label: "Learn", Icon: "學", View: ViewLearn, Shortcut: "3"},
		{Label: "Open Deck", Icon: "開", View: ViewFilePicker, Shortcut: "4"},
		{Label: "Settings", Icon: "設", View: ViewSettings, Shortcut: "5"},
	}

	app := AppModel{
		dict:         dict,
		config:       cfg,
		llmClient:    llmClient,
		parser:       pinyin.NewParser(),
		generator:    gen,
		sidebarWidth: 18,
		currentView:  ViewLookup,
		menuItems:    menuItems,
		sidebarActive: false,

		lookupView:     views.NewLookupModel(dict, cfg, gen, llmClient),
		browseView:     views.NewBrowseModel(dict, cfg, gen, llmClient),
		learnView:      views.NewLearnModel(dict, cfg, gen, llmClient),
		filePickerView: views.NewFilePickerModel(),
		settingsView:   views.NewSettingsModel(cfg),
	}

	return app
}

// NewAppWithPackage creates a new app with a pre-loaded Anki package
func NewAppWithPackage(dict *decomp.Dictionary, cfg *config.Config, pkg *anki.Package, path string) AppModel {
	app := NewApp(dict, cfg)
	app.ankiPackage = pkg
	app.ankiPath = path
	app.browseView.SetPackage(pkg)
	app.learnView.SetPackage(pkg)
	app.currentView = ViewBrowse
	app.selectedMenu = 1 // Browse
	return app
}

// Init initializes the model
func (m AppModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Help overlay - any key closes it
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}

		// Global keys
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?":
			m.showHelp = true
			return m, nil
		case "esc":
			// Esc goes back to sidebar or quits
			if m.sidebarActive {
				return m, tea.Quit
			}
			m.sidebarActive = true
			return m, nil
		case "1":
			m.currentView = ViewLookup
			m.selectedMenu = 0
			m.sidebarActive = false
			return m, nil
		case "2":
			m.currentView = ViewBrowse
			m.selectedMenu = 1
			m.sidebarActive = false
			return m, nil
		case "3":
			m.currentView = ViewLearn
			m.selectedMenu = 2
			m.sidebarActive = false
			return m, nil
		case "4":
			m.currentView = ViewFilePicker
			m.selectedMenu = 3
			m.sidebarActive = false
			return m, nil
		case "5":
			m.currentView = ViewSettings
			m.selectedMenu = 4
			m.sidebarActive = false
			return m, nil
		case "tab":
			m.sidebarActive = !m.sidebarActive
			return m, nil
		}

		// Sidebar navigation when active
		if m.sidebarActive {
			switch msg.String() {
			case "j", "down":
				if m.selectedMenu < len(m.menuItems)-1 {
					m.selectedMenu++
				}
				return m, nil
			case "k", "up":
				if m.selectedMenu > 0 {
					m.selectedMenu--
				}
				return m, nil
			case "enter", "l", "right":
				m.currentView = m.menuItems[m.selectedMenu].View
				m.sidebarActive = false
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

		// Update view sizes
		contentWidth := m.width - m.sidebarWidth - 4
		contentHeight := m.height - 2

		m.lookupView.SetSize(contentWidth, contentHeight)
		m.browseView.SetSize(contentWidth, contentHeight)
		m.learnView.SetSize(contentWidth, contentHeight)
		m.filePickerView.SetSize(contentWidth, contentHeight)
		m.settingsView.SetSize(contentWidth, contentHeight)

		return m, nil

	case ViewSwitchMsg:
		m.currentView = msg.View
		for i, item := range m.menuItems {
			if item.View == msg.View {
				m.selectedMenu = i
				break
			}
		}
		return m, nil

	case FileSelectedMsg:
		// Load the Anki package
		return m, m.loadAnkiPackage(msg.Path)

	case views.FileSelectedMsg:
		// Load the Anki package (from file picker view)
		return m, m.loadAnkiPackage(msg.Path)

	case PackageLoadedMsg:
		if msg.Err == nil && msg.Package != nil {
			m.ankiPackage = msg.Package
			m.ankiPath = msg.Path
			m.browseView.SetPackage(msg.Package)
			m.learnView.SetPackage(msg.Package)
			m.currentView = ViewBrowse
			m.selectedMenu = 1
		}
		return m, nil
	}

	// Delegate to active view if not in sidebar mode
	if !m.sidebarActive {
		var cmd tea.Cmd
		switch m.currentView {
		case ViewLookup:
			m.lookupView, cmd = m.lookupView.Update(msg)
		case ViewBrowse:
			m.browseView, cmd = m.browseView.Update(msg)
		case ViewLearn:
			m.learnView, cmd = m.learnView.Update(msg)
		case ViewFilePicker:
			m.filePickerView, cmd = m.filePickerView.Update(msg)
		case ViewSettings:
			m.settingsView, cmd = m.settingsView.Update(msg)
		}
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the UI
func (m AppModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Show help overlay if active
	if m.showHelp {
		return m.renderHelp()
	}

	// Render sidebar
	sidebar := m.renderSidebar()

	// Render main content based on current view
	var content string
	switch m.currentView {
	case ViewLookup:
		content = m.lookupView.View()
	case ViewBrowse:
		content = m.browseView.View()
	case ViewLearn:
		content = m.learnView.View()
	case ViewFilePicker:
		content = m.filePickerView.View()
	case ViewSettings:
		content = m.settingsView.View()
	}

	// Apply content styling
	contentWidth := m.width - m.sidebarWidth - 4
	mainContent := ContentStyle.
		Width(contentWidth).
		Height(m.height - 2).
		Render(content)

	// Join horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, mainContent)
}

// renderSidebar renders the sidebar navigation
func (m AppModel) renderSidebar() string {
	var items []string

	// Title
	title := SidebarTitleStyle.Render("  漢字 HMM  ")
	items = append(items, title)
	items = append(items, "")

	// Menu items
	for i, item := range m.menuItems {
		label := item.Shortcut + ". " + item.Label

		var style lipgloss.Style
		if i == m.selectedMenu {
			if m.sidebarActive {
				style = SidebarItemActiveStyle
			} else {
				// Indicate current view but not focused
				style = SidebarItemStyle.Bold(true).Foreground(ColorSecondary)
			}
		} else {
			style = SidebarItemStyle
		}

		items = append(items, style.Render(label))
	}

	// Spacer
	usedHeight := len(items) + 4 // account for borders and help
	if m.height > usedHeight {
		for i := 0; i < m.height-usedHeight-2; i++ {
			items = append(items, "")
		}
	}

	// Help text at bottom
	help := SidebarHelpStyle.Render("? Help  q Quit")
	items = append(items, help)

	content := lipgloss.JoinVertical(lipgloss.Left, items...)

	return SidebarStyle.
		Width(m.sidebarWidth).
		Height(m.height - 2).
		Render(content)
}

// loadAnkiPackage loads an Anki package asynchronously
func (m AppModel) loadAnkiPackage(path string) tea.Cmd {
	return func() tea.Msg {
		pkg, err := anki.OpenPackage(path)
		return PackageLoadedMsg{Package: pkg, Path: path, Err: err}
	}
}

// renderHelp renders the help overlay
func (m AppModel) renderHelp() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF6B6B")).
		MarginBottom(1)

	sectionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#4ECDC4")).
		MarginTop(1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFE66D")).
		Width(12)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#F1FAEE"))

	helpText := titleStyle.Render("HMM - Hanzi Movie Method") + "\n\n"

	helpText += sectionStyle.Render("Global Keys") + "\n"
	helpText += keyStyle.Render("1-5") + descStyle.Render("Switch views") + "\n"
	helpText += keyStyle.Render("tab") + descStyle.Render("Toggle sidebar focus") + "\n"
	helpText += keyStyle.Render("?") + descStyle.Render("Show this help") + "\n"
	helpText += keyStyle.Render("q") + descStyle.Render("Quit") + "\n"

	helpText += sectionStyle.Render("Lookup View") + "\n"
	helpText += keyStyle.Render("enter") + descStyle.Render("Analyze character(s)") + "\n"
	helpText += keyStyle.Render("g") + descStyle.Render("Generate LLM prompt") + "\n"
	helpText += keyStyle.Render("y") + descStyle.Render("Copy prompt to clipboard") + "\n"
	helpText += keyStyle.Render("←/→") + descStyle.Render("Navigate characters") + "\n"

	helpText += sectionStyle.Render("Browse View") + "\n"
	helpText += keyStyle.Render("j/k ↑/↓") + descStyle.Render("Navigate cards") + "\n"
	helpText += keyStyle.Render("←/→") + descStyle.Render("Navigate characters") + "\n"
	helpText += keyStyle.Render("/") + descStyle.Render("Search") + "\n"
	helpText += keyStyle.Render("g") + descStyle.Render("Generate prompt") + "\n"
	helpText += keyStyle.Render("B") + descStyle.Render("Batch generate all") + "\n"

	helpText += sectionStyle.Render("Learn View") + "\n"
	helpText += keyStyle.Render("space") + descStyle.Render("Flip card") + "\n"
	helpText += keyStyle.Render("←/→") + descStyle.Render("Prev/next card") + "\n"
	helpText += keyStyle.Render("r") + descStyle.Render("Reset to first card") + "\n"

	helpText += sectionStyle.Render("File Picker") + "\n"
	helpText += keyStyle.Render("enter") + descStyle.Render("Select file/enter dir") + "\n"
	helpText += keyStyle.Render("backspace") + descStyle.Render("Go to parent dir") + "\n"
	helpText += keyStyle.Render("~") + descStyle.Render("Go to home dir") + "\n"

	helpText += "\n" + lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true).
		Render("Press any key to close")

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#4ECDC4")).
		Padding(1, 2).
		Width(50)

	// Center the help box
	helpBox := boxStyle.Render(helpText)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, helpBox)
}
