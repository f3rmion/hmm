package views

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// FileSelectedMsg is sent when a file is selected
type FileSelectedMsg struct {
	Path string
}

// File picker styles
var (
	fpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B")).
			MarginBottom(1)

	fpPathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Italic(true).
			MarginBottom(1)

	fpDirStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ecdc4")).
			Bold(true)

	fpFileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f1faee"))

	fpSelectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffe66d")).
			Background(lipgloss.Color("#2d3436"))

	fpHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			MarginTop(1)

	fpErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6b6b")).
			Bold(true)
)

// FileEntry represents a file or directory
type FileEntry struct {
	Name  string
	IsDir bool
	Path  string
}

// FilePickerModel is the file picker view model.
type FilePickerModel struct {
	currentDir string
	entries    []FileEntry
	selected   int
	offset     int // For scrolling

	extensions []string // Filter to these extensions

	err error

	width  int
	height int
}

// NewFilePickerModel creates a new file picker model.
func NewFilePickerModel() FilePickerModel {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = "/"
	}

	// Default to ~/.config/hmm/anki if it exists
	ankiDir := filepath.Join(home, ".config", "hmm", "anki")
	startDir := ankiDir
	if _, err := os.Stat(ankiDir); os.IsNotExist(err) {
		startDir = home
	}

	m := FilePickerModel{
		currentDir: startDir,
		extensions: []string{".apkg"},
	}
	m.loadDir()
	return m
}

// SetSize updates the view dimensions.
func (m *FilePickerModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// loadDir loads the entries from the current directory
func (m *FilePickerModel) loadDir() {
	m.entries = nil
	m.selected = 0
	m.offset = 0
	m.err = nil

	entries, err := os.ReadDir(m.currentDir)
	if err != nil {
		m.err = err
		return
	}

	// Add parent directory entry
	if m.currentDir != "/" {
		m.entries = append(m.entries, FileEntry{
			Name:  "..",
			IsDir: true,
			Path:  filepath.Dir(m.currentDir),
		})
	}

	// Separate dirs and files
	var dirs, files []FileEntry

	for _, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		fe := FileEntry{
			Name:  entry.Name(),
			IsDir: entry.IsDir(),
			Path:  filepath.Join(m.currentDir, entry.Name()),
		}

		if entry.IsDir() {
			dirs = append(dirs, fe)
		} else {
			// Filter by extension
			if m.matchesExtension(entry.Name()) {
				files = append(files, fe)
			}
		}
	}

	// Sort alphabetically
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	// Dirs first, then files
	m.entries = append(m.entries, dirs...)
	m.entries = append(m.entries, files...)
}

func (m *FilePickerModel) matchesExtension(name string) bool {
	if len(m.extensions) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(name))
	for _, e := range m.extensions {
		if ext == strings.ToLower(e) {
			return true
		}
	}
	return false
}

// Update handles messages.
func (m FilePickerModel) Update(msg tea.Msg) (FilePickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selected < len(m.entries)-1 {
				m.selected++
				m.adjustScroll()
			}
			return m, nil
		case "k", "up":
			if m.selected > 0 {
				m.selected--
				m.adjustScroll()
			}
			return m, nil
		case "enter", "l", "right":
			if m.selected < len(m.entries) {
				entry := m.entries[m.selected]
				if entry.IsDir {
					m.currentDir = entry.Path
					m.loadDir()
				} else {
					// File selected
					return m, func() tea.Msg {
						return FileSelectedMsg{Path: entry.Path}
					}
				}
			}
			return m, nil
		case "backspace", "h":
			// Go to parent directory
			parent := filepath.Dir(m.currentDir)
			if parent != m.currentDir {
				m.currentDir = parent
				m.loadDir()
			}
			return m, nil
		case "~":
			// Go to home directory
			home, _ := os.UserHomeDir()
			if home != "" {
				m.currentDir = home
				m.loadDir()
			}
			return m, nil
		case "g":
			// Go to top
			m.selected = 0
			m.offset = 0
			return m, nil
		case "G":
			// Go to bottom
			m.selected = len(m.entries) - 1
			m.adjustScroll()
			return m, nil
		case "ctrl+d":
			// Page down
			visibleHeight := m.getVisibleHeight()
			m.selected += visibleHeight / 2
			if m.selected >= len(m.entries) {
				m.selected = len(m.entries) - 1
			}
			m.adjustScroll()
			return m, nil
		case "ctrl+u":
			// Page up
			visibleHeight := m.getVisibleHeight()
			m.selected -= visibleHeight / 2
			if m.selected < 0 {
				m.selected = 0
			}
			m.adjustScroll()
			return m, nil
		}
	}

	return m, nil
}

func (m *FilePickerModel) getVisibleHeight() int {
	h := m.height - 8 // Account for header, path, help
	if h < 5 {
		h = 5
	}
	return h
}

func (m *FilePickerModel) adjustScroll() {
	visibleHeight := m.getVisibleHeight()

	// Scroll up if selected is above viewport
	if m.selected < m.offset {
		m.offset = m.selected
	}

	// Scroll down if selected is below viewport
	if m.selected >= m.offset+visibleHeight {
		m.offset = m.selected - visibleHeight + 1
	}
}

// View renders the file picker.
func (m FilePickerModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(fpTitleStyle.Render("Select Anki Deck (.apkg)"))
	b.WriteString("\n")

	// Current path
	b.WriteString(fpPathStyle.Render(m.currentDir))
	b.WriteString("\n")

	// Error
	if m.err != nil {
		b.WriteString(fpErrorStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
	}

	// Separator
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#3d5a80")).Render(strings.Repeat("─", min(m.width-4, 60))))
	b.WriteString("\n")

	// File list
	visibleHeight := m.getVisibleHeight()
	start := m.offset
	end := start + visibleHeight
	if end > len(m.entries) {
		end = len(m.entries)
	}

	if len(m.entries) == 0 {
		b.WriteString(fpHelpStyle.Render("  (no .apkg files found)"))
		b.WriteString("\n")
	}

	for i := start; i < end; i++ {
		entry := m.entries[i]

		// Icon and name
		var icon, name string
		if entry.IsDir {
			icon = "[DIR]  "
			name = entry.Name
		} else {
			icon = "[FILE] "
			name = entry.Name
		}

		line := icon + name

		// Style based on selection and type
		var style lipgloss.Style
		if i == m.selected {
			style = fpSelectedStyle
		} else if entry.IsDir {
			style = fpDirStyle
		} else {
			style = fpFileStyle
		}

		// Prefix with > for selected
		prefix := "  "
		if i == m.selected {
			prefix = "> "
		}

		b.WriteString(prefix)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	// Scrollbar indicator
	if len(m.entries) > visibleHeight {
		scrollInfo := lipgloss.NewStyle().Foreground(lipgloss.Color("#666666")).Render(
			strings.Repeat(" ", 50) + "↕ scroll")
		b.WriteString(scrollInfo)
		b.WriteString("\n")
	}

	// Separator
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#3d5a80")).Render(strings.Repeat("─", min(m.width-4, 60))))
	b.WriteString("\n")

	// Help
	help := fpHelpStyle.Render("enter: select • backspace: parent • ~: home • esc: cancel")
	b.WriteString(help)

	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
