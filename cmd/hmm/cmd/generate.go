package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/f3rmion/hmm/internal/config"
	"github.com/f3rmion/hmm/internal/decomp"
	"github.com/f3rmion/hmm/internal/pinyin"
	"github.com/f3rmion/hmm/internal/prompt"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate <character>",
	Short: "Generate an image prompt for a character's HMM scene",
	Long: `Generate an image prompt for a Chinese character by combining:
  - Your actor (from pinyin initial)
  - Your set/location (from pinyin final)
  - Your tone room (from tone)
  - Your props (from character components)

The prompt can be used with DALL-E, Midjourney, Stable Diffusion, etc.

Examples:
  hmm generate 好
  hmm generate 林 --style midjourney
  hmm generate 中 --reading 1  # Use first reading if multiple`,
	Args: cobra.MinimumNArgs(1),
	RunE: runGenerate,
}

var (
	generateStyle   string
	generateReading int
	generateVerbose bool
)

func init() {
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringVarP(&generateStyle, "style", "s", "default", "Prompt style: default, midjourney, dalle, sd")
	generateCmd.Flags().IntVarP(&generateReading, "reading", "r", 0, "Which reading to use (0 = first, 1 = second, etc.)")
	generateCmd.Flags().BoolVarP(&generateVerbose, "verbose", "v", false, "Show detailed breakdown")
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Load dictionary for decomposition
	if err := loadDictionary(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load dictionary: %v\n", err)
	}

	// Load user config
	configDir := getConfigDir()
	cfg, err := loadUserConfig(configDir)
	if err != nil {
		// Config not found - use empty config with warnings
		fmt.Fprintf(os.Stderr, "Note: Config not found at %s. Run 'hmm init' to create config.\n", configDir)
		fmt.Fprintf(os.Stderr, "Generating prompt with placeholder values...\n\n")
		cfg = &config.Config{}
	}

	// Create prompt generator
	gen := prompt.NewGenerator(cfg.Actors, cfg.Sets, cfg.Props)

	// Set template based on style
	switch generateStyle {
	case "midjourney", "mj":
		if err := gen.SetTemplate(prompt.MidjourneyTemplate); err != nil {
			return err
		}
		gen.SetStyle(prompt.Style{
			Name:        "cinematic",
			AspectRatio: "16:9",
			Suffix:      "",
		})
	case "dalle", "openai":
		if err := gen.SetTemplate(prompt.DALLETemplate); err != nil {
			return err
		}
		gen.SetStyle(prompt.Style{
			Name:   "digital art",
			Suffix: "highly detailed, dramatic lighting",
		})
	case "sd", "stable-diffusion":
		if err := gen.SetTemplate(prompt.StableDiffusionTemplate); err != nil {
			return err
		}
		gen.SetStyle(prompt.Style{
			Name:   "cinematic lighting",
			Suffix: "8k uhd, detailed",
		})
	}

	parser := pinyin.NewParser()
	input := args[0]

	for _, char := range input {
		charStr := string(char)

		// Get pinyin readings
		readings := parser.ParseChar(charStr)
		if readings == nil || len(readings) == 0 {
			fmt.Fprintf(os.Stderr, "Warning: No pinyin found for %s\n", charStr)
			continue
		}

		// Select reading
		readingIdx := generateReading
		if readingIdx >= len(readings) {
			readingIdx = 0
		}
		reading := readings[readingIdx]

		// Get decomposition info
		var meaning, etymology, decompStr string
		var components []string

		if dict != nil {
			if entry := dict.Lookup(charStr); entry != nil {
				meaning = entry.Definition
				if entry.Etymology != nil {
					if entry.Etymology.Hint != "" {
						etymology = entry.Etymology.Hint
					} else {
						etymology = entry.Etymology.Type
					}
				}
				decompStr = decomp.FormatDecomposition(entry.Decomposition)
				components = decomp.ExtractComponents(entry.Decomposition)
			}
		}

		// Build scene data
		actorID := pinyin.GetActorID(reading.Initial)
		setID := pinyin.GetSetID(reading.Final)

		sceneData := gen.BuildSceneData(
			charStr,
			reading.Full,
			actorID,
			setID,
			reading.Tone,
			components,
			meaning,
			etymology,
			decompStr,
		)

		// Show verbose breakdown if requested
		if generateVerbose {
			fmt.Printf("Character: %s (%s)\n", charStr, reading.Full)
			fmt.Printf("Meaning: %s\n", meaning)
			fmt.Printf("Components: %v\n", components)
			fmt.Println()
			fmt.Printf("HMM Breakdown:\n")
			fmt.Printf("  Initial: %s → Actor ID: %s", displayInitial(reading.Initial), actorID)
			if sceneData.Actor != nil && sceneData.Actor.Name != "" {
				fmt.Printf(" → %s", sceneData.Actor.Name)
			} else {
				fmt.Printf(" → (not configured)")
			}
			fmt.Println()

			fmt.Printf("  Final: %s → Set ID: %s", displayFinal(reading.Final), setID)
			if sceneData.Set != nil && sceneData.Set.Name != "" {
				fmt.Printf(" → %s", sceneData.Set.Name)
			} else {
				fmt.Printf(" → (not configured)")
			}
			fmt.Println()

			fmt.Printf("  Tone: %d → Room: %s\n", reading.Tone, sceneData.ToneRoom)

			fmt.Printf("  Props:\n")
			for _, comp := range components {
				prop := gen.GetProp(comp)
				if prop != nil && prop.Name != "" {
					fmt.Printf("    %s → %s\n", comp, prop.Name)
				} else {
					fmt.Printf("    %s → (not configured)\n", comp)
				}
			}
			fmt.Println()
			fmt.Println("Generated Prompt:")
			fmt.Println("─────────────────")
		}

		// Generate prompt
		promptText, err := gen.Generate(sceneData)
		if err != nil {
			return fmt.Errorf("generating prompt for %s: %w", charStr, err)
		}

		fmt.Println(promptText)

		if len(input) > 1 {
			fmt.Println()
		}
	}

	return nil
}

func loadUserConfig(configDir string) (*config.Config, error) {
	actorsPath := filepath.Join(configDir, "actors.yaml")
	setsPath := filepath.Join(configDir, "sets.yaml")
	propsPath := filepath.Join(configDir, "props.yaml")

	// Check if config files exist
	if _, err := os.Stat(actorsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config not found in %s", configDir)
	}

	actors, err := config.LoadActors(actorsPath)
	if err != nil {
		return nil, err
	}

	sets, err := config.LoadSets(setsPath)
	if err != nil {
		return nil, err
	}

	props, err := config.LoadProps(propsPath)
	if err != nil {
		return nil, err
	}

	return &config.Config{
		Actors: actors,
		Sets:   sets,
		Props:  props,
	}, nil
}
