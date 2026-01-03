package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f3rmion/hmm/internal/anki"
	"github.com/f3rmion/hmm/internal/config"
	"github.com/f3rmion/hmm/internal/decomp"
	"github.com/f3rmion/hmm/internal/tui"
	"github.com/spf13/cobra"
)

var browseCmd = &cobra.Command{
	Use:   "browse <file.apkg>",
	Short: "Browse an Anki deck in the TUI",
	Long: `Load an Anki deck and browse through cards in an interactive terminal UI.

Features:
  - Navigate through cards with arrow keys
  - See HMM breakdown for each character
  - Generate image prompts
  - Filter by deck or search

Controls:
  ↑/↓ or j/k    Navigate cards
  ←/→ or h/l    Navigate characters in a card
  g             Generate image prompt
  /             Search
  Esc           Quit`,
	Args: cobra.ExactArgs(1),
	RunE: runBrowse,
}

func init() {
	rootCmd.AddCommand(browseCmd)
}

func runBrowse(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Load dictionary
	dict := decomp.NewDictionary()
	dictPaths := []string{
		"data/dictionary.jsonl",
		"/usr/local/share/hmm/dictionary.jsonl",
	}
	for _, p := range dictPaths {
		if _, err := os.Stat(p); err == nil {
			if err := dict.LoadFromFile(p); err == nil {
				break
			}
		}
	}

	// Load user config
	configDir := getConfigDir()
	cfg, err := loadUserConfig(configDir)
	if err != nil {
		cfg, _ = loadUserConfig("config")
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	// Open Anki package
	pkg, err := anki.OpenPackage(path)
	if err != nil {
		return fmt.Errorf("opening package: %w", err)
	}
	defer pkg.Close()

	fmt.Fprintf(os.Stderr, "Loaded: %s (%d notes)\n", path, len(pkg.Notes))

	// Create and run unified TUI with pre-loaded package
	p := tea.NewProgram(
		tui.NewAppWithPackage(dict, cfg, pkg, path),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
