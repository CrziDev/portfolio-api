package store

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
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
