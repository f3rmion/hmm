package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/f3rmion/hmm/internal/anki"
	"github.com/f3rmion/hmm/internal/clipboard"
	"github.com/f3rmion/hmm/internal/config"
	"github.com/f3rmion/hmm/internal/decomp"
	"github.com/f3rmion/hmm/internal/hmm"
	"github.com/f3rmion/hmm/internal/llm"
	"github.com/f3rmion/hmm/internal/pinyin"
	"github.com/f3rmion/hmm/internal/prompt"
	"github.com/f3rmion/hmm/internal/tui/components"
)

// Learn view styles
var (
	learnBigCharStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#ffe66d")).
				Background(lipgloss.Color("#1a1a2e")).
				Padding(3, 10).
				Align(lipgloss.Center)

	learnPinyinStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4ecdc4")).
				Bold(true).
				Align(lipgloss.Center).
				Padding(1, 1)

	learnMeaningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f1faee")).
				Align(lipgloss.Center)

	learnProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	learnFlipHintStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4ecdc4")).
				Bold(true).
				Align(lipgloss.Center)

	learnCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#3d5a80")).
			Padding(2, 4).
			Align(lipgloss.Center)
)

// Message types for learn view
type learnLLMResultMsg struct {
	prompt string
	err    error
}

type learnClearCopiedMsg struct{}

func learnClearCopiedAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return learnClearCopiedMsg{}
	})
}

// LearnModel is the flashcard learning view model.
type LearnModel struct {
	pkg       *anki.Package
	parser    *pinyin.Parser
	dict      *decomp.Dictionary
	generator *prompt.Generator
	config    *config.Config

	// Card state
	notes       []*anki.Note
	currentNote int
	flipped     bool

	// Current character data
	character *components.CharacterResult

	// LLM
	llmClient     *llm.Client
	llmPrompt     string
	llmGenerating bool
	llmError      error

	// Clipboard
	copied bool

	// Display
	chineseField string
	width        int
	height       int
}

// NewLearnModel creates a new learn view model.
func NewLearnModel(dict *decomp.Dictionary, cfg *config.Config, gen *prompt.Generator, llmClient *llm.Client) LearnModel {
	return LearnModel{
		parser:    pinyin.NewParser(),
		dict:      dict,
		generator: gen,
		config:    cfg,
		llmClient: llmClient,
	}
}

// SetPackage sets the Anki package to learn from.
func (m *LearnModel) SetPackage(pkg *anki.Package) {
	m.pkg = pkg
	m.llmPrompt = ""
	m.flipped = false

	if pkg == nil {
		m.notes = nil
		m.character = nil
		return
	}

	// Find Chinese field
	m.chineseField = detectChineseFieldFromPkg(pkg)

	// Filter notes to only those with Chinese characters
	var notes []*anki.Note
	for _, note := range pkg.Notes {
		value := pkg.GetFieldValue(note, m.chineseField)
		if containsChineseChars(value) {
			notes = append(notes, note)
		}
	}

	m.notes = notes
	m.currentNote = 0

	if len(notes) > 0 {
		m.loadCurrentCard()
	}
}

// SetSize updates the view dimensions.
func (m *LearnModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages.
func (m LearnModel) Update(msg tea.Msg) (LearnModel, tea.Cmd) {
	// No package loaded
	if m.pkg == nil {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case " ", "enter":
			// Flip card
			m.flipped = !m.flipped
			return m, nil
		case "right", "l", "n":
			// Next card
			if m.currentNote < len(m.notes)-1 {
				m.currentNote++
				m.loadCurrentCard()
				m.flipped = false
				m.llmPrompt = ""
				m.llmError = nil
			}
			return m, nil
		case "left", "h", "p":
			// Previous card
			if m.currentNote > 0 {
				m.currentNote--
				m.loadCurrentCard()
				m.flipped = false
				m.llmPrompt = ""
				m.llmError = nil
			}
			return m, nil
		case "r":
			// Reset to beginning
			m.currentNote = 0
			m.loadCurrentCard()
			m.flipped = false
			m.llmPrompt = ""
			return m, nil
		case "g":
			if m.flipped && m.character != nil && !m.llmGenerating {
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
					return m, learnClearCopiedAfter(2 * time.Second)
				}
			}
			return m, nil
		}

	case learnLLMResultMsg:
		m.llmGenerating = false
		if msg.err != nil {
			m.llmError = msg.err
		} else {
			m.llmPrompt = msg.prompt
		}
		return m, nil

	case learnClearCopiedMsg:
		m.copied = false
		return m, nil
	}

	return m, nil
}

func (m *LearnModel) loadCurrentCard() {
	if m.currentNote >= len(m.notes) {
		return
	}

	note := m.notes[m.currentNote]
	value := m.pkg.GetFieldValue(note, m.chineseField)
	value = stripHTMLTags(value)

	m.character = nil

	// Get first Chinese character
	for _, r := range value {
		if r >= 0x4E00 && r <= 0x9FFF {
			char := string(r)
			m.character = m.analyzeChar(char)
			break
		}
	}
}

func (m *LearnModel) analyzeChar(char string) *components.CharacterResult {
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

func (m *LearnModel) generateLLMPrompt() tea.Cmd {
	if m.character == nil || m.llmClient == nil {
		return nil
	}

	r := m.character
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
	}

	return func() tea.Msg {
		prompt, err := client.GenerateScene(elements)
		return learnLLMResultMsg{prompt: prompt, err: err}
	}
}

// View renders the learn view.
func (m LearnModel) View() string {
	// No package loaded
	if m.pkg == nil {
		return m.renderNoPackage()
	}

	if m.character == nil {
		return helpStyle.Render("No characters found in deck")
	}

	var b strings.Builder

	// Progress
	progress := learnProgressStyle.Render(
		fmt.Sprintf("Card %d of %d", m.currentNote+1, len(m.notes)),
	)
	b.WriteString(progress)
	b.WriteString("\n\n")

	// Card content
	contentWidth := m.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}

	if m.flipped {
		b.WriteString(m.renderFlippedCard(contentWidth))
	} else {
		b.WriteString(m.renderFrontCard(contentWidth))
	}

	// Help
	b.WriteString("\n\n")
	if m.flipped {
		helpText := "space: flip • ←/→: prev/next • r: reset"
		if m.llmPrompt != "" {
			helpText += " • y: copy"
		} else {
			helpText += " • g: generate"
		}
		b.WriteString(helpStyle.Render(helpText))
	} else {
		b.WriteString(helpStyle.Render("space: flip • ←/→: prev/next • r: reset"))
	}

	return b.String()
}

func (m LearnModel) renderNoPackage() string {
	var b strings.Builder

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3d5a80")).
		Padding(2, 4).
		Align(lipgloss.Center)

	content := browseNoDataStyle.Render("No Anki Deck Loaded") + "\n\n" +
		helpStyle.Render("Load a deck in Browse or Open Deck first")

	b.WriteString("\n\n")
	b.WriteString(box.Render(content))

	return b.String()
}

func (m LearnModel) renderFrontCard(contentWidth int) string {
	r := m.character

	// Just show the big character
	charDisplay := learnBigCharStyle.Render(r.Character)

	charBlock := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(charDisplay)

	hint := learnFlipHintStyle.Width(contentWidth).Render("Press SPACE to reveal")

	return charBlock + "\n\n" + hint
}

func (m LearnModel) renderFlippedCard(contentWidth int) string {
	var b strings.Builder
	r := m.character

	// Character with pinyin
	charDisplay := learnBigCharStyle.Render(r.Character)
	pinyinDisplay := learnPinyinStyle.Render(r.Pinyin)

	charBlock := lipgloss.JoinVertical(lipgloss.Center, charDisplay, pinyinDisplay)
	centered := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(charBlock)

	b.WriteString(centered)
	b.WriteString("\n")

	// Meaning
	if r.Meaning != "" {
		meaning := r.Meaning
		if len(meaning) > 60 {
			meaning = meaning[:60] + "..."
		}
		meaningDisplay := learnMeaningStyle.Width(contentWidth).Render(meaning)
		b.WriteString(meaningDisplay)
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// HMM Breakdown
	b.WriteString(m.renderHMMBox(r))
	b.WriteString("\n")

	// Components
	if len(r.Components) > 0 {
		b.WriteString(m.renderComponentsBox(r))
		b.WriteString("\n")
	}

	// LLM prompt
	if m.llmGenerating {
		b.WriteString("\n")
		b.WriteString(loadingStyle.Render("Generating image prompt..."))
	} else if m.llmError != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.llmError.Error()))
	} else if m.llmPrompt != "" {
		width := 70
		if m.width > 0 && m.width-10 < width {
			width = m.width - 10
		}
		headerText := actorStyle.Render("Image Prompt")
		if m.copied {
			headerText += "  " + copiedStyle.Render("Copied!")
		}
		llmBox := llmPromptStyle.Width(width).Render(
			headerText + "\n\n" + wordWrap(m.llmPrompt, width-6),
		)
		b.WriteString(llmBox)
	}

	return b.String()
}

func (m LearnModel) renderHMMBox(r *components.CharacterResult) string {
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

func (m LearnModel) renderComponentsBox(r *components.CharacterResult) string {
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

	content := strings.Join(lines, "\n")
	return boxStyle.Render(
		subtitleStyle.Render("Components (Props)") + "\n\n" + content,
	)
}
