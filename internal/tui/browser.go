package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
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
)

// BrowserModel is the Bubble Tea model for browsing Anki decks.
type BrowserModel struct {
	pkg       *anki.Package
	parser    *pinyin.Parser
	dict      *decomp.Dictionary
	generator *prompt.Generator
	config    *config.Config

	// Card navigation
	notes         []*anki.Note
	filteredNotes []*anki.Note
	currentNote   int

	// Character navigation within a note
	characters []CharacterResult
	selected   int

	// Search
	searchInput textinput.Model
	searching   bool
	searchTerm  string

	// LLM
	llmClient     *llm.Client
	llmPrompt     string
	llmGenerating bool
	llmError      error

	// Batch generation
	charPrompts       map[int]string // Stores generated prompts by character index
	batchGenerating   bool
	batchTotal        int
	batchCompleted    int

	// Clipboard
	copied bool

	// Display
	chineseField string
	width        int
	height       int
	ready        bool
}

// clearCopiedBrowserMsg is sent to clear the copied indicator
type clearCopiedBrowserMsg struct{}

// clearCopiedBrowserAfter returns a command that clears the copied state after a duration
func clearCopiedBrowserAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return clearCopiedBrowserMsg{}
	})
}

// batchResultMsg is sent when a batch prompt generation completes for one character
type batchResultMsg struct {
	index  int
	prompt string
	err    error
}

// Additional styles for browser
var (
	cardCountStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Padding(0, 1)

	deckNameStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#4ecdc4")).
			Bold(true)

	fieldLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	fieldValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f1faee"))

	searchBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#ffe66d")).
			Padding(0, 1)
)

// NewBrowser creates a new browser TUI model.
func NewBrowser(dict *decomp.Dictionary, cfg *config.Config, pkg *anki.Package) BrowserModel {
	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 50
	si.Width = 30

	var gen *prompt.Generator
	if cfg != nil {
		gen = prompt.NewGenerator(cfg.Actors, cfg.Sets, cfg.Props)
	} else {
		gen = prompt.NewGenerator(nil, nil, nil)
	}

	llmClient, _ := llm.NewClient()

	// Find Chinese field
	chineseField := detectChineseFieldFromPkg(pkg)

	// Filter notes to only those with Chinese characters
	var notes []*anki.Note
	for _, note := range pkg.Notes {
		value := pkg.GetFieldValue(note, chineseField)
		if containsChineseChars(value) {
			notes = append(notes, note)
		}
	}

	m := BrowserModel{
		pkg:           pkg,
		parser:        pinyin.NewParser(),
		dict:          dict,
		generator:     gen,
		config:        cfg,
		notes:         notes,
		filteredNotes: notes,
		searchInput:   si,
		llmClient:     llmClient,
		chineseField:  chineseField,
		charPrompts:   make(map[int]string),
	}

	// Load first note
	if len(notes) > 0 {
		m.loadCurrentNote()
	}

	return m
}

// Init initializes the model.
func (m BrowserModel) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m BrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.searching {
			switch msg.String() {
			case "enter":
				m.searching = false
				m.searchTerm = m.searchInput.Value()
				m.applyFilter()
				return m, nil
			case "esc":
				m.searching = false
				m.searchInput.SetValue("")
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		case "up", "k":
			if m.currentNote > 0 {
				m.currentNote--
				m.loadCurrentNote()
				m.llmPrompt = ""
				m.llmError = nil
			}
			return m, nil
		case "down", "j":
			if m.currentNote < len(m.filteredNotes)-1 {
				m.currentNote++
				m.loadCurrentNote()
				m.llmPrompt = ""
				m.llmError = nil
			}
			return m, nil
		case "left", "h":
			if len(m.characters) > 0 && m.selected > 0 {
				m.selected--
				// Load cached prompt if available
				if p, ok := m.charPrompts[m.selected]; ok {
					m.llmPrompt = p
				} else {
					m.llmPrompt = ""
				}
				m.llmError = nil
			}
			return m, nil
		case "right", "l":
			if len(m.characters) > 0 && m.selected < len(m.characters)-1 {
				m.selected++
				// Load cached prompt if available
				if p, ok := m.charPrompts[m.selected]; ok {
					m.llmPrompt = p
				} else {
					m.llmPrompt = ""
				}
				m.llmError = nil
			}
			return m, nil
		case "/":
			m.searching = true
			m.searchInput.Focus()
			return m, textinput.Blink
		case "c":
			// Clear search
			m.searchTerm = ""
			m.searchInput.SetValue("")
			m.filteredNotes = m.notes
			m.currentNote = 0
			if len(m.filteredNotes) > 0 {
				m.loadCurrentNote()
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
					return m, clearCopiedBrowserAfter(2 * time.Second)
				}
			}
			return m, nil
		case "B":
			// Batch generate prompts for all characters in current card
			if len(m.characters) > 0 && !m.batchGenerating && !m.llmGenerating {
				if m.llmClient == nil {
					m.llmError = fmt.Errorf("ANTHROPIC_API_KEY not set")
					return m, nil
				}
				m.batchGenerating = true
				m.batchTotal = len(m.characters)
				m.batchCompleted = 0
				m.llmError = nil
				return m, m.generateBatchPrompts()
			}
			return m, nil
		}

	case llmResultMsg:
		m.llmGenerating = false
		if msg.err != nil {
			m.llmError = msg.err
		} else {
			m.llmPrompt = msg.prompt
			m.charPrompts[m.selected] = msg.prompt
		}
		return m, nil

	case batchResultMsg:
		m.batchCompleted++
		if msg.err == nil && msg.prompt != "" {
			m.charPrompts[msg.index] = msg.prompt
			// Update llmPrompt if this is the currently selected character
			if msg.index == m.selected {
				m.llmPrompt = msg.prompt
			}
		}
		if m.batchCompleted >= m.batchTotal {
			m.batchGenerating = false
			// Set llmPrompt to current selection's prompt if available
			if p, ok := m.charPrompts[m.selected]; ok {
				m.llmPrompt = p
			}
		}
		return m, nil

	case clearCopiedBrowserMsg:
		m.copied = false
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	}

	return m, tea.Batch(cmds...)
}

// loadCurrentNote analyzes the current note.
func (m *BrowserModel) loadCurrentNote() {
	if m.currentNote >= len(m.filteredNotes) {
		return
	}

	note := m.filteredNotes[m.currentNote]
	value := m.pkg.GetFieldValue(note, m.chineseField)
	value = stripHTMLTags(value)

	m.characters = nil
	m.selected = 0
	m.charPrompts = make(map[int]string)
	m.batchGenerating = false
	m.batchCompleted = 0
	m.batchTotal = 0

	for _, r := range value {
		if r >= 0x4E00 && r <= 0x9FFF {
			char := string(r)
			result := m.analyzeChar(char)
			if result != nil {
				m.characters = append(m.characters, *result)
			}
		}
	}
}

// analyzeChar analyzes a single character.
func (m *BrowserModel) analyzeChar(char string) *CharacterResult {
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

// applyFilter filters notes by search term.
func (m *BrowserModel) applyFilter() {
	if m.searchTerm == "" {
		m.filteredNotes = m.notes
	} else {
		m.filteredNotes = nil
		term := strings.ToLower(m.searchTerm)
		for _, note := range m.notes {
			// Search all fields
			for _, field := range note.Fields {
				if strings.Contains(strings.ToLower(stripHTMLTags(field)), term) {
					m.filteredNotes = append(m.filteredNotes, note)
					break
				}
			}
		}
	}
	m.currentNote = 0
	if len(m.filteredNotes) > 0 {
		m.loadCurrentNote()
	} else {
		m.characters = nil
	}
}

// generateLLMPrompt creates a command that generates a scene via the LLM.
func (m *BrowserModel) generateLLMPrompt() tea.Cmd {
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
	}

	return func() tea.Msg {
		prompt, err := client.GenerateScene(elements)
		return llmResultMsg{prompt: prompt, err: err}
	}
}

// generateBatchPrompts creates commands to generate scenes for all characters.
func (m *BrowserModel) generateBatchPrompts() tea.Cmd {
	if len(m.characters) == 0 || m.llmClient == nil {
		return nil
	}

	var cmds []tea.Cmd
	client := m.llmClient

	for i, r := range m.characters {
		// Skip if already generated
		if _, exists := m.charPrompts[i]; exists {
			cmds = append(cmds, func() tea.Msg {
				return batchResultMsg{index: i, prompt: m.charPrompts[i], err: nil}
			})
			continue
		}

		idx := i
		char := r

		elements := llm.SceneElements{
			Character: char.Character,
			Pinyin:    char.Pinyin,
			Meaning:   char.Meaning,
			ActorName: char.ActorName,
			SetName:   char.SetName,
			ToneRoom:  char.ToneRoom,
			Props:     char.PropNames,
		}

		if m.config != nil {
			for _, a := range m.config.Actors {
				if a.ID == char.ActorID {
					elements.ActorDesc = a.Description
					break
				}
			}
			for _, s := range m.config.Sets {
				if s.ID == char.SetID {
					elements.SetDesc = s.Description
					for _, room := range s.Rooms {
						if hmm.Tone(room.Tone) == char.Tone {
							elements.ToneRoomDesc = room.Description
							break
						}
					}
					break
				}
			}
		}

		cmds = append(cmds, func() tea.Msg {
			prompt, err := client.GenerateScene(elements)
			return batchResultMsg{index: idx, prompt: prompt, err: err}
		})
	}

	return tea.Batch(cmds...)
}

// View renders the UI.
func (m BrowserModel) View() string {
	var b strings.Builder

	// Header
	header := titleStyle.Render("  漢字 Movie Method  ") + "  " +
		subtitleStyle.Render("Anki Browser")
	b.WriteString(header)
	b.WriteString("\n\n")

	// Search bar
	if m.searching {
		b.WriteString("  ")
		b.WriteString(searchBoxStyle.Render("🔍 " + m.searchInput.View()))
		b.WriteString("\n\n")
	} else if m.searchTerm != "" {
		b.WriteString("  ")
		b.WriteString(helpStyle.Render(fmt.Sprintf("Filter: \"%s\" (press 'c' to clear)", m.searchTerm)))
		b.WriteString("\n\n")
	}

	// Card counter
	if len(m.filteredNotes) > 0 {
		counter := cardCountStyle.Render(
			fmt.Sprintf("Card %d of %d", m.currentNote+1, len(m.filteredNotes)),
		)
		b.WriteString("  ")
		b.WriteString(counter)
		b.WriteString("\n\n")
	}

	// Current note
	if m.currentNote < len(m.filteredNotes) {
		b.WriteString(m.renderNoteView())
	} else {
		b.WriteString("  ")
		b.WriteString(helpStyle.Render("No cards match your search"))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	helpText := "  ↑/↓: prev/next card • ←/→: prev/next char • /: search • g: generate"
	if len(m.characters) > 1 {
		helpText += " • B: batch"
	}
	if m.llmPrompt != "" {
		helpText += " • y: copy"
	}
	helpText += " • q: quit"
	help := helpStyle.Render(helpText)
	b.WriteString(help)

	return b.String()
}

// renderNoteView renders the current note.
func (m BrowserModel) renderNoteView() string {
	var b strings.Builder

	note := m.filteredNotes[m.currentNote]
	fieldNames := m.pkg.GetFieldNames(note)

	// Show main fields
	for i, value := range note.Fields {
		if i >= 3 {
			break // Only show first 3 fields to avoid clutter
		}
		fieldName := fmt.Sprintf("Field %d", i)
		if i < len(fieldNames) {
			fieldName = fieldNames[i]
		}
		cleanValue := stripHTMLTags(value)
		if len(cleanValue) > 80 {
			cleanValue = cleanValue[:80] + "..."
		}
		b.WriteString("  ")
		b.WriteString(fieldLabelStyle.Render(fieldName + ": "))
		b.WriteString(fieldValueStyle.Render(cleanValue))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Character tabs
	if len(m.characters) > 1 {
		b.WriteString(m.renderCharTabs())
		b.WriteString("\n")
	}

	// Selected character details
	if m.selected < len(m.characters) {
		b.WriteString(m.renderCharacterDetail(m.characters[m.selected]))
	}

	return b.String()
}

// renderCharTabs renders horizontal character tabs.
func (m BrowserModel) renderCharTabs() string {
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

// renderCharacterDetail renders HMM breakdown for a character.
func (m BrowserModel) renderCharacterDetail(r CharacterResult) string {
	var b strings.Builder

	// HMM Breakdown
	b.WriteString(m.renderHMMBox(r))
	b.WriteString("\n")

	// Components
	if len(r.Components) > 0 {
		b.WriteString(m.renderComponentsBox(r))
		b.WriteString("\n")
	}

	// LLM prompt
	if m.batchGenerating {
		b.WriteString("\n")
		progress := fmt.Sprintf("  Generating prompts... %d/%d", m.batchCompleted, m.batchTotal)
		b.WriteString(loadingStyle.Render(progress))
		b.WriteString("\n")
	} else if m.llmGenerating {
		b.WriteString("\n")
		b.WriteString(loadingStyle.Render("  Generating image prompt..."))
		b.WriteString("\n")
	} else if m.llmError != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("  " + m.llmError.Error()))
		b.WriteString("\n")
	} else if m.llmPrompt != "" {
		width := 80
		if m.width > 0 && m.width-6 < width {
			width = m.width - 6
		}
		headerText := actorStyle.Render("Image Prompt")
		if m.copied {
			headerText += "  " + copiedStyle.Render("✓ Copied!")
		}
		// Show batch completion count if multiple characters
		if len(m.charPrompts) > 1 {
			headerText += "  " + helpStyle.Render(fmt.Sprintf("(%d/%d generated)", len(m.charPrompts), len(m.characters)))
		}
		llmBox := llmPromptStyle.Width(width).Render(
			headerText + "\n\n" +
				wordWrap(m.llmPrompt, width-6),
		)
		b.WriteString(llmBox)
	} else {
		b.WriteString("\n")
		hint := "  Press 'g' to generate image prompt"
		if len(m.characters) > 1 {
			hint += " or 'B' for batch"
		}
		b.WriteString(helpStyle.Render(hint))
		b.WriteString("\n")
	}

	return b.String()
}

// renderHMMBox renders the HMM breakdown box.
func (m BrowserModel) renderHMMBox(r CharacterResult) string {
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
		subtitleStyle.Render("🎬 HMM Breakdown") + "\n\n" + content,
	)
}

// renderComponentsBox renders components/props.
func (m BrowserModel) renderComponentsBox(r CharacterResult) string {
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
		subtitleStyle.Render("🎭 Components (Props)") + "\n\n" + content,
	)
}

// Helper functions

func detectChineseFieldFromPkg(pkg *anki.Package) string {
	for i, note := range pkg.Notes {
		if i >= 10 {
			break
		}
		fieldNames := pkg.GetFieldNames(note)
		for j, value := range note.Fields {
			if containsChineseChars(value) {
				if j < len(fieldNames) {
					return fieldNames[j]
				}
				return fmt.Sprintf("field_%d", j)
			}
		}
	}
	return ""
}

func containsChineseChars(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

func stripHTMLTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}
