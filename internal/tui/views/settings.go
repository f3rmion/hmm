package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/f3rmion/hmm/internal/config"
)

// Settings view styles
var (
	settingsTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF6B6B")).
				MarginBottom(1)

	settingsPathStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666")).
				Italic(true).
				MarginBottom(1)

	settingsTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Padding(0, 2)

	settingsTabActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffe66d")).
				Background(lipgloss.Color("#2d3436")).
				Padding(0, 2)

	settingsHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#a8dadc"))

	settingsRowStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f1faee"))

	settingsMutedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666"))

	settingsHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666")).
				MarginTop(1)
)

// SettingsModel is the settings view model.
type SettingsModel struct {
	config    *config.Config
	configDir string

	// Tabs: 0=Actors, 1=Sets, 2=Props
	tab     int
	scrollY int

	width  int
	height int
}

// NewSettingsModel creates a new settings model.
func NewSettingsModel(cfg *config.Config) SettingsModel {
	configDir := os.Getenv("HOME") + "/.config/hmm"
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		configDir = filepath.Join(xdg, "hmm")
	}

	return SettingsModel{
		config:    cfg,
		configDir: configDir,
	}
}

// SetSize updates the view dimensions.
func (m *SettingsModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages.
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "right", "l":
			m.tab = (m.tab + 1) % 3
			m.scrollY = 0
			return m, nil
		case "shift+tab", "left", "h":
			m.tab--
			if m.tab < 0 {
				m.tab = 2
			}
			m.scrollY = 0
			return m, nil
		case "j", "down":
			m.scrollY++
			return m, nil
		case "k", "up":
			if m.scrollY > 0 {
				m.scrollY--
			}
			return m, nil
		case "g":
			m.scrollY = 0
			return m, nil
		}
	}
	return m, nil
}

// View renders the settings view.
func (m SettingsModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(settingsTitleStyle.Render("HMM Configuration"))
	b.WriteString("\n")

	// Config path
	b.WriteString(settingsPathStyle.Render("Config: " + m.configDir))
	b.WriteString("\n\n")

	// Tabs
	tabs := []string{"Actors", "Sets", "Props"}
	var tabViews []string
	for i, t := range tabs {
		var style lipgloss.Style
		if i == m.tab {
			style = settingsTabActiveStyle
		} else {
			style = settingsTabStyle
		}
		tabViews = append(tabViews, style.Render(t))
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabViews...))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#3d5a80")).Render(strings.Repeat("─", minInt(m.width-4, 60))))
	b.WriteString("\n\n")

	// Content based on tab
	switch m.tab {
	case 0:
		b.WriteString(m.renderActors())
	case 1:
		b.WriteString(m.renderSets())
	case 2:
		b.WriteString(m.renderProps())
	}

	// Help
	b.WriteString("\n")
	b.WriteString(settingsHelpStyle.Render("tab/←→: switch tabs • j/k: scroll"))

	return b.String()
}

func (m SettingsModel) renderActors() string {
	var b strings.Builder

	if m.config == nil || len(m.config.Actors) == 0 {
		b.WriteString(settingsMutedStyle.Render("No actors configured"))
		b.WriteString("\n")
		b.WriteString(settingsMutedStyle.Render("Run 'hmm init' to create config files"))
		return b.String()
	}

	b.WriteString(settingsHeaderStyle.Render(fmt.Sprintf("Actors (%d configured)", len(m.config.Actors))))
	b.WriteString("\n\n")

	// Header row
	headerFmt := "%-6s %-12s %-15s %s"
	header := fmt.Sprintf(headerFmt, "ID", "Initial", "Category", "Name")
	b.WriteString(settingsMutedStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(settingsMutedStyle.Render(strings.Repeat("─", 50)))
	b.WriteString("\n")

	// Calculate visible range
	visibleHeight := m.height - 12
	if visibleHeight < 5 {
		visibleHeight = 5
	}
	start := m.scrollY
	end := start + visibleHeight
	if end > len(m.config.Actors) {
		end = len(m.config.Actors)
	}
	if start > len(m.config.Actors) {
		start = 0
	}

	// Actor rows
	for i := start; i < end; i++ {
		a := m.config.Actors[i]
		initial := a.Initial
		if initial == "" {
			initial = "(null)"
		}
		row := fmt.Sprintf("%-6s %-12s %-15s %s", a.ID, initial, a.Category, a.Name)
		b.WriteString(settingsRowStyle.Render(row))
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.config.Actors) > visibleHeight {
		b.WriteString("\n")
		b.WriteString(settingsMutedStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.config.Actors))))
	}

	return b.String()
}

func (m SettingsModel) renderSets() string {
	var b strings.Builder

	if m.config == nil || len(m.config.Sets) == 0 {
		b.WriteString(settingsMutedStyle.Render("No sets configured"))
		b.WriteString("\n")
		b.WriteString(settingsMutedStyle.Render("Run 'hmm init' to create config files"))
		return b.String()
	}

	b.WriteString(settingsHeaderStyle.Render(fmt.Sprintf("Sets (%d configured)", len(m.config.Sets))))
	b.WriteString("\n\n")

	// Table styles
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#4ecdc4"))
	idStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffe66d")).Width(6)
	finalStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b")).Width(6)
	nameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f1faee"))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Italic(true)
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#3d5a80"))

	// Header row
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-6s %-6s %s", "Final", "ID", "Name / Description")))
	b.WriteString("\n")
	b.WriteString(borderStyle.Render(strings.Repeat("─", 60)))
	b.WriteString("\n")

	// Calculate visible range (3 lines per set: name + description + tones)
	visibleHeight := m.height - 14
	if visibleHeight < 9 {
		visibleHeight = 9
	}
	visibleSets := visibleHeight / 3
	start := m.scrollY
	end := start + visibleSets
	if end > len(m.config.Sets) {
		end = len(m.config.Sets)
	}
	if start >= len(m.config.Sets) {
		start = 0
		end = visibleSets
		if end > len(m.config.Sets) {
			end = len(m.config.Sets)
		}
	}

	// Tone label styles
	toneMarkStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b")).Bold(true)
	toneNameStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#a8e6cf"))

	// Set rows
	for i := start; i < end; i++ {
		s := m.config.Sets[i]
		final := s.Final
		if final == "" {
			final = "Ø"
		}
		id := s.ID
		if id == "" {
			id = "Ø"
		}

		// First line: Final, ID, Name
		b.WriteString(finalStyle.Render(final))
		b.WriteString(idStyle.Render(id))
		b.WriteString(nameStyle.Render(s.Name))
		b.WriteString("\n")

		// Second line: Description (indented)
		if s.Description != "" {
			b.WriteString("            ")
			b.WriteString(descStyle.Render(s.Description))
			b.WriteString("\n")
		}

		// Third line: Tones
		if len(s.Rooms) > 0 {
			b.WriteString("            ")
			for j, room := range s.Rooms {
				if j > 0 {
					b.WriteString("  ")
				}
				// Apply tone mark to the final
				tonedFinal := applyToneMark(s.Final, int(room.Tone))
				b.WriteString(toneMarkStyle.Render(tonedFinal + ":"))
				b.WriteString(toneNameStyle.Render(room.Name))
			}
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(m.config.Sets) > visibleSets {
		b.WriteString("\n")
		b.WriteString(settingsMutedStyle.Render(fmt.Sprintf("Showing %d-%d of %d (j/k to scroll)", start+1, end, len(m.config.Sets))))
	}

	return b.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// applyToneMark adds a tone mark to a pinyin final
func applyToneMark(final string, tone int) string {
	if final == "" {
		// Null initial - just return tone marker on 'a'
		tones := map[int]string{1: "ā", 2: "á", 3: "ǎ", 4: "à", 5: "a"}
		if m, ok := tones[tone]; ok {
			return m
		}
		return "?"
	}

	// Tone mark mappings for each vowel
	toneMap := map[rune][]rune{
		'a': {'ā', 'á', 'ǎ', 'à', 'a'},
		'e': {'ē', 'é', 'ě', 'è', 'e'},
		'i': {'ī', 'í', 'ǐ', 'ì', 'i'},
		'o': {'ō', 'ó', 'ǒ', 'ò', 'o'},
		'u': {'ū', 'ú', 'ǔ', 'ù', 'u'},
		'ü': {'ǖ', 'ǘ', 'ǚ', 'ǜ', 'ü'},
	}

	// Find which vowel to mark (pinyin rules)
	// 1. 'a' or 'e' always gets the mark
	// 2. In 'ou', 'o' gets the mark
	// 3. Otherwise, the last vowel gets the mark
	runes := []rune(final)
	markIndex := -1

	for i, r := range runes {
		if r == 'a' || r == 'e' {
			markIndex = i
			break
		}
	}

	if markIndex == -1 {
		// Check for 'ou'
		for i, r := range runes {
			if r == 'o' && i+1 < len(runes) && runes[i+1] == 'u' {
				markIndex = i
				break
			}
		}
	}

	if markIndex == -1 {
		// Find last vowel
		for i := len(runes) - 1; i >= 0; i-- {
			if _, isVowel := toneMap[runes[i]]; isVowel {
				markIndex = i
				break
			}
		}
	}

	if markIndex == -1 || tone < 1 || tone > 5 {
		return final
	}

	// Apply the tone mark
	vowel := runes[markIndex]
	if tones, ok := toneMap[vowel]; ok {
		runes[markIndex] = tones[tone-1]
	}

	return string(runes)
}

func (m SettingsModel) renderProps() string {
	var b strings.Builder

	if m.config == nil || len(m.config.Props) == 0 {
		b.WriteString(settingsMutedStyle.Render("No props configured"))
		b.WriteString("\n")
		b.WriteString(settingsMutedStyle.Render("Run 'hmm init' to create config files"))
		return b.String()
	}

	b.WriteString(settingsHeaderStyle.Render(fmt.Sprintf("Props (%d configured)", len(m.config.Props))))
	b.WriteString("\n\n")

	// Header row
	headerFmt := "%-6s %-8s %s"
	header := fmt.Sprintf(headerFmt, "ID", "Component", "Name")
	b.WriteString(settingsMutedStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(settingsMutedStyle.Render(strings.Repeat("─", 50)))
	b.WriteString("\n")

	// Calculate visible range
	visibleHeight := m.height - 12
	if visibleHeight < 5 {
		visibleHeight = 5
	}
	start := m.scrollY
	end := start + visibleHeight
	if end > len(m.config.Props) {
		end = len(m.config.Props)
	}
	if start > len(m.config.Props) {
		start = 0
	}

	// Prop rows
	for i := start; i < end; i++ {
		p := m.config.Props[i]
		row := fmt.Sprintf("%-6s %-8s %s", p.ID, p.Component, p.Name)
		b.WriteString(settingsRowStyle.Render(row))
		b.WriteString("\n")
	}

	// Scroll indicator
	if len(m.config.Props) > visibleHeight {
		b.WriteString("\n")
		b.WriteString(settingsMutedStyle.Render(fmt.Sprintf("Showing %d-%d of %d", start+1, end, len(m.config.Props))))
	}

	return b.String()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
