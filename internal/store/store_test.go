package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSeed(t *testing.T) {
	ctx := context.Background()
	s, err := Open(ctx, filepath.Join(t.TempDir(), "portfolio.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Portfolio(ctx)
	if err != nil {
		t.Fatal(err)
	}

	var want Portfolio
	if err := json.Unmarshal(seedJSON, &want); err != nil {
		t.Fatal(err)
	}
	// The seed has no IDs; everything else should come back unchanged and in order.
	for i := range got.Skills {
		got.Skills[i].ID = 0
	}
	for i := range got.Projects {
		got.Projects[i].ID = 0
	}
	for i := range got.Experience {
		got.Experience[i].ID = 0
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("portfolio doesn't match seed.json\n got: %+v\nwant: %+v", got, want)
	}
}

func TestSeedUploads(t *testing.T) {
	dir := t.TempDir()
	if err := CopySeedUploads(dir); err != nil {
		t.Fatal(err)
	}
	var p Portfolio
	if err := json.Unmarshal(seedJSON, &p); err != nil {
		t.Fatal(err)
	}
	for _, project := range p.Projects {
		for _, src := range append([]string{project.Image}, project.Gallery...) {
			name, ok := strings.CutPrefix(src, "/uploads/")
			if !ok {
				continue
			}
			if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
				t.Errorf("%s: %s is not in seed-uploads", project.Name, src)
			}
		}
	}

	// A file already there is kept, not overwritten.
	path := filepath.Join(dir, "pos-cover.webp")
	if err := os.WriteFile(path, []byte("replaced"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopySeedUploads(dir); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); string(data) != "replaced" {
		t.Error("CopySeedUploads overwrote an existing file")
	}
}

// A database made before projects had details and a gallery gets the new columns.
func TestAddColumns(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "portfolio.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE projects (
			id INTEGER PRIMARY KEY, position INTEGER NOT NULL, name TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT '', language TEXT NOT NULL DEFAULT '', description TEXT NOT NULL DEFAULT '',
			stack TEXT NOT NULL DEFAULT '[]', image TEXT NOT NULL DEFAULT '', url TEXT NOT NULL DEFAULT ''
		);
		INSERT INTO projects (position, name) VALUES (1, 'old-project')`); err != nil {
		t.Fatal(err)
	}
	db.Close()

	s, err := Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.Portfolio(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// The seed's projects follow, since the old database had no profile yet.
	if old := got.Projects[0]; old.Name != "old-project" || old.Gallery == nil {
		t.Fatalf("first project = %+v, want old-project with an empty gallery", old)
	}
}
