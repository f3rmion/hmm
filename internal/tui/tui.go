// Package tui provides an interactive terminal UI for HMM.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/f3rmion/hmm/internal/clipboard"
	"github.com/f3rmion/hmm/internal/config"
	"github.com/f3rmion/hmm/internal/decomp"
	"github.com/f3rmion/hmm/internal/hmm"
	"github.com/f3rmion/hmm/internal/llm"
	"github.com/f3rmion/hmm/internal/pinyin"
	"github.com/f3rmion/hmm/internal/prompt"
	"github.com/mattn/go-runewidth"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B")).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ecdc4"))

	// Character tab styles
	charTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 2).
			Margin(0, 1)

	charTabActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffe66d")).
				Background(lipgloss.Color("#2d3436")).
				Padding(0, 2).
				Margin(0, 1)

	charTabPinyinStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666")).
				Italic(true)

	characterStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffe66d")).
			Background(lipgloss.Color("#2d3436")).
			Padding(1, 4).
			Margin(1, 0)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a8dadc")).
			Bold(true).
			Width(12)

	valueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f1faee"))

	actorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6b6b")).
			Bold(true)

	setStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ecdc4")).
			Bold(true)

	propStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffe66d")).
			Bold(true)

	toneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a8e6cf")).
			Bold(true)

	promptBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#4ecdc4")).
			Padding(1, 2).
			Margin(1, 0)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6b6b")).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3d5a80")).
			Padding(1, 2)

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#3d5a80"))

	wordNavStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ecdc4")).
			Bold(true).
			Padding(0, 1)

	wordDisplayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#ffe66d")).
				Padding(0, 2).
				Margin(1, 0)

	llmPromptStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#ff6b6b")).
			Padding(1, 2).
			Margin(1, 0)

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffe66d")).
			Bold(true).
			Italic(true)

	copiedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a8e6cf")).
			Bold(true)
)

// LLM generation messages
type llmResultMsg struct {
	prompt string
	err    error
}

// Clipboard messages
type clearCopiedMsg struct{}

func clearCopiedAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearCopiedMsg{}
	})
}

// Model is the Bubble Tea model for the HMM TUI.
type Model struct {
	input     textinput.Model
	parser    *pinyin.Parser
	dict      *decomp.Dictionary
	generator *prompt.Generator
	config    *config.Config

	// Multi-character support
	characters []CharacterResult
	selected   int // Currently selected character index
	inputText  string

	prompt string
	err    error

	// LLM integration
	llmClient     *llm.Client
	llmPrompt     string
	llmGenerating bool
	llmError      error

	// Clipboard
	copied bool

	width  int
	height int
	ready  bool
}

// CharacterResult holds the analysis of a character.
type CharacterResult struct {
	Character  string
	Pinyin     string
	Meaning    string
	Decomp     string
	Components []string
	Etymology  string
	Initial    string
	Final      string
	Tone       hmm.Tone
	ActorID    string
	ActorName  string
	SetID      string
	SetName    string
	ToneRoom   string
	PropNames  []string
}

// New creates a new TUI model.
func New(dict *decomp.Dictionary, cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "Enter Chinese characters or words..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 40
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4ecdc4"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffe66d"))

	var gen *prompt.Generator
	if cfg != nil {
		gen = prompt.NewGenerator(cfg.Actors, cfg.Sets, cfg.Props)
	} else {
		gen = prompt.NewGenerator(nil, nil, nil)
	}

	// Try to create LLM client (optional - won't fail if no API key)
	llmClient, _ := llm.NewClient()

	return Model{
		input:     ti,
		parser:    pinyin.NewParser(),
		dict:      dict,
		generator: gen,
		config:    cfg,
		llmClient: llmClient,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "enter":
			m.analyzeInput()
			m.llmPrompt = ""
			m.llmError = nil
			return m, nil
		case "left", "h":
			if len(m.characters) > 0 {
				m.selected--
				if m.selected < 0 {
					m.selected = len(m.characters) - 1
				}
				m.updatePrompt()
				m.llmPrompt = ""
				m.llmError = nil
			}
			return m, nil
		case "right", "l":
			if len(m.characters) > 0 {
				m.selected++
				if m.selected >= len(m.characters) {
					m.selected = 0
				}
				m.updatePrompt()
				m.llmPrompt = ""
				m.llmError = nil
			}
			return m, nil
		case "tab":
			// Same as right
			if len(m.characters) > 0 {
				m.selected = (m.selected + 1) % len(m.characters)
				m.updatePrompt()
				m.llmPrompt = ""
				m.llmError = nil
			}
			return m, nil
		case "shift+tab":
			// Same as left
			if len(m.characters) > 0 {
				m.selected--
				if m.selected < 0 {
					m.selected = len(m.characters) - 1
				}
				m.updatePrompt()
				m.llmPrompt = ""
				m.llmError = nil
			}
			return m, nil
		case "g":
			// Generate LLM prompt
			if len(m.characters) > 0 && !m.llmGenerating {
				if m.llmClient == nil {
					m.llmError = fmt.Errorf("ANTHROPIC_API_KEY not set")
					return m, nil
				}
				m.llmGenerating = true
				m.llmError = nil
				return m, m.generateLLMPrompt()
			}
			return m, nil
		case "y":
			// Copy prompt to clipboard
			if m.llmPrompt != "" {
				if err := clipboard.Write(m.llmPrompt); err == nil {
					m.copied = true
					return m, clearCopiedAfter(2 * time.Second)
				}
			}
			return m, nil
		}

	case llmResultMsg:
		m.llmGenerating = false
		if msg.err != nil {
			m.llmError = msg.err
		} else {
			m.llmPrompt = msg.prompt
		}
		return m, nil

	case clearCopiedMsg:
		m.copied = false
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// analyzeInput processes the current input.
func (m *Model) analyzeInput() {
	input := strings.TrimSpace(m.input.Value())
	if input == "" {
		return
	}

	m.inputText = input
	m.characters = nil
	m.selected = 0
	m.err = nil

	// Extract and analyze each Chinese character
	for _, r := range input {
		// Skip non-Chinese characters
		if r < 0x4E00 || r > 0x9FFF {
			continue
		}

		char := string(r)
		result := m.analyzeChar(char)
		if result != nil {
			m.characters = append(m.characters, *result)
		}
	}

	if len(m.characters) == 0 {
		m.err = fmt.Errorf("no Chinese characters found in: %s", input)
		return
	}

	m.updatePrompt()
}

// analyzeChar analyzes a single character.
func (m *Model) analyzeChar(char string) *CharacterResult {
	readings := m.parser.ParseChar(char)
	if len(readings) == 0 {
		return nil
	}

	reading := readings[0]

	result := &CharacterResult{
		Character: char,
		Pinyin:    reading.Full,
		Initial:   reading.Initial,
		Final:     reading.Final,
		Tone:      reading.Tone,
		ActorID:   pinyin.GetActorID(reading.Initial),
		SetID:     pinyin.GetSetID(reading.Final),
	}

	// Get dictionary info
	if m.dict != nil {
		if entry := m.dict.Lookup(char); entry != nil {
			result.Meaning = entry.Definition
			result.Decomp = decomp.FormatDecomposition(entry.Decomposition)
			result.Components = decomp.ExtractComponents(entry.Decomposition)
			if entry.Etymology != nil {
				if entry.Etymology.Hint != "" {
					result.Etymology = entry.Etymology.Hint
				} else {
					result.Etymology = entry.Etymology.Type
				}
			}
		}
	}

	// Get actor/set info from config
	if actor := m.generator.GetActor(result.ActorID); actor != nil {
		result.ActorName = actor.Name
	}
	if set := m.generator.GetSet(result.SetID); set != nil {
		result.SetName = set.Name
	}
	result.ToneRoom = m.generator.GetToneRoom(
		m.generator.GetSet(result.SetID),
		reading.Tone,
	)

	// Get prop names
	for _, comp := range result.Components {
		if p := m.generator.GetProp(comp); p != nil && p.Name != "" {
			result.PropNames = append(result.PropNames, p.Name)
		}
	}

	return result
}

// updatePrompt generates the prompt for the current selection.
func (m *Model) updatePrompt() {
	if m.selected >= len(m.characters) {
		return
	}

	r := m.characters[m.selected]
	sceneData := m.generator.BuildSceneData(
		r.Character,
		r.Pinyin,
		r.ActorID,
		r.SetID,
		r.Tone,
		r.Components,
		r.Meaning,
		r.Etymology,
		r.Decomp,
	)
	if p, err := m.generator.Generate(sceneData); err == nil {
		m.prompt = p
	}
}

// generateLLMPrompt creates a command that generates a scene via the LLM.
func (m *Model) generateLLMPrompt() tea.Cmd {
	if m.selected >= len(m.characters) || m.llmClient == nil {
		return nil
	}

	r := m.characters[m.selected]
	client := m.llmClient

	// Build elements for LLM
	elements := llm.SceneElements{
		Character: r.Character,
		Pinyin:    r.Pinyin,
		Meaning:   r.Meaning,
		ActorName: r.ActorName,
		SetName:   r.SetName,
		ToneRoom:  r.ToneRoom,
		Props:     r.PropNames,
	}

	// Get descriptions from config
	if m.config != nil {
		for _, a := range m.config.Actors {
			if a.ID == r.ActorID {
				elements.ActorDesc = a.Description
				break
			}
		}
		for _, s := range m.config.Sets {
			if s.ID == r.SetID {
				elements.SetDesc = s.Description
				for _, room := range s.Rooms {
					if hmm.Tone(room.Tone) == r.Tone {
						elements.ToneRoomDesc = room.Description
						break
					}
				}
				break
			}
		}
		for _, comp := range r.Components {
			for _, p := range m.config.Props {
				if p.ID == comp || p.Component == comp {
					elements.PropDescs = append(elements.PropDescs, p.Description)
					break
				}
			}
		}
	}

	return func() tea.Msg {
		prompt, err := client.GenerateScene(elements)
		return llmResultMsg{prompt: prompt, err: err}
	}
}

// View renders the UI.
func (m Model) View() string {
	var b strings.Builder

	// Header
	header := titleStyle.Render("  漢字 Movie Method  ") + "  " +
		subtitleStyle.Render("Interactive Character Explorer")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Input
	b.WriteString("  ")
	b.WriteString(m.input.View())
	b.WriteString("\n")

	// Error
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  " + m.err.Error()))
		b.WriteString("\n")
	}

	// Results
	if len(m.characters) > 0 {
		b.WriteString(m.renderMultiCharView())
	} else {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Type Chinese characters and press Enter"))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	if len(m.characters) > 0 {
		var helpParts []string
		if len(m.characters) > 1 {
			helpParts = append(helpParts, "←/→: navigate")
		}
		helpParts = append(helpParts, "g: generate")
		if m.llmPrompt != "" {
			helpParts = append(helpParts, "y: copy")
		}
		helpParts = append(helpParts, "enter: analyze")
		helpParts = append(helpParts, "esc: quit")
		help := helpStyle.Render("  " + strings.Join(helpParts, " • "))
		b.WriteString(help)
	} else {
		help := helpStyle.Render("  enter: analyze • esc: quit")
		b.WriteString(help)
	}

	return b.String()
}

// renderMultiCharView renders the multi-character view with navigation.
func (m Model) renderMultiCharView() string {
	var b strings.Builder

	// Word display with all characters
	if len(m.characters) > 1 {
		b.WriteString(m.renderWordBar())
		b.WriteString("\n")
	}

	// Selected character details
	if m.selected < len(m.characters) {
		b.WriteString(m.renderCharacterDetail(m.characters[m.selected]))
	}

	return b.String()
}

// renderWordBar renders the horizontal character navigation bar.
func (m Model) renderWordBar() string {
	var tabs []string

	for i, c := range m.characters {
		charWithPinyin := fmt.Sprintf("%s\n%s", c.Character, charTabPinyinStyle.Render(c.Pinyin))

		var tab string
		if i == m.selected {
			tab = charTabActiveStyle.Render(charWithPinyin)
		} else {
			tab = charTabStyle.Render(charWithPinyin)
		}
		tabs = append(tabs, tab)
	}

	// Navigation hints
	nav := ""
	if len(m.characters) > 1 {
		nav = wordNavStyle.Render(fmt.Sprintf("◀ %d/%d ▶", m.selected+1, len(m.characters)))
	}

	charBar := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)
	combined := lipgloss.JoinHorizontal(lipgloss.Center, charBar, "  ", nav)

	return wordDisplayStyle.Render(combined)
}

// renderCharacterDetail renders the detailed view for a single character.
func (m Model) renderCharacterDetail(r CharacterResult) string {
	var b strings.Builder

	// Big character display (only if single char or want emphasis)
	if len(m.characters) == 1 {
		charDisplay := characterStyle.Render(r.Character)
		b.WriteString(charDisplay)
		b.WriteString("\n")
	}

	// Pinyin and meaning
	b.WriteString(m.renderRow("Pinyin", r.Pinyin))
	if r.Meaning != "" {
		meaning := r.Meaning
		maxLen := 60
		if m.width > 0 {
			maxLen = m.width - 20
		}
		if len(meaning) > maxLen {
			meaning = meaning[:maxLen] + "..."
		}
		b.WriteString(m.renderRow("Meaning", meaning))
	}
	b.WriteString("\n")

	// HMM Breakdown Box
	hmmBox := m.renderHMMBox(r)
	b.WriteString(hmmBox)
	b.WriteString("\n")

	// Components
	if len(r.Components) > 0 {
		compBox := m.renderComponentsBox(r)
		b.WriteString(compBox)
		b.WriteString("\n")
	}

	// Etymology
	if r.Etymology != "" {
		b.WriteString(m.renderRow("Etymology", r.Etymology))
		b.WriteString("\n")
	}

	// LLM-generated image prompt
	if m.llmGenerating {
		b.WriteString("\n")
		b.WriteString(loadingStyle.Render("  Generating image prompt with Claude..."))
		b.WriteString("\n")
	} else if m.llmError != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  LLM Error: " + m.llmError.Error()))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  (Set ANTHROPIC_API_KEY and press 'g' to retry)"))
		b.WriteString("\n")
	} else if m.llmPrompt != "" {
		width := 80
		if m.width > 0 && m.width-6 < width {
			width = m.width - 6
		}
		header := actorStyle.Render("Image Prompt")
		if m.copied {
			header += "  " + copiedStyle.Render("Copied!")
		}
		llmBox := llmPromptStyle.Width(width).Render(
			header + "\n\n" +
				wordWrap(m.llmPrompt, width-6),
		)
		b.WriteString(llmBox)
	} else {
		// No LLM prompt yet - show hint
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  Press 'g' to generate image prompt"))
		b.WriteString("\n")
	}

	return b.String()
}

// renderRow renders a label-value row.
func (m Model) renderRow(label, value string) string {
	return "  " + labelStyle.Render(label+":") + " " + valueStyle.Render(value) + "\n"
}

// renderHMMBox renders the HMM breakdown in a nice box.
func (m Model) renderHMMBox(r CharacterResult) string {
	var lines []string

	// Initial → Actor
	initial := r.Initial
	if initial == "" {
		initial = "Ø"
	}
	actorLine := fmt.Sprintf("%s  %s → %s",
		labelStyle.Render("Initial:"),
		actorStyle.Render(initial),
		actorStyle.Render(formatActorName(r.ActorID, r.ActorName)),
	)
	lines = append(lines, actorLine)

	// Final → Set
	final := r.Final
	if final == "" {
		final = "Ø"
	}
	setLine := fmt.Sprintf("%s  %s → %s",
		labelStyle.Render("Final:"),
		setStyle.Render(final),
		setStyle.Render(formatSetName(r.SetID, r.SetName)),
	)
	lines = append(lines, setLine)

	// Tone → Room
	toneLine := fmt.Sprintf("%s  %s → %s",
		labelStyle.Render("Tone:"),
		toneStyle.Render(fmt.Sprintf("%d", r.Tone)),
		toneStyle.Render(r.ToneRoom),
	)
	lines = append(lines, toneLine)

	content := strings.Join(lines, "\n")
	return boxStyle.Render(
		subtitleStyle.Render("🎬 HMM Breakdown") + "\n\n" + content,
	)
}

// renderComponentsBox renders the components/props box.
func (m Model) renderComponentsBox(r CharacterResult) string {
	var lines []string

	for i, comp := range r.Components {
		propName := "(not configured)"
		if i < len(r.PropNames) && r.PropNames[i] != "" {
			propName = r.PropNames[i]
		}
		line := fmt.Sprintf("  %s → %s",
			propStyle.Render(comp),
			valueStyle.Render(propName),
		)
		lines = append(lines, line)
	}

	if r.Decomp != "" {
		lines = append(lines, "")
		lines = append(lines, helpStyle.Render("Structure: "+r.Decomp))
	}

	content := strings.Join(lines, "\n")
	return boxStyle.Render(
		subtitleStyle.Render("🎭 Components (Props)") + "\n\n" + content,
	)
}

func formatActorName(id, name string) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("Actor [%s]", id)
}

func formatSetName(id, name string) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("Set [%s]", id)
}

func wordWrap(s string, width int) string {
	if width <= 0 {
		width = 60
	}
	var lines []string
	var currentLine strings.Builder
	currentWidth := 0

	words := strings.Fields(s)
	for _, word := range words {
		wordWidth := runewidth.StringWidth(word)
		if currentWidth+wordWidth+1 > width && currentWidth > 0 {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentWidth = 0
		}
		if currentWidth > 0 {
			currentLine.WriteString(" ")
			currentWidth++
		}
		currentLine.WriteString(word)
		currentWidth += wordWidth
	}
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return strings.Join(lines, "\n")
}
