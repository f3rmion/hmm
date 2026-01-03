// Package views provides the individual views for the unified TUI.
package views

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
	"github.com/f3rmion/hmm/internal/tui/bigchar"
	"github.com/f3rmion/hmm/internal/tui/components"
	"github.com/mattn/go-runewidth"
)

// Styles (use from parent package or define locally)
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF6B6B")).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(0, 1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ecdc4"))

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
			Padding(2, 6).
			Margin(1, 0).
			Align(lipgloss.Center)

	bigCharStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#ffe66d")).
			Background(lipgloss.Color("#1a1a2e")).
			Padding(3, 12).
			Align(lipgloss.Center)

	pinyinUnderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4ecdc4")).
				Bold(true).
				Align(lipgloss.Center).
				Padding(0, 1)

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

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff6b6b")).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3d5a80")).
			Padding(1, 2)

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

// Message types
type llmResultMsg struct {
	prompt string
	err    error
}

type clearCopiedMsg struct{}

func clearCopiedAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearCopiedMsg{}
	})
}

// LookupModel is the character lookup view model.
type LookupModel struct {
	input     textinput.Model
	parser    *pinyin.Parser
	dict      *decomp.Dictionary
	generator *prompt.Generator
	config    *config.Config

	// Multi-character support
	characters []components.CharacterResult
	selected   int
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
}

// NewLookupModel creates a new lookup view model.
func NewLookupModel(dict *decomp.Dictionary, cfg *config.Config, gen *prompt.Generator, llmClient *llm.Client) LookupModel {
	ti := textinput.New()
	ti.Placeholder = "Enter Chinese characters..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = 40
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#4ecdc4"))
	ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffe66d"))

	return LookupModel{
		input:     ti,
		parser:    pinyin.NewParser(),
		dict:      dict,
		generator: gen,
		config:    cfg,
		llmClient: llmClient,
	}
}

// SetSize updates the view dimensions.
func (m *LookupModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages.
func (m LookupModel) Update(msg tea.Msg) (LookupModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
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
		case "g":
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
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the lookup view.
func (m LookupModel) View() string {
	var b strings.Builder

	// Input
	b.WriteString(m.input.View())
	b.WriteString("\n")

	// Error
	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.err.Error()))
		b.WriteString("\n")
	}

	// Results
	if len(m.characters) > 0 {
		b.WriteString(m.renderMultiCharView())
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
		help := helpStyle.Render(strings.Join(helpParts, " • "))
		b.WriteString(help)
	} else {
		help := helpStyle.Render("Type characters and press Enter to analyze")
		b.WriteString(help)
	}

	return b.String()
}

func (m *LookupModel) analyzeInput() {
	input := strings.TrimSpace(m.input.Value())
	if input == "" {
		return
	}

	m.inputText = input
	m.characters = nil
	m.selected = 0
	m.err = nil

	for _, r := range input {
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

func (m *LookupModel) analyzeChar(char string) *components.CharacterResult {
	readings := m.parser.ParseChar(char)
	if len(readings) == 0 {
		return nil
	}

	reading := readings[0]

	result := &components.CharacterResult{
		Character: char,
		Pinyin:    reading.Full,
		Initial:   reading.Initial,
		Final:     reading.Final,
		Tone:      reading.Tone,
		ActorID:   pinyin.GetActorID(reading.Initial),
		SetID:     pinyin.GetSetID(reading.Final),
	}

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

	for _, comp := range result.Components {
		if p := m.generator.GetProp(comp); p != nil && p.Name != "" {
			result.PropNames = append(result.PropNames, p.Name)
		}
	}

	return result
}

func (m *LookupModel) updatePrompt() {
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

func (m *LookupModel) generateLLMPrompt() tea.Cmd {
	if m.selected >= len(m.characters) || m.llmClient == nil {
		return nil
	}

	r := m.characters[m.selected]
	client := m.llmClient

	elements := llm.SceneElements{
		Character: r.Character,
		Pinyin:    r.Pinyin,
		Meaning:   r.Meaning,
		ActorName: r.ActorName,
		SetName:   r.SetName,
		ToneRoom:  r.ToneRoom,
		Props:     r.PropNames,
	}

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

func (m LookupModel) renderMultiCharView() string {
	var b strings.Builder

	// Character tabs for multi-character words
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

func (m LookupModel) renderWordBar() string {
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

	nav := ""
	if len(m.characters) > 1 {
		nav = wordNavStyle.Render(fmt.Sprintf("◀ %d/%d ▶", m.selected+1, len(m.characters)))
	}

	charBar := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)
	combined := lipgloss.JoinHorizontal(lipgloss.Center, charBar, "  ", nav)

	return wordDisplayStyle.Render(combined)
}

func (m LookupModel) renderCharacterDetail(r components.CharacterResult) string {
	var b strings.Builder

	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	// Try ASCII art rendering first
	var charDisplay string
	if bigchar.IsAvailable() {
		asciiChar := bigchar.GetCached(r.Character, 30, 15)
		if asciiChar != "" {
			charDisplay = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffe66d")).
				Render(asciiChar)
		}
	}

	// Fallback to regular character in a box
	if charDisplay == "" {
		charDisplay = bigCharStyle.Render(r.Character)
	}

	pinyinDisplay := pinyinUnderStyle.Render(r.Pinyin)

	// Center the character block within view width
	charBlock := lipgloss.JoinVertical(lipgloss.Center, charDisplay, pinyinDisplay)
	centeredChar := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(charBlock)

	b.WriteString("\n")
	b.WriteString(centeredChar)
	b.WriteString("\n")

	// Meaning (centered)
	if r.Meaning != "" {
		meaning := r.Meaning
		maxLen := 60
		if m.width > 0 {
			maxLen = m.width - 20
		}
		if len(meaning) > maxLen {
			meaning = meaning[:maxLen] + "..."
		}
		meaningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f1faee")).
			Width(contentWidth).
			Align(lipgloss.Center)
		b.WriteString(meaningStyle.Render(meaning))
		b.WriteString("\n")
	}

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
		b.WriteString(loadingStyle.Render("Generating image prompt with Claude..."))
		b.WriteString("\n")
	} else if m.llmError != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("LLM Error: " + m.llmError.Error()))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("(Set ANTHROPIC_API_KEY and press 'g' to retry)"))
		b.WriteString("\n")
	} else if m.llmPrompt != "" {
		width := 70
		if m.width > 0 && m.width-10 < width {
			width = m.width - 10
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
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Press 'g' to generate image prompt"))
		b.WriteString("\n")
	}

	return b.String()
}

func (m LookupModel) renderRow(label, value string) string {
	return labelStyle.Render(label+":") + " " + valueStyle.Render(value) + "\n"
}

func (m LookupModel) renderHMMBox(r components.CharacterResult) string {
	var lines []string

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

	toneLine := fmt.Sprintf("%s  %s → %s",
		labelStyle.Render("Tone:"),
		toneStyle.Render(fmt.Sprintf("%d", r.Tone)),
		toneStyle.Render(r.ToneRoom),
	)
	lines = append(lines, toneLine)

	content := strings.Join(lines, "\n")
	return boxStyle.Render(
		subtitleStyle.Render("HMM Breakdown") + "\n\n" + content,
	)
}

func (m LookupModel) renderComponentsBox(r components.CharacterResult) string {
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
		subtitleStyle.Render("Components (Props)") + "\n\n" + content,
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
