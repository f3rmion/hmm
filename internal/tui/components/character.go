// Package components provides shared UI components for the TUI.
package components

import (
	"github.com/f3rmion/hmm/internal/hmm"
)

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
