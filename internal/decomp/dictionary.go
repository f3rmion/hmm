// Package decomp handles Chinese character decomposition using Make Me a Hanzi data.
package decomp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/f3rmion/hmm/internal/hmm"
)

// DictionaryEntry represents a single entry from Make Me a Hanzi dictionary.
type DictionaryEntry struct {
	Character     string       `json:"character"`
	Definition    string       `json:"definition"`
	Pinyin        []string     `json:"pinyin"`
	Decomposition string       `json:"decomposition"`
	Etymology     *Etymology   `json:"etymology,omitempty"`
	Radical       string       `json:"radical"`
	Matches       [][]int      `json:"matches,omitempty"`
}

// Etymology from Make Me a Hanzi.
type Etymology struct {
	Type     string `json:"type"`     // pictophonetic, pictographic, ideographic
	Semantic string `json:"semantic,omitempty"` // meaning component
	Phonetic string `json:"phonetic,omitempty"` // sound component
	Hint     string `json:"hint,omitempty"`
}

// Dictionary holds all character data.
type Dictionary struct {
	entries map[string]*DictionaryEntry
}

// NewDictionary creates an empty dictionary.
func NewDictionary() *Dictionary {
	return &Dictionary{
		entries: make(map[string]*DictionaryEntry),
	}
}

// LoadFromFile loads the dictionary from a Make Me a Hanzi dictionary.txt file.
func (d *Dictionary) LoadFromFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening dictionary file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry DictionaryEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			// Skip malformed entries
			continue
		}

		d.entries[entry.Character] = &entry
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading dictionary file: %w", err)
	}

	return nil
}

// Lookup returns the dictionary entry for a character.
func (d *Dictionary) Lookup(char string) *DictionaryEntry {
	return d.entries[char]
}

// Size returns the number of entries in the dictionary.
func (d *Dictionary) Size() int {
	return len(d.entries)
}

// ToHanziEntry converts a DictionaryEntry to an hmm.HanziEntry.
func (e *DictionaryEntry) ToHanziEntry() *hmm.HanziEntry {
	var etymology *hmm.Etymology
	if e.Etymology != nil {
		etymology = &hmm.Etymology{
			Type:     e.Etymology.Type,
			Semantic: e.Etymology.Semantic,
			Phonetic: e.Etymology.Phonetic,
			Hint:     e.Etymology.Hint,
		}
	}

	return &hmm.HanziEntry{
		Character:     e.Character,
		Pinyin:        e.Pinyin,
		Definition:    e.Definition,
		Decomposition: e.Decomposition,
		Components:    ExtractComponents(e.Decomposition),
		Radical:       e.Radical,
		Etymology:     etymology,
	}
}

// IDS (Ideographic Description Sequence) characters.
// These describe how components are arranged.
var idsChars = map[rune]string{
	'⿰': "left-right",      // ⿰AB = A on left, B on right
	'⿱': "top-bottom",      // ⿱AB = A on top, B on bottom
	'⿲': "left-mid-right",  // ⿲ABC = A left, B middle, C right
	'⿳': "top-mid-bottom",  // ⿳ABC = A top, B middle, C bottom
	'⿴': "surround",        // ⿴AB = A surrounds B
	'⿵': "surround-top",    // ⿵AB = A surrounds B from top
	'⿶': "surround-bottom", // ⿶AB = A surrounds B from bottom
	'⿷': "surround-left",   // ⿷AB = A surrounds B from left
	'⿸': "surround-upper-left",
	'⿹': "surround-upper-right",
	'⿺': "surround-lower-left",
	'⿻': "overlaid",        // ⿻AB = A and B overlaid
}

// ExtractComponents extracts the component characters from an IDS decomposition string.
func ExtractComponents(decomposition string) []string {
	if decomposition == "" || decomposition == "？" {
		return nil
	}

	var components []string
	for _, r := range decomposition {
		// Skip IDS structure characters
		if _, isIDS := idsChars[r]; isIDS {
			continue
		}
		// Skip unknown marker
		if r == '？' {
			continue
		}
		// Include actual character components
		if unicode.Is(unicode.Han, r) || isRadicalChar(r) {
			components = append(components, string(r))
		}
	}

	return components
}

// isRadicalChar checks if a rune is a CJK radical character.
func isRadicalChar(r rune) bool {
	// CJK Radicals Supplement: U+2E80–U+2EFF
	// Kangxi Radicals: U+2F00–U+2FDF
	// CJK Radicals: part of CJK Unified Ideographs
	return (r >= 0x2E80 && r <= 0x2EFF) || (r >= 0x2F00 && r <= 0x2FDF)
}

// GetDecompositionType returns a human-readable description of the IDS structure.
func GetDecompositionType(decomposition string) string {
	if decomposition == "" || decomposition == "？" {
		return "unknown"
	}

	for _, r := range decomposition {
		if desc, ok := idsChars[r]; ok {
			return desc
		}
	}

	return "simple"
}

// GetComponentPositions returns a description of where each component appears.
func GetComponentPositions(decomposition string) map[string]string {
	positions := make(map[string]string)

	if decomposition == "" || decomposition == "？" {
		return positions
	}

	// Parse the IDS structure
	runes := []rune(decomposition)
	if len(runes) == 0 {
		return positions
	}

	// Simple parsing for common structures
	structure := ""
	var components []string

	for _, r := range runes {
		if desc, isIDS := idsChars[r]; isIDS {
			structure = desc
		} else if r != '？' && (unicode.Is(unicode.Han, r) || isRadicalChar(r)) {
			components = append(components, string(r))
		}
	}

	// Assign positions based on structure
	switch structure {
	case "left-right":
		if len(components) >= 2 {
			positions[components[0]] = "left"
			positions[components[1]] = "right"
		}
	case "top-bottom":
		if len(components) >= 2 {
			positions[components[0]] = "top"
			positions[components[1]] = "bottom"
		}
	case "left-mid-right":
		if len(components) >= 3 {
			positions[components[0]] = "left"
			positions[components[1]] = "middle"
			positions[components[2]] = "right"
		}
	case "top-mid-bottom":
		if len(components) >= 3 {
			positions[components[0]] = "top"
			positions[components[1]] = "middle"
			positions[components[2]] = "bottom"
		}
	case "surround", "surround-top", "surround-bottom", "surround-left",
		"surround-upper-left", "surround-upper-right", "surround-lower-left":
		if len(components) >= 2 {
			positions[components[0]] = "outer"
			positions[components[1]] = "inner"
		}
	default:
		for i, comp := range components {
			positions[comp] = fmt.Sprintf("component %d", i+1)
		}
	}

	return positions
}

// FormatDecomposition returns a human-readable decomposition description.
func FormatDecomposition(decomposition string) string {
	if decomposition == "" || decomposition == "？" {
		return "No decomposition available"
	}

	structure := GetDecompositionType(decomposition)
	components := ExtractComponents(decomposition)

	if len(components) == 0 {
		return "No components found"
	}

	return fmt.Sprintf("%s: %s", structure, strings.Join(components, " + "))
}
