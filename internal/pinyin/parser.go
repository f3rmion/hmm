// Package pinyin handles pinyin parsing and HMM initial/final extraction.
package pinyin

import (
	"strings"
	"unicode"

	"github.com/f3rmion/hmm/internal/hmm"
	gopinyin "github.com/mozillazg/go-pinyin"
)

// Parser handles pinyin conversion and HMM mapping.
type Parser struct {
	args gopinyin.Args
}

// NewParser creates a new pinyin parser.
func NewParser() *Parser {
	args := gopinyin.NewArgs()
	args.Style = gopinyin.Tone // Returns tone marks: zhōng
	args.Heteronym = true      // Return all possible readings
	return &Parser{args: args}
}

// ParsedPinyin contains the HMM-relevant parts of a pinyin syllable.
type ParsedPinyin struct {
	Full    string   // Full pinyin with tone mark (e.g., "hǎo")
	Initial string   // HMM initial (e.g., "h")
	Final   string   // HMM final (e.g., "ao")
	Tone    hmm.Tone // Tone number (1-5)
}

// GetPinyin returns all pinyin readings for a character.
func (p *Parser) GetPinyin(char string) []string {
	result := gopinyin.Pinyin(char, p.args)
	if len(result) == 0 {
		return nil
	}
	return result[0]
}

// Parse extracts HMM components from a pinyin syllable.
func (p *Parser) Parse(pinyin string) ParsedPinyin {
	result := ParsedPinyin{Full: pinyin}

	// Extract tone from tone mark
	result.Tone, pinyin = extractTone(pinyin)

	// Extract initial and final using HMM rules
	result.Initial, result.Final = extractInitialFinal(pinyin)

	return result
}

// ParseChar parses a character and returns all possible HMM breakdowns.
func (p *Parser) ParseChar(char string) []ParsedPinyin {
	readings := p.GetPinyin(char)
	if readings == nil {
		return nil
	}

	results := make([]ParsedPinyin, len(readings))
	for i, reading := range readings {
		results[i] = p.Parse(reading)
	}
	return results
}

// extractTone extracts the tone number and returns the pinyin without tone marks.
func extractTone(pinyin string) (hmm.Tone, string) {
	tone := hmm.ToneUnknown
	var result strings.Builder

	toneMarks := map[rune]struct {
		base rune
		tone hmm.Tone
	}{
		'ā': {'a', hmm.Tone1}, 'á': {'a', hmm.Tone2}, 'ǎ': {'a', hmm.Tone3}, 'à': {'a', hmm.Tone4},
		'ē': {'e', hmm.Tone1}, 'é': {'e', hmm.Tone2}, 'ě': {'e', hmm.Tone3}, 'è': {'e', hmm.Tone4},
		'ī': {'i', hmm.Tone1}, 'í': {'i', hmm.Tone2}, 'ǐ': {'i', hmm.Tone3}, 'ì': {'i', hmm.Tone4},
		'ō': {'o', hmm.Tone1}, 'ó': {'o', hmm.Tone2}, 'ǒ': {'o', hmm.Tone3}, 'ò': {'o', hmm.Tone4},
		'ū': {'u', hmm.Tone1}, 'ú': {'u', hmm.Tone2}, 'ǔ': {'u', hmm.Tone3}, 'ù': {'u', hmm.Tone4},
		'ǖ': {'ü', hmm.Tone1}, 'ǘ': {'ü', hmm.Tone2}, 'ǚ': {'ü', hmm.Tone3}, 'ǜ': {'ü', hmm.Tone4},
	}

	for _, r := range pinyin {
		if mark, ok := toneMarks[r]; ok {
			result.WriteRune(mark.base)
			tone = mark.tone
		} else {
			result.WriteRune(r)
		}
	}

	// If no tone mark found, it's neutral tone (5)
	if tone == hmm.ToneUnknown {
		tone = hmm.Tone5
	}

	return tone, result.String()
}

// extractInitialFinal extracts the HMM initial and final from toneless pinyin.
// This follows the HMM reorganization: 55 initials, 13 finals.
func extractInitialFinal(pinyin string) (initial, final string) {
	pinyin = strings.ToLower(pinyin)

	// HMM finals (13 total)
	// The key insight: i, u, ü are moved from finals to initials
	hmmFinals := []string{
		"ong", "ang", "eng", "ing", // 3-letter finals (ing maps to eng with floating e)
		"ai", "ei", "ao", "ou", "an", "en", // 2-letter finals
		"a", "o", "e", // 1-letter finals
	}

	// Special case: null initial syllables (start with a, o, e, or use y/w)
	// y- maps to yi- (female), w- maps to wu- (fictional)
	if strings.HasPrefix(pinyin, "y") {
		if pinyin == "yi" || pinyin == "y" {
			return "y", ""
		}
		// yi + final -> y + final (e.g., yao -> y + ao)
		initial = "y"
		rest := strings.TrimPrefix(pinyin, "y")
		if strings.HasPrefix(rest, "i") {
			rest = strings.TrimPrefix(rest, "i")
		}
		final = matchFinal(rest, hmmFinals)
		return
	}

	if strings.HasPrefix(pinyin, "w") {
		if pinyin == "wu" || pinyin == "w" {
			return "w", ""
		}
		// wu + final -> w + final (e.g., wai -> w + ai)
		initial = "w"
		rest := strings.TrimPrefix(pinyin, "w")
		if strings.HasPrefix(rest, "u") {
			rest = strings.TrimPrefix(rest, "u")
		}
		final = matchFinal(rest, hmmFinals)
		return
	}

	// Handle yu- initials (god/leader category)
	if strings.HasPrefix(pinyin, "yu") {
		if pinyin == "yu" {
			return "yu", ""
		}
		initial = "yu"
		rest := strings.TrimPrefix(pinyin, "yu")
		final = matchFinal(rest, hmmFinals)
		return
	}

	// Check for consonant clusters first (zh, ch, sh)
	consonantClusters := []string{"zh", "ch", "sh"}
	for _, cc := range consonantClusters {
		if strings.HasPrefix(pinyin, cc) {
			rest := strings.TrimPrefix(pinyin, cc)
			return extractWithInitial(cc, rest, hmmFinals)
		}
	}

	// Single consonant initials
	if len(pinyin) > 0 && isConsonant(rune(pinyin[0])) {
		consonant := string(pinyin[0])
		rest := pinyin[1:]
		return extractWithInitial(consonant, rest, hmmFinals)
	}

	// No initial (null initial) - syllable starts with vowel
	// These are a, o, e, ai, ao, an, ang, en, ou, etc.
	return "", matchFinal(pinyin, hmmFinals)
}

// extractWithInitial handles the HMM initial/final split after identifying the consonant.
func extractWithInitial(consonant, rest string, finals []string) (initial, final string) {
	// Check for HMM compound initials (consonant + i/u/ü)
	// Female: bi, pi, mi, di, ti, ni, li, ji, qi, xi (consonant + i)
	// Fictional: bu, pu, mu, fu, du, tu, nu, lu, gu, ku, hu, zhu, chu, shu, ru, zu, cu, su (consonant + u)
	// God/Leader: nü, lü, ju, qu, xu (consonant + ü)

	if len(rest) == 0 {
		// Just the consonant (shouldn't happen in valid pinyin)
		return consonant, ""
	}

	// Special handling for j, q, x - they always have ü (written as u) or i
	if consonant == "j" || consonant == "q" || consonant == "x" {
		if strings.HasPrefix(rest, "u") {
			// ju, qu, xu are god/leader (ü sound)
			if rest == "u" {
				return consonant + "u", ""
			}
			initial = consonant + "u"
			rest = strings.TrimPrefix(rest, "u")
			final = matchFinal(rest, finals)
			return
		}
		if strings.HasPrefix(rest, "i") {
			// ji, qi, xi are female
			if rest == "i" {
				return consonant + "i", ""
			}
			initial = consonant + "i"
			rest = strings.TrimPrefix(rest, "i")
			final = matchFinal(rest, finals)
			return
		}
	}

	// Check for ü (nü, lü)
	if strings.HasPrefix(rest, "ü") || strings.HasPrefix(rest, "v") {
		if rest == "ü" || rest == "v" {
			return consonant + "ü", ""
		}
		initial = consonant + "ü"
		rest = strings.TrimPrefix(rest, "ü")
		rest = strings.TrimPrefix(rest, "v")
		final = matchFinal(rest, finals)
		return
	}

	// Check for i (female actors) - but NOT for "fake i" (zhi, chi, shi, ri, zi, ci, si)
	fakeI := map[string]bool{"zh": true, "ch": true, "sh": true, "r": true, "z": true, "c": true, "s": true}
	if strings.HasPrefix(rest, "i") && !fakeI[consonant] {
		if rest == "i" {
			return consonant + "i", ""
		}
		initial = consonant + "i"
		rest = strings.TrimPrefix(rest, "i")
		final = matchFinal(rest, finals)
		return
	}

	// Check for u (fictional actors)
	if strings.HasPrefix(rest, "u") && consonant != "j" && consonant != "q" && consonant != "x" {
		if rest == "u" {
			return consonant + "u", ""
		}
		initial = consonant + "u"
		rest = strings.TrimPrefix(rest, "u")
		final = matchFinal(rest, finals)
		return
	}

	// Plain consonant initial (male actors)
	return consonant, matchFinal(rest, finals)
}

// matchFinal finds the matching HMM final from the remaining pinyin.
func matchFinal(rest string, finals []string) string {
	// Try to match longest final first
	for _, f := range finals {
		if rest == f {
			return f
		}
	}

	// Handle special cases
	// ing -> (e)ng final (floating e drops)
	if strings.HasSuffix(rest, "ing") {
		return "eng"
	}
	// in -> (e)n final (floating e drops)
	if rest == "in" || strings.HasSuffix(rest, "in") {
		return "en"
	}
	// un -> (e)n final
	if rest == "un" || strings.HasSuffix(rest, "un") {
		return "en"
	}
	// iong -> ong
	if rest == "iong" || strings.HasSuffix(rest, "iong") {
		return "ong"
	}

	// After stripping i/u from compound initials, we may have:
	// "n" (from lin -> li + n) -> maps to (e)n set
	// "ng" (from ling -> li + ng) -> maps to (e)ng set
	if rest == "n" {
		return "en"
	}
	if rest == "ng" {
		return "eng"
	}

	// If no match, return as-is (might be null final)
	if rest == "" || rest == "i" || rest == "u" || rest == "ü" {
		return ""
	}

	return rest
}

// isConsonant checks if a rune is a pinyin consonant.
func isConsonant(r rune) bool {
	consonants := "bpmfdtnlgkhjqxzhchshrzcs"
	return strings.ContainsRune(consonants, unicode.ToLower(r))
}

// GetActorID returns the actor ID for a given HMM initial.
func GetActorID(initial string) string {
	if initial == "" {
		return "null"
	}
	// Handle ü variations
	initial = strings.ReplaceAll(initial, "ü", "v")
	return initial
}

// GetSetID returns the set ID for a given HMM final.
func GetSetID(final string) string {
	if final == "" {
		return "null"
	}
	return final
}
