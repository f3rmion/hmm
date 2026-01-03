package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize HMM configuration",
	Long: `Initialize HMM configuration files in your config directory.

This creates template YAML files for:
  - actors.yaml   (55 pinyin initials → people)
  - sets.yaml     (13 pinyin finals → locations)
  - props.yaml    (character components → objects)

You should then edit these files to add your personal actors, sets, and props.`,
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().Bool("force", false, "overwrite existing configuration")
}

func runInit(cmd *cobra.Command, args []string) error {
	force, _ := cmd.Flags().GetBool("force")
	configDir := getConfigDir()

	// Check if config already exists
	if _, err := os.Stat(configDir); err == nil && !force {
		return fmt.Errorf("config directory already exists: %s\nUse --force to overwrite", configDir)
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	fmt.Printf("Initializing HMM configuration in %s\n\n", configDir)

	// Copy template files
	files := []string{"actors.yaml", "sets.yaml", "props.yaml"}
	for _, file := range files {
		if err := copyConfigFile(configDir, file); err != nil {
			return err
		}
		fmt.Printf("  Created %s\n", file)
	}

	fmt.Println()
	fmt.Println("Configuration initialized!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Edit the YAML files to add your personal actors, sets, and props")
	fmt.Printf("  2. Run 'hmm lookup <character>' to test a character lookup\n")
	fmt.Printf("  3. Run 'hmm generate <character>' to generate a mnemonic prompt\n")

	return nil
}

func copyConfigFile(configDir, filename string) error {
	// For now, we'll create placeholder files
	// In a full implementation, these would be embedded or downloaded

	var content string
	switch filename {
	case "actors.yaml":
		content = actorsTemplate
	case "sets.yaml":
		content = setsTemplate
	case "props.yaml":
		content = propsTemplate
	default:
		return fmt.Errorf("unknown config file: %s", filename)
	}

	destPath := filepath.Join(configDir, filename)
	return os.WriteFile(destPath, []byte(content), 0644)
}

// Templates will be loaded from the config directory in the project
// For the init command, we embed minimal templates

const actorsTemplate = `# Hanzi Movie Method - Actors Configuration
# 55 Actors representing Pinyin Initials
#
# Categories:
# - male: Real men (basic consonant initials)
# - female: Real women (consonant+i initials)
# - fictional: Fictional characters (consonant+u initials)
# - god_leader: Gods or world leaders (consonant+ü initials)
# - null: No initial

actors:
  # NULL INITIAL
  - id: "null"
    initial: ""
    category: null
    name: ""  # e.g., Jackie Chan

  # MALE ACTORS (18)
  - id: "b"
    initial: "b"
    category: male
    name: ""
  - id: "p"
    initial: "p"
    category: male
    name: ""
  - id: "m"
    initial: "m"
    category: male
    name: ""
  - id: "f"
    initial: "f"
    category: male
    name: ""
  - id: "d"
    initial: "d"
    category: male
    name: ""
  - id: "t"
    initial: "t"
    category: male
    name: ""
  - id: "n"
    initial: "n"
    category: male
    name: ""
  - id: "l"
    initial: "l"
    category: male
    name: ""
  - id: "g"
    initial: "g"
    category: male
    name: ""
  - id: "k"
    initial: "k"
    category: male
    name: ""
  - id: "h"
    initial: "h"
    category: male
    name: ""
  - id: "zh"
    initial: "zh"
    category: male
    name: ""
  - id: "ch"
    initial: "ch"
    category: male
    name: ""
  - id: "sh"
    initial: "sh"
    category: male
    name: ""
  - id: "r"
    initial: "r"
    category: male
    name: ""
  - id: "z"
    initial: "z"
    category: male
    name: ""
  - id: "c"
    initial: "c"
    category: male
    name: ""
  - id: "s"
    initial: "s"
    category: male
    name: ""

  # FEMALE ACTORS (11)
  - id: "y"
    initial: "y"
    category: female
    name: ""
  - id: "bi"
    initial: "bi"
    category: female
    name: ""
  - id: "pi"
    initial: "pi"
    category: female
    name: ""
  - id: "mi"
    initial: "mi"
    category: female
    name: ""
  - id: "di"
    initial: "di"
    category: female
    name: ""
  - id: "ti"
    initial: "ti"
    category: female
    name: ""
  - id: "ni"
    initial: "ni"
    category: female
    name: ""
  - id: "li"
    initial: "li"
    category: female
    name: ""
  - id: "ji"
    initial: "ji"
    category: female
    name: ""
  - id: "qi"
    initial: "qi"
    category: female
    name: ""
  - id: "xi"
    initial: "xi"
    category: female
    name: ""

  # FICTIONAL ACTORS (19)
  - id: "w"
    initial: "w"
    category: fictional
    name: ""
  - id: "bu"
    initial: "bu"
    category: fictional
    name: ""
  - id: "pu"
    initial: "pu"
    category: fictional
    name: ""
  - id: "mu"
    initial: "mu"
    category: fictional
    name: ""
  - id: "fu"
    initial: "fu"
    category: fictional
    name: ""
  - id: "du"
    initial: "du"
    category: fictional
    name: ""
  - id: "tu"
    initial: "tu"
    category: fictional
    name: ""
  - id: "nu"
    initial: "nu"
    category: fictional
    name: ""
  - id: "lu"
    initial: "lu"
    category: fictional
    name: ""
  - id: "gu"
    initial: "gu"
    category: fictional
    name: ""
  - id: "ku"
    initial: "ku"
    category: fictional
    name: ""
  - id: "hu"
    initial: "hu"
    category: fictional
    name: ""
  - id: "zhu"
    initial: "zhu"
    category: fictional
    name: ""
  - id: "chu"
    initial: "chu"
    category: fictional
    name: ""
  - id: "shu"
    initial: "shu"
    category: fictional
    name: ""
  - id: "ru"
    initial: "ru"
    category: fictional
    name: ""
  - id: "zu"
    initial: "zu"
    category: fictional
    name: ""
  - id: "cu"
    initial: "cu"
    category: fictional
    name: ""
  - id: "su"
    initial: "su"
    category: fictional
    name: ""

  # GODS/LEADERS (6)
  - id: "yu"
    initial: "yu"
    category: god_leader
    name: ""
  - id: "nv"
    initial: "nü"
    category: god_leader
    name: ""
  - id: "lv"
    initial: "lü"
    category: god_leader
    name: ""
  - id: "ju"
    initial: "ju"
    category: god_leader
    name: ""
  - id: "qu"
    initial: "qu"
    category: god_leader
    name: ""
  - id: "xu"
    initial: "xu"
    category: god_leader
    name: ""
`

const setsTemplate = `# Hanzi Movie Method - Sets Configuration
# 13 Sets representing Pinyin Finals
#
# Each set has 5 tone rooms:
# - Tone 1: Outside entrance
# - Tone 2: Kitchen/Hallway
# - Tone 3: Bedroom/Living room
# - Tone 4: Bathroom/Backyard
# - Tone 5: Roof

sets:
  - id: "null"
    final: ""
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "a"
    final: "a"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "o"
    final: "o"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "e"
    final: "e"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "ai"
    final: "ai"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "ei"
    final: "ei"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "ao"
    final: "ao"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "ou"
    final: "ou"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "an"
    final: "an"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "ang"
    final: "ang"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "en"
    final: "en"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "eng"
    final: "eng"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"

  - id: "ong"
    final: "ong"
    name: ""
    rooms:
      - tone: 1
        name: "Outside entrance"
      - tone: 2
        name: "Kitchen"
      - tone: 3
        name: "Bedroom"
      - tone: 4
        name: "Bathroom"
      - tone: 5
        name: "Roof"
`

const propsTemplate = `# Hanzi Movie Method - Props Configuration
# Props represent character components (radicals)
#
# Choose props based on:
# - appearance: what does it look like?
# - meaning: what does it mean?
# - combination: both

props:
  # Basic strokes
  - id: "一"
    component: "一"
    name: ""
    meaning: "one"
  - id: "丨"
    component: "丨"
    name: ""
    meaning: "vertical stroke"
  - id: "丿"
    component: "丿"
    name: ""
    meaning: "left-falling stroke"

  # Common radicals
  - id: "人"
    component: "人"
    name: ""
    meaning: "person"
  - id: "口"
    component: "口"
    name: ""
    meaning: "mouth"
  - id: "日"
    component: "日"
    name: ""
    meaning: "sun"
  - id: "月"
    component: "月"
    name: ""
    meaning: "moon"
  - id: "木"
    component: "木"
    name: ""
    meaning: "tree"
  - id: "水"
    component: "水"
    name: ""
    meaning: "water"
  - id: "火"
    component: "火"
    name: ""
    meaning: "fire"
  - id: "土"
    component: "土"
    name: ""
    meaning: "earth"
  - id: "金"
    component: "金"
    name: ""
    meaning: "gold/metal"
  - id: "心"
    component: "心"
    name: ""
    meaning: "heart"
  - id: "手"
    component: "手"
    name: ""
    meaning: "hand"
  - id: "女"
    component: "女"
    name: ""
    meaning: "woman"
  - id: "子"
    component: "子"
    name: ""
    meaning: "child"
`
