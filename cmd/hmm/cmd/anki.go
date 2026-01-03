package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/f3rmion/hmm/internal/anki"
	"github.com/f3rmion/hmm/internal/config"
	"github.com/f3rmion/hmm/internal/decomp"
	"github.com/f3rmion/hmm/internal/hmm"
	"github.com/f3rmion/hmm/internal/pinyin"
	"github.com/f3rmion/hmm/internal/prompt"
	"github.com/spf13/cobra"
)

// CharacterHMM holds HMM data for a single character.
type CharacterHMM struct {
	Char       string   `json:"char"`
	Pinyin     string   `json:"pinyin"`
	Meaning    string   `json:"meaning,omitempty"`
	Initial    string   `json:"initial"`
	Final      string   `json:"final"`
	Tone       int      `json:"tone"`
	ActorID    string   `json:"actor_id"`
	ActorName  string   `json:"actor_name,omitempty"`
	SetID      string   `json:"set_id"`
	SetName    string   `json:"set_name,omitempty"`
	ToneRoom   string   `json:"tone_room"`
	Components []string `json:"components,omitempty"`
	Props      []string `json:"props,omitempty"`
}

// AugmentedNote holds the augmented data for a note.
type AugmentedNote struct {
	NoteID    int64             `json:"note_id"`
	Character string            `json:"character"`
	Original  map[string]string `json:"original_fields"`
	HMM       []CharacterHMM    `json:"hmm"`
	Prompt    string            `json:"prompt,omitempty"`
}

var ankiCmd = &cobra.Command{
	Use:   "anki",
	Short: "Work with Anki decks",
	Long:  `Commands for reading and augmenting Anki .apkg files with HMM data.`,
}

var ankiInspectCmd = &cobra.Command{
	Use:   "inspect <file.apkg>",
	Short: "Inspect an Anki deck",
	Long: `Inspect an Anki .apkg file to see its structure:
  - Decks
  - Note types (models) and their fields
  - Sample notes

Example:
  hmm anki inspect chinese.apkg`,
	Args: cobra.ExactArgs(1),
	RunE: runAnkiInspect,
}

var ankiAugmentCmd = &cobra.Command{
	Use:   "augment <file.apkg>",
	Short: "Augment Anki notes with HMM data",
	Long: `Read an Anki deck and generate HMM data for Chinese characters.

This command:
1. Reads the .apkg file
2. Finds fields containing Chinese characters
3. Generates HMM breakdown (actor, set, room, props)
4. Outputs augmented data (JSON or CSV)

Examples:
  hmm anki augment chinese.apkg
  hmm anki augment chinese.apkg --field "Hanzi"
  hmm anki augment chinese.apkg --output augmented.json`,
	Args: cobra.ExactArgs(1),
	RunE: runAnkiAugment,
}

var (
	ankiInspectLimit   int
	ankiAugmentField   string
	ankiAugmentOutput  string
	ankiAugmentFormat  string
	ankiAugmentWritePkg bool
)

func init() {
	rootCmd.AddCommand(ankiCmd)
	ankiCmd.AddCommand(ankiInspectCmd)
	ankiCmd.AddCommand(ankiAugmentCmd)

	ankiInspectCmd.Flags().IntVarP(&ankiInspectLimit, "limit", "n", 5, "Number of sample notes to show")

	ankiAugmentCmd.Flags().StringVarP(&ankiAugmentField, "field", "f", "", "Field name containing Chinese characters (auto-detect if not specified)")
	ankiAugmentCmd.Flags().StringVarP(&ankiAugmentOutput, "output", "o", "", "Output file (stdout if not specified)")
	ankiAugmentCmd.Flags().StringVarP(&ankiAugmentFormat, "format", "", "json", "Output format: json, csv, tsv, apkg")
	ankiAugmentCmd.Flags().BoolVar(&ankiAugmentWritePkg, "write-apkg", false, "Write augmented data back to a new .apkg file")
}

func runAnkiInspect(cmd *cobra.Command, args []string) error {
	path := args[0]

	fmt.Printf("Opening: %s\n\n", path)

	pkg, err := anki.OpenPackage(path)
	if err != nil {
		return fmt.Errorf("opening package: %w", err)
	}
	defer pkg.Close()

	// Print summary
	fmt.Print(pkg.Summary())
	fmt.Println()

	// Show field details for each model
	fmt.Println("Field Details:")
	for _, model := range pkg.Models {
		fmt.Printf("  %s:\n", model.Name)
		for _, field := range model.Fields {
			fmt.Printf("    [%d] %s\n", field.Ord, field.Name)
		}
	}
	fmt.Println()

	// Show sample notes
	fmt.Printf("Sample Notes (first %d):\n", ankiInspectLimit)
	count := 0
	for _, note := range pkg.Notes {
		if count >= ankiInspectLimit {
			break
		}

		model := pkg.GetModel(note)
		modelName := "unknown"
		if model != nil {
			modelName = model.Name
		}

		fmt.Printf("\n  Note %d (Model: %s):\n", note.ID, modelName)
		fieldNames := pkg.GetFieldNames(note)
		for i, value := range note.Fields {
			fieldName := fmt.Sprintf("Field %d", i)
			if i < len(fieldNames) {
				fieldName = fieldNames[i]
			}
			// Truncate long values
			displayValue := value
			if len(displayValue) > 100 {
				displayValue = displayValue[:100] + "..."
			}
			// Strip HTML tags for display
			displayValue = stripHTML(displayValue)
			fmt.Printf("    %s: %s\n", fieldName, displayValue)
		}
		count++
	}

	return nil
}

func runAnkiAugment(cmd *cobra.Command, args []string) error {
	path := args[0]

	// Load dictionary
	if err := loadDictionary(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not load dictionary: %v\n", err)
	}

	// Load user config
	configDir := getConfigDir()
	cfg, err := loadUserConfig(configDir)
	if err != nil {
		cfg = &config.Config{}
	}

	// Create prompt generator
	gen := prompt.NewGenerator(cfg.Actors, cfg.Sets, cfg.Props)
	parser := pinyin.NewParser()

	// Open Anki package
	pkg, err := anki.OpenPackage(path)
	if err != nil {
		return fmt.Errorf("opening package: %w", err)
	}
	defer pkg.Close()

	fmt.Fprintf(os.Stderr, "Opened: %s (%d notes)\n", path, len(pkg.Notes))

	// Determine which field to use
	targetField := ankiAugmentField
	if targetField == "" {
		// Auto-detect field with Chinese characters
		targetField = detectChineseField(pkg)
		if targetField == "" {
			return fmt.Errorf("could not auto-detect field with Chinese characters. Use --field to specify")
		}
		fmt.Fprintf(os.Stderr, "Auto-detected Chinese field: %s\n", targetField)
	}

	// Process notes
	var results []AugmentedNote

	for _, note := range pkg.Notes {
		chineseValue := pkg.GetFieldValue(note, targetField)
		if chineseValue == "" {
			continue
		}

		// Strip HTML
		chineseValue = stripHTML(chineseValue)

		// Extract Chinese characters
		chars := extractChineseChars(chineseValue)
		if len(chars) == 0 {
			continue
		}

		augmented := AugmentedNote{
			NoteID:    note.ID,
			Character: chineseValue,
			Original:  make(map[string]string),
		}

		// Store original fields
		fieldNames := pkg.GetFieldNames(note)
		for i, value := range note.Fields {
			fieldName := fmt.Sprintf("field_%d", i)
			if i < len(fieldNames) {
				fieldName = fieldNames[i]
			}
			augmented.Original[fieldName] = stripHTML(value)
		}

		// Process each character
		for _, char := range chars {
			readings := parser.ParseChar(char)
			if len(readings) == 0 {
				continue
			}

			reading := readings[0] // Use first reading

			// Get decomposition
			var meaning string
			var components []string
			if dict != nil {
				if entry := dict.Lookup(char); entry != nil {
					meaning = entry.Definition
					components = decomp.ExtractComponents(entry.Decomposition)
				}
			}

			actorID := pinyin.GetActorID(reading.Initial)
			setID := pinyin.GetSetID(reading.Final)

			actor := gen.GetActor(actorID)
			set := gen.GetSet(setID)

			hmmData := CharacterHMM{
				Char:       char,
				Pinyin:     reading.Full,
				Meaning:    meaning,
				Initial:    reading.Initial,
				Final:      reading.Final,
				Tone:       int(reading.Tone),
				ActorID:    actorID,
				SetID:      setID,
				ToneRoom:   gen.GetToneRoom(set, reading.Tone),
				Components: components,
			}

			if actor != nil {
				hmmData.ActorName = actor.Name
			}
			if set != nil {
				hmmData.SetName = set.Name
			}

			// Get prop names
			for _, comp := range components {
				if p := gen.GetProp(comp); p != nil && p.Name != "" {
					hmmData.Props = append(hmmData.Props, p.Name)
				}
			}

			augmented.HMM = append(augmented.HMM, hmmData)
		}

		// Generate combined prompt if we have data
		if len(augmented.HMM) > 0 && len(chars) == 1 {
			// Single character - generate full prompt
			hmmData := augmented.HMM[0]
			sceneData := gen.BuildSceneData(
				hmmData.Char,
				hmmData.Pinyin,
				hmmData.ActorID,
				hmmData.SetID,
				hmm.Tone(hmmData.Tone),
				hmmData.Components,
				hmmData.Meaning,
				"",
				"",
			)
			if p, err := gen.Generate(sceneData); err == nil {
				augmented.Prompt = p
			}
		}

		results = append(results, augmented)
	}

	// Output results
	var output *os.File
	if ankiAugmentOutput != "" {
		f, err := os.Create(ankiAugmentOutput)
		if err != nil {
			return fmt.Errorf("creating output file: %w", err)
		}
		defer f.Close()
		output = f
	} else {
		output = os.Stdout
	}

	// Handle apkg output format
	if ankiAugmentFormat == "apkg" || ankiAugmentWritePkg {
		return writeAugmentedApkg(pkg, results, gen, ankiAugmentOutput, path)
	}

	switch ankiAugmentFormat {
	case "json":
		encoder := json.NewEncoder(output)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(results); err != nil {
			return fmt.Errorf("encoding JSON: %w", err)
		}
	case "csv", "tsv":
		sep := ","
		if ankiAugmentFormat == "tsv" {
			sep = "\t"
		}
		// Header
		fmt.Fprintf(output, "note_id%scharacter%spinyin%smeaning%sinitial%sfinal%stone%sactor_id%sactor_name%sset_id%sset_name%stone_room%scomponents%sprops%sprompt\n",
			sep, sep, sep, sep, sep, sep, sep, sep, sep, sep, sep, sep, sep, sep)
		// Data
		for _, r := range results {
			for _, h := range r.HMM {
				fmt.Fprintf(output, "%d%s%s%s%s%s%s%s%s%s%s%s%d%s%s%s%s%s%s%s%s%s%s%s%s%s%s\n",
					r.NoteID, sep,
					h.Char, sep,
					h.Pinyin, sep,
					h.Meaning, sep,
					h.Initial, sep,
					h.Final, sep,
					h.Tone, sep,
					h.ActorID, sep,
					h.ActorName, sep,
					h.SetID, sep,
					h.SetName, sep,
					h.ToneRoom, sep,
					strings.Join(h.Components, ";"), sep,
					strings.Join(h.Props, ";"), sep,
					r.Prompt,
				)
			}
		}
	default:
		return fmt.Errorf("unknown format: %s", ankiAugmentFormat)
	}

	fmt.Fprintf(os.Stderr, "Processed %d notes with Chinese characters\n", len(results))

	return nil
}

// writeAugmentedApkg writes the augmented data back to a new .apkg file.
func writeAugmentedApkg(pkg *anki.Package, results []AugmentedNote, gen *prompt.Generator, outputPath, inputPath string) error {
	// Determine output path
	if outputPath == "" {
		// Default: input_hmm.apkg
		ext := filepath.Ext(inputPath)
		base := strings.TrimSuffix(inputPath, ext)
		outputPath = base + "_hmm" + ext
	}

	// Add HMM fields to all models that have notes with Chinese
	modelsToUpdate := make(map[int64]bool)
	for _, r := range results {
		note := pkg.GetNoteByID(r.NoteID)
		if note != nil {
			modelsToUpdate[note.ModelID] = true
		}
	}

	for modelID := range modelsToUpdate {
		if err := pkg.AddHMMFieldsToModel(modelID); err != nil {
			return fmt.Errorf("adding HMM fields to model: %w", err)
		}
	}

	// Update each note with HMM data
	for _, r := range results {
		note := pkg.GetNoteByID(r.NoteID)
		if note == nil {
			continue
		}

		// Combine HMM data for all characters in the note
		var actors, sets, toneRooms, props []string
		for _, h := range r.HMM {
			if h.ActorName != "" {
				actors = append(actors, h.ActorName)
			}
			if h.SetName != "" {
				sets = append(sets, h.SetName)
			}
			if h.ToneRoom != "" {
				toneRooms = append(toneRooms, h.ToneRoom)
			}
			props = append(props, h.Props...)
		}

		data := anki.AugmentedData{
			Actor:       strings.Join(unique(actors), ", "),
			Set:         strings.Join(unique(sets), ", "),
			ToneRoom:    strings.Join(unique(toneRooms), ", "),
			Props:       strings.Join(unique(props), ", "),
			ImagePrompt: r.Prompt,
		}

		if err := pkg.SetNoteHMMData(note, data); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not set HMM data for note %d: %v\n", note.ID, err)
		}
	}

	// Save the augmented package
	if err := pkg.SaveAs(outputPath); err != nil {
		return fmt.Errorf("saving augmented package: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Processed %d notes with Chinese characters\n", len(results))
	fmt.Fprintf(os.Stderr, "Wrote augmented deck to: %s\n", outputPath)
	fmt.Fprintf(os.Stderr, "\nNew fields added to notes:\n")
	for _, field := range anki.HMMFields {
		fmt.Fprintf(os.Stderr, "  - %s\n", field)
	}

	return nil
}

// unique removes duplicates from a string slice.
func unique(s []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, v := range s {
		if v != "" && !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// detectChineseField tries to find a field containing Chinese characters.
func detectChineseField(pkg *anki.Package) string {
	// Check first few notes
	for i, note := range pkg.Notes {
		if i >= 10 {
			break
		}
		fieldNames := pkg.GetFieldNames(note)
		for j, value := range note.Fields {
			if containsChinese(value) {
				if j < len(fieldNames) {
					return fieldNames[j]
				}
				return fmt.Sprintf("field_%d", j)
			}
		}
	}
	return ""
}

// containsChinese checks if a string contains Chinese characters.
func containsChinese(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// extractChineseChars extracts all Chinese characters from a string.
func extractChineseChars(s string) []string {
	var chars []string
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			chars = append(chars, string(r))
		}
	}
	return chars
}

// stripHTML removes HTML tags from a string.
func stripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}
