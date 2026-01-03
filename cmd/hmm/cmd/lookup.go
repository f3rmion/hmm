package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/f3rmion/hmm/internal/decomp"
	"github.com/f3rmion/hmm/internal/hmm"
	"github.com/f3rmion/hmm/internal/pinyin"
	"github.com/spf13/cobra"
)

var lookupCmd = &cobra.Command{
	Use:   "lookup <character>",
	Short: "Look up pinyin and HMM breakdown for a character",
	Long: `Look up a Chinese character and display its:
  - Pinyin reading(s)
  - HMM initial (actor)
  - HMM final (set)
  - Tone (room)
  - Character components (props)
  - Etymology

Example:
  hmm lookup 好
  hmm lookup 中国`,
	Args: cobra.MinimumNArgs(1),
	RunE: runLookup,
}

var dict *decomp.Dictionary

func init() {
	rootCmd.AddCommand(lookupCmd)
}

func loadDictionary() error {
	if dict != nil {
		return nil
	}

	dict = decomp.NewDictionary()

	// Try to find dictionary file
	paths := []string{
		"data/dictionary.jsonl",
		filepath.Join(getConfigDir(), "dictionary.jsonl"),
	}

	// Also check relative to executable
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "data", "dictionary.jsonl"))
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return dict.LoadFromFile(path)
		}
	}

	// Dictionary not found - that's okay, we just won't show decomposition
	return nil
}

func runLookup(cmd *cobra.Command, args []string) error {
	parser := pinyin.NewParser()

	// Try to load dictionary for decomposition info
	if err := loadDictionary(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load dictionary: %v\n", err)
	}

	input := args[0]

	fmt.Printf("Looking up: %s\n\n", input)

	for _, char := range input {
		charStr := string(char)
		readings := parser.ParseChar(charStr)

		fmt.Printf("Character: %s\n", charStr)

		// Show dictionary info if available
		if dict != nil {
			if entry := dict.Lookup(charStr); entry != nil {
				if entry.Definition != "" {
					fmt.Printf("  Meaning: %s\n", entry.Definition)
				}
				if entry.Decomposition != "" && entry.Decomposition != "？" {
					fmt.Printf("  Structure: %s\n", decomp.FormatDecomposition(entry.Decomposition))
					components := decomp.ExtractComponents(entry.Decomposition)
					if len(components) > 0 {
						fmt.Printf("  Components (Props): %s\n", strings.Join(components, ", "))
					}
				}
				if entry.Etymology != nil {
					fmt.Printf("  Etymology: %s", entry.Etymology.Type)
					if entry.Etymology.Hint != "" {
						fmt.Printf(" - %s", entry.Etymology.Hint)
					}
					fmt.Println()
					if entry.Etymology.Semantic != "" {
						fmt.Printf("    Semantic: %s\n", entry.Etymology.Semantic)
					}
					if entry.Etymology.Phonetic != "" {
						fmt.Printf("    Phonetic: %s\n", entry.Etymology.Phonetic)
					}
				}
				if entry.Radical != "" {
					fmt.Printf("  Radical: %s\n", entry.Radical)
				}
			}
		}

		// Show pinyin breakdown
		if readings == nil {
			fmt.Printf("  Pinyin: (not found)\n")
		} else {
			fmt.Println("  ---")
			fmt.Println("  HMM Breakdown:")
			for i, r := range readings {
				if i > 0 {
					fmt.Println("  ---")
				}
				fmt.Printf("    Pinyin:  %s\n", r.Full)
				fmt.Printf("    Initial: %s → Actor: %s\n", displayInitial(r.Initial), pinyin.GetActorID(r.Initial))
				fmt.Printf("    Final:   %s → Set: %s\n", displayFinal(r.Final), pinyin.GetSetID(r.Final))
				fmt.Printf("    Tone:    %d → Room: %s\n", r.Tone, toneRoomName(r.Tone))
			}
		}
		fmt.Println()
	}

	return nil
}

func displayInitial(initial string) string {
	if initial == "" {
		return "Ø (null)"
	}
	return initial
}

func displayFinal(final string) string {
	if final == "" {
		return "Ø (null)"
	}
	return final
}

func toneRoomName(tone hmm.Tone) string {
	switch tone {
	case 1:
		return "Outside entrance"
	case 2:
		return "Kitchen/Hallway"
	case 3:
		return "Bedroom/Living room"
	case 4:
		return "Bathroom/Backyard"
	case 5:
		return "Roof"
	default:
		return "Unknown"
	}
}
