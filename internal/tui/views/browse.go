package views

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
	"github.com/f3rmion/hmm/internal/tui/components"
)

// Browse view styles (reuse from lookup)
var (
	browseCharTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Padding(0, 2).
				Margin(0, 1)

	browseCharTabActiveStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("#ffe66d")).
					Background(lipgloss.Color("#2d3436")).
					Padding(0, 2).
					Margin(0, 1)

	browseCharTabPinyinStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#666666")).
					Italic(true)

	browseWordNavStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#4ecdc4")).
				Bold(true).
				Padding(0, 1)

	browseWordDisplayStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#ffe66d")).
				Padding(0, 2).
				Margin(1, 0)

	browseCardCountStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888")).
				Padding(0, 1)

	browseFieldLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	browseFieldValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f1faee"))

	browseSearchBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#ffe66d")).
				Padding(0, 1)

	browseNoDataStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#666666")).
				Italic(true).
				Align(lipgloss.Center)
)

// Message types for browse view
type browseLLMResultMsg struct {
	prompt string
	err    error
}

type browseBatchResultMsg struct {
	index  int
	prompt string
	err    error
}

type browseClearCopiedMsg struct{}

func browseClearCopiedAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return browseClearCopiedMsg{}
	})
}

// BrowseModel is the Anki deck browser view model.
type BrowseModel struct {
	pkg       *anki.Package
	parser    *pinyin.Parser
	dict      *decomp.Dictionary
	generator *prompt.Generator
	config    *config.Config

	// Card navigation
	notes         []*anki.Note
	filteredNotes []*anki.Note
	currentNote   int

	// Character navigation
	characters []components.CharacterResult
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
	charPrompts     map[int]string
	batchGenerating bool
	batchTotal      int
	batchCompleted  int

	// Clipboard
	copied bool

	// Display
	chineseField string
	width        int
	height       int
}

// NewBrowseModel creates a new browse view model.
func NewBrowseModel(dict *decomp.Dictionary, cfg *config.Config, gen *prompt.Generator, llmClient *llm.Client) BrowseModel {
	si := textinput.New()
	si.Placeholder = "Search..."
	si.CharLimit = 50
	si.Width = 30

	return BrowseModel{
		parser:      pinyin.NewParser(),
		dict:        dict,
		generator:   gen,
		config:      cfg,
		searchInput: si,
		llmClient:   llmClient,
		charPrompts: make(map[int]string),
	}
}

// SetPackage sets the Anki package to browse.
func (m *BrowseModel) SetPackage(pkg *anki.Package) {
	m.pkg = pkg
	m.charPrompts = make(map[int]string)
	m.llmPrompt = ""
	m.searchTerm = ""
	m.searchInput.SetValue("")

	if pkg == nil {
		m.notes = nil
		m.filteredNotes = nil
		m.characters = nil
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
	m.filteredNotes = notes
	m.currentNote = 0

	if len(notes) > 0 {
		m.loadCurrentNote()
	}
}

// SetSize updates the view dimensions.
func (m *BrowseModel) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update handles messages.
func (m BrowseModel) Update(msg tea.Msg) (BrowseModel, tea.Cmd) {
	var cmds []tea.Cmd

	// No package loaded - limited interaction
	if m.pkg == nil {
		return m, nil
	}

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
					return m, browseClearCopiedAfter(2 * time.Second)
				}
			}
			return m, nil
		case "B":
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

	case browseLLMResultMsg:
		m.llmGenerating = false
		if msg.err != nil {
			m.llmError = msg.err
		} else {
			m.llmPrompt = msg.prompt
			m.charPrompts[m.selected] = msg.prompt
		}
		return m, nil

	case browseBatchResultMsg:
		m.batchCompleted++
		if msg.err == nil && msg.prompt != "" {
			m.charPrompts[msg.index] = msg.prompt
			if msg.index == m.selected {
				m.llmPrompt = msg.prompt
			}
		}
		if m.batchCompleted >= m.batchTotal {
			m.batchGenerating = false
			if p, ok := m.charPrompts[m.selected]; ok {
				m.llmPrompt = p
			}
		}
		return m, nil

	case browseClearCopiedMsg:
		m.copied = false
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m *BrowseModel) loadCurrentNote() {
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

func (m *BrowseModel) analyzeChar(char string) *components.CharacterResult {
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

func (m *BrowseModel) applyFilter() {
	if m.searchTerm == "" {
		m.filteredNotes = m.notes
	} else {
		m.filteredNotes = nil
		term := strings.ToLower(m.searchTerm)
		for _, note := range m.notes {
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

func (m *BrowseModel) generateLLMPrompt() tea.Cmd {
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
		return browseLLMResultMsg{prompt: prompt, err: err}
	}
}

func (m *BrowseModel) generateBatchPrompts() tea.Cmd {
	if len(m.characters) == 0 || m.llmClient == nil {
		return nil
	}

	var cmds []tea.Cmd
	client := m.llmClient

	for i, r := range m.characters {
		if _, exists := m.charPrompts[i]; exists {
			cmds = append(cmds, func() tea.Msg {
				return browseBatchResultMsg{index: i, prompt: m.charPrompts[i], err: nil}
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
			return browseBatchResultMsg{index: idx, prompt: prompt, err: err}
		})
	}

	return tea.Batch(cmds...)
}

// View renders the browse view.
func (m BrowseModel) View() string {
	// No package loaded
	if m.pkg == nil {
		return m.renderNoPackage()
	}

	var b strings.Builder

	// Search bar
	if m.searching {
		b.WriteString(browseSearchBoxStyle.Render("Search: " + m.searchInput.View()))
		b.WriteString("\n\n")
	} else if m.searchTerm != "" {
		b.WriteString(helpStyle.Render(fmt.Sprintf("Filter: \"%s\" (press 'c' to clear)", m.searchTerm)))
		b.WriteString("\n\n")
	}

	// Card counter
	if len(m.filteredNotes) > 0 {
		counter := browseCardCountStyle.Render(
			fmt.Sprintf("Card %d of %d", m.currentNote+1, len(m.filteredNotes)),
		)
		b.WriteString(counter)
		b.WriteString("\n\n")
	}

	// Current note
	if m.currentNote < len(m.filteredNotes) {
		b.WriteString(m.renderNoteView())
	} else {
		b.WriteString(helpStyle.Render("No cards match your search"))
		b.WriteString("\n")
	}

	// Help
	b.WriteString("\n")
	helpText := "↑/↓: cards • ←/→: chars • /: search • g: generate"
	if len(m.characters) > 1 {
		helpText += " • B: batch"
	}
	if m.llmPrompt != "" {
		helpText += " • y: copy"
	}
	b.WriteString(helpStyle.Render(helpText))

	return b.String()
}

func (m BrowseModel) renderNoPackage() string {
	var b strings.Builder

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3d5a80")).
		Padding(2, 4).
		Align(lipgloss.Center)

	content := browseNoDataStyle.Render("No Anki Deck Loaded") + "\n\n" +
		helpStyle.Render("Press '3' or go to \"Open Deck\"\nto load an .apkg file")

	b.WriteString("\n\n")
	b.WriteString(box.Render(content))

	return b.String()
}

func (m BrowseModel) renderNoteView() string {
	var b strings.Builder

	note := m.filteredNotes[m.currentNote]
	fieldNames := m.pkg.GetFieldNames(note)

	// Show main fields
	for i, value := range note.Fields {
		if i >= 3 {
			break
		}
		fieldName := fmt.Sprintf("Field %d", i)
		if i < len(fieldNames) {
			fieldName = fieldNames[i]
		}
		cleanValue := stripHTMLTags(value)
		if len(cleanValue) > 80 {
			cleanValue = cleanValue[:80] + "..."
		}
		b.WriteString(browseFieldLabelStyle.Render(fieldName + ": "))
		b.WriteString(browseFieldValueStyle.Render(cleanValue))
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

func (m BrowseModel) renderCharTabs() string {
	var tabs []string

	for i, c := range m.characters {
		charWithPinyin := fmt.Sprintf("%s\n%s", c.Character, browseCharTabPinyinStyle.Render(c.Pinyin))

		var tab string
		if i == m.selected {
			tab = browseCharTabActiveStyle.Render(charWithPinyin)
		} else {
			tab = browseCharTabStyle.Render(charWithPinyin)
		}
		tabs = append(tabs, tab)
	}

	nav := ""
	if len(m.characters) > 1 {
		nav = browseWordNavStyle.Render(fmt.Sprintf("◀ %d/%d ▶", m.selected+1, len(m.characters)))
	}

	charBar := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)
	combined := lipgloss.JoinHorizontal(lipgloss.Center, charBar, "  ", nav)

	return browseWordDisplayStyle.Render(combined)
}

func (m BrowseModel) renderCharacterDetail(r components.CharacterResult) string {
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
		progress := fmt.Sprintf("Generating prompts... %d/%d", m.batchCompleted, m.batchTotal)
		b.WriteString(loadingStyle.Render(progress))
		b.WriteString("\n")
	} else if m.llmGenerating {
		b.WriteString("\n")
		b.WriteString(loadingStyle.Render("Generating image prompt..."))
		b.WriteString("\n")
	} else if m.llmError != nil {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render(m.llmError.Error()))
		b.WriteString("\n")
	} else if m.llmPrompt != "" {
		width := 70
		if m.width > 0 && m.width-10 < width {
			width = m.width - 10
		}
		headerText := actorStyle.Render("Image Prompt")
		if m.copied {
			headerText += "  " + copiedStyle.Render("✓ Copied!")
		}
		if len(m.charPrompts) > 1 {
			headerText += "  " + helpStyle.Render(fmt.Sprintf("(%d/%d generated)", len(m.charPrompts), len(m.characters)))
		}
		llmBox := llmPromptStyle.Width(width).Render(
			headerText + "\n\n" + wordWrap(m.llmPrompt, width-6),
		)
		b.WriteString(llmBox)
	} else {
		b.WriteString("\n")
		hint := "Press 'g' to generate image prompt"
		if len(m.characters) > 1 {
			hint += " or 'B' for batch"
		}
		b.WriteString(helpStyle.Render(hint))
		b.WriteString("\n")
	}

	return b.String()
}

func (m BrowseModel) renderHMMBox(r components.CharacterResult) string {
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

func (m BrowseModel) renderComponentsBox(r components.CharacterResult) string {
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
