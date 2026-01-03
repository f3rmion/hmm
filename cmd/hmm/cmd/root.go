// Package cmd contains all CLI commands for the HMM tool.
package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/f3rmion/hmm/internal/config"
	"github.com/f3rmion/hmm/internal/decomp"
	"github.com/f3rmion/hmm/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hmm",
	Short: "Hanzi Movie Method - Learn Chinese characters with mnemonics",
	Long: `HMM (Hanzi Movie Method) is a CLI tool for learning Chinese characters
using the movie method mnemonic system.

The system maps:
  - Actors (people) → Pinyin initials (55 total)
  - Sets (locations) → Pinyin finals (13 total)
  - Rooms (areas) → Tones (5 total)
  - Props (objects) → Character components

Each character becomes a memorable movie scene combining these elements.

Running 'hmm' without arguments launches the interactive TUI.`,
	RunE: runUnifiedTUI,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config directory (default is $HOME/.config/hmm)")
	rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")

	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.Set("config_dir", cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error finding home directory:", err)
			os.Exit(1)
		}

		configDir := filepath.Join(home, ".config", "hmm")
		viper.Set("config_dir", configDir)
	}

	viper.SetEnvPrefix("HMM")
	viper.AutomaticEnv()
}

// getConfigDir returns the configuration directory path.
func getConfigDir() string {
	return viper.GetString("config_dir")
}

// runUnifiedTUI launches the unified TUI application.
func runUnifiedTUI(cmd *cobra.Command, args []string) error {
	// Ensure config directory is set up
	configDir := getConfigDir()
	ensureConfigSetup(configDir)

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

	// Load user config from ~/.config/hmm/
	cfg, err := loadUserConfig(configDir)
	if err != nil {
		// Config not available, use empty config
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

// ensureConfigSetup creates the config directory and copies default files if needed.
func ensureConfigSetup(configDir string) {
	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return
	}

	// Create anki subdirectory
	ankiDir := filepath.Join(configDir, "anki")
	if err := os.MkdirAll(ankiDir, 0755); err != nil {
		return
	}

	// Copy config files if they don't exist
	configFiles := []string{"actors.yaml", "sets.yaml", "props.yaml"}
	for _, file := range configFiles {
		destPath := filepath.Join(configDir, file)
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			// Try to copy from local config/ directory
			srcPath := filepath.Join("config", file)
			copyFile(srcPath, destPath)
		}
	}

	// Copy example Anki deck if it doesn't exist
	ankiDest := filepath.Join(ankiDir, "All_214_Chinese_Radicals.apkg")
	if _, err := os.Stat(ankiDest); os.IsNotExist(err) {
		srcPath := filepath.Join("anki", "All_214_Chinese_Radicals.apkg")
		copyFile(srcPath, ankiDest)
	}
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
