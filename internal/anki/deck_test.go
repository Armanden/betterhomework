package anki

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestWriteAPKG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.apkg")
	deck := &Deck{
		Name: "test_deck",
		Cards: []Card{
			{Front: "What is the capital of France?", Back: "Paris"},
			{Front: "What is 2+2?", Back: "4"},
		},
	}

	if err := deck.WriteAPKG(path); err != nil {
		t.Fatalf("WriteAPKG: %v", err)
	}

	// Open as a zip and check it contains collection.anki2.
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open apkg: %v", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(f, stat.Size())
	if err != nil {
		t.Fatalf("apkg is not a valid zip archive: %v", err)
	}

	var entry *zip.File
	for _, z := range zr.File {
		if z.Name == "collection.anki2" {
			entry = z
			break
		}
	}
	if entry == nil {
		t.Fatal("collection.anki2 not found in apkg")
	}

	// Extract the db to a temp file and open with sqlite.
	dbFile := filepath.Join(dir, "collection.anki2")
	rc, err := entry.Open()
	if err != nil {
		t.Fatal(err)
	}
	content, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dbFile, content, 0o600); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM notes`).Scan(&n); err != nil {
		t.Fatalf("query notes: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 notes, got %d", n)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM cards`).Scan(&n); err != nil {
		t.Fatalf("query cards: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 cards, got %d", n)
	}

	// Anki's importer parses the dconf/decks/models JSON blobs on import.
	// Verify they are valid JSON so import won't fail with a JsonError.
	var models, decks, dconf, conf string
	if err := db.QueryRow(`SELECT models, decks, dconf, conf FROM col`).Scan(&models, &decks, &dconf, &conf); err != nil {
		t.Fatalf("query col: %v", err)
	}
	for name, blob := range map[string]string{"models": models, "decks": decks, "dconf": dconf, "conf": conf} {
		var m map[string]json.RawMessage
		if err := json.Unmarshal([]byte(blob), &m); err != nil {
			t.Fatalf("col.%s is not valid JSON: %v\n%s", name, err, blob)
		}
	}
}

func TestWriteAPKGEmptyDeck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.apkg")
	deck := &Deck{Name: "empty"}
	if err := deck.WriteAPKG(path); err != nil {
		t.Fatalf("WriteAPKG empty deck: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected apkg file to exist: %v", err)
	}
}
