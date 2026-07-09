package db

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createPlaceholderFile creates an empty, content-irrelevant file at path
// (InsertEngine only checks the path exists via os.Stat) and closes the
// handle immediately - on Windows an open handle blocks the temp dir's
// cleanup from deleting it.
func createPlaceholderFile(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("os.Create(%s) error: %v", path, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("f.Close(%s) error: %v", path, err)
	}
}

func TestInsertEngine_NonexistentPathFails(t *testing.T) {
	dir := setupTestDB(t)
	ctx := context.Background()

	path := filepath.Join(dir, "does-not-exist.exe")
	if err := InsertEngine("ENGINE1", path, "phantom"); err == nil {
		t.Fatal("InsertEngine() with a nonexistent path succeeded, want error")
	}

	e, err := SelectEngine(ctx, "ENGINE1")
	if err != nil {
		t.Fatalf("SelectEngine() error: %v", err)
	}
	if e != nil {
		t.Fatalf("SelectEngine() = %+v, want nil (insert should not have happened)", e)
	}
}

func TestInsertEngine_SelectEngine_Roundtrip(t *testing.T) {
	dir := setupTestDB(t)
	ctx := context.Background()

	enginePath := filepath.Join(dir, "engine.exe")
	createPlaceholderFile(t, enginePath)

	if err := InsertEngine("ENGINE1", enginePath, "my engine"); err != nil {
		t.Fatalf("InsertEngine() error: %v", err)
	}

	e, err := SelectEngine(ctx, "ENGINE1")
	if err != nil {
		t.Fatalf("SelectEngine() error: %v", err)
	}
	if e == nil {
		t.Fatal("SelectEngine() returned nil for an inserted engine")
	}
	if e.ID != "ENGINE1" {
		t.Errorf("ID = %q, want %q", e.ID, "ENGINE1")
	}
	if e.Path != enginePath {
		t.Errorf("Path = %q, want %q", e.Path, enginePath)
	}
	if e.Name != "my engine" {
		t.Errorf("Name = %q, want %q", e.Name, "my engine")
	}
	if d := time.Since(e.Created); d < -time.Minute || d > time.Minute {
		t.Errorf("Created = %v, expected to be close to now", e.Created)
	}
}

func TestSelectEngine_NotFound(t *testing.T) {
	setupTestDB(t)
	ctx := context.Background()

	e, err := SelectEngine(ctx, "NOPE")
	if err != nil {
		t.Fatalf("SelectEngine() error: %v", err)
	}
	if e != nil {
		t.Fatalf("SelectEngine() = %+v, want nil for an unknown id", e)
	}
}

func TestUpdateEngineName(t *testing.T) {
	dir := setupTestDB(t)
	ctx := context.Background()

	enginePath := filepath.Join(dir, "engine.exe")
	createPlaceholderFile(t, enginePath)
	if err := InsertEngine("ENGINE1", enginePath, "old name"); err != nil {
		t.Fatalf("InsertEngine() error: %v", err)
	}

	if err := UpdateEngineName("ENGINE1", "new name"); err != nil {
		t.Fatalf("UpdateEngineName() error: %v", err)
	}

	e, err := SelectEngine(ctx, "ENGINE1")
	if err != nil {
		t.Fatalf("SelectEngine() error: %v", err)
	}
	if e == nil {
		t.Fatal("SelectEngine() returned nil")
	}
	if e.Name != "new name" {
		t.Errorf("Name = %q, want %q", e.Name, "new name")
	}
}

func TestDeleteEngine(t *testing.T) {
	dir := setupTestDB(t)
	ctx := context.Background()

	enginePath := filepath.Join(dir, "engine.exe")
	createPlaceholderFile(t, enginePath)
	if err := InsertEngine("ENGINE1", enginePath, "a"); err != nil {
		t.Fatalf("InsertEngine() error: %v", err)
	}

	if err := DeleteEngine("ENGINE1"); err != nil {
		t.Fatalf("DeleteEngine() error: %v", err)
	}

	e, err := SelectEngine(ctx, "ENGINE1")
	if err != nil {
		t.Fatalf("SelectEngine() error: %v", err)
	}
	if e != nil {
		t.Fatalf("SelectEngine() = %+v, want nil after DeleteEngine()", e)
	}
}

func TestFindEngines_OrdersByUpdatedDateDesc(t *testing.T) {
	dir := setupTestDB(t)
	ctx := context.Background()

	path1 := filepath.Join(dir, "engine1.exe")
	path2 := filepath.Join(dir, "engine2.exe")
	createPlaceholderFile(t, path1)
	createPlaceholderFile(t, path2)

	if err := InsertEngine("ENGINE1", path1, "first"); err != nil {
		t.Fatalf("InsertEngine() error: %v", err)
	}
	// InsertEngine timestamps with time.Now(); sleep to force a real gap
	// (DATETIME() resolution is unknown) so ordering is deterministic.
	time.Sleep(1100 * time.Millisecond)
	if err := InsertEngine("ENGINE2", path2, "second"); err != nil {
		t.Fatalf("InsertEngine() error: %v", err)
	}

	engines, err := FindEngines(ctx)
	if err != nil {
		t.Fatalf("FindEngines() error: %v", err)
	}
	if len(engines) != 2 {
		t.Fatalf("FindEngines() returned %d engines, want 2", len(engines))
	}
	if engines[0].ID != "ENGINE2" || engines[1].ID != "ENGINE1" {
		t.Fatalf("FindEngines() order = [%s, %s], want [ENGINE2, ENGINE1] (most recently updated first)", engines[0].ID, engines[1].ID)
	}
}
