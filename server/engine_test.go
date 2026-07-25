package server

import (
	"context"
	"testing"

	"github.com/ShinteLab/prokishi/db"
	"github.com/ShinteLab/prokishi/internal/testfakeengine"
)

// These tests confirm GenerateEngineId/RegisterEngineId/DeleteEngineId
// correctly delegate to the db package; db's own logic is already covered
// in depth by db's test suite, so these stay shallow.
//
// db.InsertEngine requires the path to actually exist (os.Stat), so these
// use a real (built) fake-engine binary as the path rather than an
// arbitrary string.

func TestGenerateEngineId(t *testing.T) {
	setupTestDB(t)

	binPath := testfakeengine.Build(t)

	if err := GenerateEngineId(binPath); err != nil {
		t.Fatalf("GenerateEngineId() error: %v", err)
	}

	engines, err := db.FindEngines(context.Background())
	if err != nil {
		t.Fatalf("db.FindEngines() error: %v", err)
	}
	if len(engines) != 1 {
		t.Fatalf("len(engines) = %d, want 1", len(engines))
	}
	if engines[0].ID == "" {
		t.Fatalf("GenerateEngineId() produced an empty ID")
	}
	if engines[0].Path != binPath {
		t.Fatalf("engines[0].Path = %q, want %q", engines[0].Path, binPath)
	}
}

func TestRegisterEngineId(t *testing.T) {
	setupTestDB(t)

	binPath := testfakeengine.Build(t)
	const id = "my-engine-id"

	if err := RegisterEngineId(id, binPath); err != nil {
		t.Fatalf("RegisterEngineId() error: %v", err)
	}

	got, err := db.SelectEngine(context.Background(), id)
	if err != nil {
		t.Fatalf("db.SelectEngine() error: %v", err)
	}
	if got == nil {
		t.Fatalf("RegisterEngineId(%q) did not persist the engine", id)
	}
	if got.Path != binPath {
		t.Fatalf("got.Path = %q, want %q", got.Path, binPath)
	}
}

func TestDeleteEngineId(t *testing.T) {
	setupTestDB(t)

	binPath := testfakeengine.Build(t)
	const id = "to-delete"

	if err := RegisterEngineId(id, binPath); err != nil {
		t.Fatalf("RegisterEngineId() error: %v", err)
	}

	if err := DeleteEngineId(id); err != nil {
		t.Fatalf("DeleteEngineId() error: %v", err)
	}

	got, err := db.SelectEngine(context.Background(), id)
	if err != nil {
		t.Fatalf("db.SelectEngine() error: %v", err)
	}
	if got != nil {
		t.Fatalf("DeleteEngineId(%q) did not remove the engine", id)
	}
}
