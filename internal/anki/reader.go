// Package anki handles reading and writing Anki .apkg files.
package anki

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Package represents an Anki .apkg file.
type Package struct {
	path    string
	tempDir string
	db      *sql.DB
	Models  map[int64]*Model
	Decks   map[int64]*Deck
	Notes   []*Note
	Cards   []*Card
}

// Model represents an Anki note type (model).
type Model struct {
	ID     int64    `json:"id"`
	Name   string   `json:"name"`
	Fields []Field  `json:"flds"`
	CSS    string   `json:"css"`
	Type   int      `json:"type"` // 0 = standard, 1 = cloze
}

// Field represents a field in a note type.
type Field struct {
	Name     string `json:"name"`
	Ord      int    `json:"ord"`
	Sticky   bool   `json:"sticky"`
	RTL      bool   `json:"rtl"`
	Font     string `json:"font"`
	Size     int    `json:"size"`
}

// Deck represents an Anki deck.
type Deck struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// Note represents an Anki note.
type Note struct {
	ID      int64
	GUID    string
	ModelID int64
	Mod     int64
	USN     int
	Tags    string
	Fields  []string // Parsed from flds
	RawFlds string   // Original flds string
	SFLD    string   // Sort field
	CSum    int64
	Flags   int
	Data    string
}

// Card represents an Anki card.
type Card struct {
	ID     int64
	NoteID int64
	DeckID int64
	Ord    int
	Mod    int64
	USN    int
	Type   int
	Queue  int
	Due    int
	IVL    int
	Factor int
	Reps   int
	Lapses int
	Left   int
	ODue   int
	ODid   int64
	Flags  int
	Data   string
}

// OpenPackage opens an Anki .apkg file for reading.
func OpenPackage(path string) (*Package, error) {
	pkg := &Package{
		path:   path,
		Models: make(map[int64]*Model),
		Decks:  make(map[int64]*Deck),
	}

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "anki-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	pkg.tempDir = tempDir

	// Extract .apkg (it's a zip file)
	if err := pkg.extract(); err != nil {
		pkg.Close()
		return nil, err
	}

	// Open the SQLite database
	dbPath := filepath.Join(tempDir, "collection.anki2")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Try anki21 format
		dbPath = filepath.Join(tempDir, "collection.anki21")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		pkg.Close()
		return nil, fmt.Errorf("opening database: %w", err)
	}
	pkg.db = db

	// Load collection metadata
	if err := pkg.loadCollection(); err != nil {
		pkg.Close()
		return nil, err
	}

	// Load notes
	if err := pkg.loadNotes(); err != nil {
		pkg.Close()
		return nil, err
	}

	// Load cards
	if err := pkg.loadCards(); err != nil {
		pkg.Close()
		return nil, err
	}

	return pkg, nil
}

// extract unzips the .apkg file.
func (p *Package) extract() error {
	r, err := zip.OpenReader(p.path)
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(p.tempDir, f.Name)

		// Prevent zip slip
		if !strings.HasPrefix(fpath, filepath.Clean(p.tempDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

// loadCollection loads models and decks from the col table.
func (p *Package) loadCollection() error {
	var models, decks string

	row := p.db.QueryRow("SELECT models, decks FROM col")
	if err := row.Scan(&models, &decks); err != nil {
		return fmt.Errorf("reading collection: %w", err)
	}

	// Parse models
	var modelsMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(models), &modelsMap); err != nil {
		return fmt.Errorf("parsing models: %w", err)
	}

	for _, modelJSON := range modelsMap {
		var model Model
		if err := json.Unmarshal(modelJSON, &model); err != nil {
			continue // Skip malformed models
		}
		p.Models[model.ID] = &model
	}

	// Parse decks
	var decksMap map[string]json.RawMessage
	if err := json.Unmarshal([]byte(decks), &decksMap); err != nil {
		return fmt.Errorf("parsing decks: %w", err)
	}

	for _, deckJSON := range decksMap {
		var deck Deck
		if err := json.Unmarshal(deckJSON, &deck); err != nil {
			continue // Skip malformed decks
		}
		p.Decks[deck.ID] = &deck
	}

	return nil
}

// loadNotes loads all notes from the database.
func (p *Package) loadNotes() error {
	rows, err := p.db.Query(`
		SELECT id, guid, mid, mod, usn, tags, flds, sfld, csum, flags, data
		FROM notes
	`)
	if err != nil {
		return fmt.Errorf("querying notes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var note Note
		if err := rows.Scan(
			&note.ID, &note.GUID, &note.ModelID, &note.Mod, &note.USN,
			&note.Tags, &note.RawFlds, &note.SFLD, &note.CSum, &note.Flags, &note.Data,
		); err != nil {
			return fmt.Errorf("scanning note: %w", err)
		}

		// Parse fields (separated by ASCII 31)
		note.Fields = strings.Split(note.RawFlds, "\x1f")
		p.Notes = append(p.Notes, &note)
	}

	return rows.Err()
}

// loadCards loads all cards from the database.
func (p *Package) loadCards() error {
	rows, err := p.db.Query(`
		SELECT id, nid, did, ord, mod, usn, type, queue, due, ivl, factor, reps, lapses, left, odue, odid, flags, data
		FROM cards
	`)
	if err != nil {
		return fmt.Errorf("querying cards: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var card Card
		if err := rows.Scan(
			&card.ID, &card.NoteID, &card.DeckID, &card.Ord, &card.Mod, &card.USN,
			&card.Type, &card.Queue, &card.Due, &card.IVL, &card.Factor, &card.Reps,
			&card.Lapses, &card.Left, &card.ODue, &card.ODid, &card.Flags, &card.Data,
		); err != nil {
			return fmt.Errorf("scanning card: %w", err)
		}
		p.Cards = append(p.Cards, &card)
	}

	return rows.Err()
}

// GetModel returns the model for a note.
func (p *Package) GetModel(note *Note) *Model {
	return p.Models[note.ModelID]
}

// GetDeck returns the deck for a card.
func (p *Package) GetDeck(card *Card) *Deck {
	return p.Decks[card.DeckID]
}

// GetNoteByID finds a note by ID.
func (p *Package) GetNoteByID(id int64) *Note {
	for _, note := range p.Notes {
		if note.ID == id {
			return note
		}
	}
	return nil
}

// GetFieldValue returns a specific field value from a note by field name.
func (p *Package) GetFieldValue(note *Note, fieldName string) string {
	model := p.GetModel(note)
	if model == nil {
		return ""
	}

	for _, field := range model.Fields {
		if strings.EqualFold(field.Name, fieldName) && field.Ord < len(note.Fields) {
			return note.Fields[field.Ord]
		}
	}

	return ""
}

// GetFieldNames returns all field names for a note's model.
func (p *Package) GetFieldNames(note *Note) []string {
	model := p.GetModel(note)
	if model == nil {
		return nil
	}

	names := make([]string, len(model.Fields))
	for i, field := range model.Fields {
		names[i] = field.Name
	}
	return names
}

// Close cleans up resources.
func (p *Package) Close() error {
	if p.db != nil {
		p.db.Close()
	}
	if p.tempDir != "" {
		os.RemoveAll(p.tempDir)
	}
	return nil
}

// Summary returns a summary of the package contents.
func (p *Package) Summary() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Anki Package: %s\n", p.path))
	sb.WriteString(fmt.Sprintf("  Decks: %d\n", len(p.Decks)))
	for _, deck := range p.Decks {
		sb.WriteString(fmt.Sprintf("    - %s\n", deck.Name))
	}
	sb.WriteString(fmt.Sprintf("  Models (Note Types): %d\n", len(p.Models)))
	for _, model := range p.Models {
		sb.WriteString(fmt.Sprintf("    - %s (%d fields)\n", model.Name, len(model.Fields)))
	}
	sb.WriteString(fmt.Sprintf("  Notes: %d\n", len(p.Notes)))
	sb.WriteString(fmt.Sprintf("  Cards: %d\n", len(p.Cards)))

	return sb.String()
}
