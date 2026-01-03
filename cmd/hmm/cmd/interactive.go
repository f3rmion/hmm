package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f3rmion/hmm/internal/config"
	"github.com/f3rmion/hmm/internal/decomp"
	"github.com/f3rmion/hmm/internal/tui"
	"github.com/spf13/cobra"
)

var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"i", "ui"},
	Short:   "Launch interactive TUI",
	Long: `Launch an interactive terminal UI for exploring Chinese characters.

Features:
  - Type any Chinese character to see its HMM breakdown
  - View actor, set, tone room, and props
  - Generate image prompts
  - Beautiful colorful interface

Controls:
  Enter   Analyze character
  Esc     Quit`,
	RunE: runInteractive,
}

func init() {
	rootCmd.AddCommand(interactiveCmd)
}

func runInteractive(cmd *cobra.Command, args []string) error {
	// Load dictionary
	dict := decomp.NewDictionary()
	dictPaths := []string{
		"data/dictionary.jsonl",
		"/usr/local/share/hmm/dictionary.jsonl",
	}
	for _, path := range dictPaths {
		if _, err := os.Stat(path); err == nil {
			if err := dict.LoadFromFile(path); err == nil {
				break
			}
		}
	}

	// Load user config
	configDir := getConfigDir()
	cfg, err := loadUserConfig(configDir)
	if err != nil {
		// Try local config
		cfg, _ = loadUserConfig("config")
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	// Create and run unified TUI
	p := tea.NewProgram(
		tui.NewApp(dict, cfg),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running TUI: %w", err)
	}

	return nil
}
