package anki

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// HMMFields are the fields we add to augmented notes.
var HMMFields = []string{
	"HMM_Actor",
	"HMM_Set",
	"HMM_ToneRoom",
	"HMM_Props",
	"HMM_ImagePrompt",
}

// AugmentedData holds HMM data for a note.
type AugmentedData struct {
	Actor       string
	Set         string
	ToneRoom    string
	Props       string
	ImagePrompt string
}

// AddHMMFieldsToModel adds HMM fields to a model if they don't exist.
func (p *Package) AddHMMFieldsToModel(modelID int64) error {
	model, ok := p.Models[modelID]
	if !ok {
		return fmt.Errorf("model %d not found", modelID)
	}

	// Check which fields already exist
	existingFields := make(map[string]bool)
	for _, f := range model.Fields {
		existingFields[f.Name] = true
	}

	// Add missing HMM fields
	nextOrd := len(model.Fields)
	for _, fieldName := range HMMFields {
		if !existingFields[fieldName] {
			model.Fields = append(model.Fields, Field{
				Name:   fieldName,
				Ord:    nextOrd,
				Sticky: false,
				RTL:    false,
				Font:   "Arial",
				Size:   20,
			})
			nextOrd++
		}
	}

	return nil
}

// SetNoteHMMData sets the HMM fields for a note.
func (p *Package) SetNoteHMMData(note *Note, data AugmentedData) error {
	model := p.GetModel(note)
	if model == nil {
		return fmt.Errorf("model not found for note %d", note.ID)
	}

	// Build a map of field name to index
	fieldIndex := make(map[string]int)
	for _, f := range model.Fields {
		fieldIndex[f.Name] = f.Ord
	}

	// Ensure note.Fields has enough slots
	for len(note.Fields) < len(model.Fields) {
		note.Fields = append(note.Fields, "")
	}

	// Set the HMM field values
	if idx, ok := fieldIndex["HMM_Actor"]; ok {
		note.Fields[idx] = data.Actor
	}
	if idx, ok := fieldIndex["HMM_Set"]; ok {
		note.Fields[idx] = data.Set
	}
	if idx, ok := fieldIndex["HMM_ToneRoom"]; ok {
		note.Fields[idx] = data.ToneRoom
	}
	if idx, ok := fieldIndex["HMM_Props"]; ok {
		note.Fields[idx] = data.Props
	}
	if idx, ok := fieldIndex["HMM_ImagePrompt"]; ok {
		note.Fields[idx] = data.ImagePrompt
	}

	// Update RawFlds
	note.RawFlds = strings.Join(note.Fields, "\x1f")

	// Update modification time
	note.Mod = time.Now().Unix()

	return nil
}

// SaveAs writes the modified package to a new .apkg file.
func (p *Package) SaveAs(outputPath string) error {
	// Update the database first
	if err := p.updateDatabase(); err != nil {
		return fmt.Errorf("updating database: %w", err)
	}

	// Create the output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer outFile.Close()

	// Create zip writer
	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// Walk the temp directory and add all files to the zip
	err = filepath.Walk(p.tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(p.tempDir, path)
		if err != nil {
			return err
		}

		// Create zip entry
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// Copy file contents
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		return fmt.Errorf("creating zip: %w", err)
	}

	return nil
}

// updateDatabase writes changes back to the SQLite database.
func (p *Package) updateDatabase() error {
	// Update models in col table
	if err := p.updateModels(); err != nil {
		return err
	}

	// Update notes
	if err := p.updateNotes(); err != nil {
		return err
	}

	return nil
}

// updateModels updates the models JSON in the col table.
func (p *Package) updateModels() error {
	// Build models map
	modelsMap := make(map[string]interface{})
	for id, model := range p.Models {
		// Convert model to a map to preserve all original fields
		modelMap := map[string]interface{}{
			"id":   model.ID,
			"name": model.Name,
			"flds": model.Fields,
			"css":  model.CSS,
			"type": model.Type,
		}
		modelsMap[strconv.FormatInt(id, 10)] = modelMap
	}

	modelsJSON, err := json.Marshal(modelsMap)
	if err != nil {
		return fmt.Errorf("marshaling models: %w", err)
	}

	_, err = p.db.Exec("UPDATE col SET models = ?", string(modelsJSON))
	if err != nil {
		return fmt.Errorf("updating models: %w", err)
	}

	return nil
}

// updateNotes updates all modified notes in the database.
func (p *Package) updateNotes() error {
	for _, note := range p.Notes {
		// Calculate new checksum (first 8 digits of SHA256 of sort field)
		h := sha256.New()
		h.Write([]byte(note.SFLD))
		hashStr := fmt.Sprintf("%x", h.Sum(nil))
		if len(hashStr) >= 8 {
			csum, _ := strconv.ParseInt(hashStr[:8], 16, 64)
			note.CSum = csum
		}

		_, err := p.db.Exec(`
			UPDATE notes SET
				mod = ?,
				flds = ?,
				sfld = ?,
				csum = ?
			WHERE id = ?
		`, note.Mod, note.RawFlds, note.SFLD, note.CSum, note.ID)

		if err != nil {
			return fmt.Errorf("updating note %d: %w", note.ID, err)
		}
	}

	return nil
}
