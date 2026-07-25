package main

import (
	"os"
	"testing"

	"github.com/ShinteLab/prokishi/db"
)

// chdirTemp creates a fresh temporary directory, changes the process's
// working directory into it, and restores the original working directory
// once the test finishes.
//
// ServerService.configPath (in dev mode) and db.Init/db.Open (also in dev
// mode) both resolve their storage location via prokishi.GetRunDir(true),
// which is simply os.Getwd(). Chdir'ing into a fresh temp dir is therefore
// the only way to sandbox both per test.
func chdirTemp(t *testing.T) string {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir() error: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("os.Chdir(restore) error: %v", err)
		}
	})

	return dir
}

// setupTestDB sandboxes a fresh CSVQ-backed DB (schema created via db.Init,
// connection opened via db.Open) in a temp dir and returns that dir's path.
//
// This mirrors the root module's db/testhelper_test.go pattern. It can't be
// imported directly (unexported helpers in a _test.go file, plus this is a
// separate module/package), so it's replicated here against the identical
// db package API (resolved via the "replace prokishi => ../../" directive
// in this module's go.mod).
//
// Like the root package's tests, tests using this helper must not run with
// t.Parallel() - db's gDB is an unexported package-level *sql.DB shared by
// every test in this binary.
func setupTestDB(t *testing.T) string {
	t.Helper()

	dir := chdirTemp(t)

	// Registered after chdirTemp's own cleanup, so ours runs first
	// (t.Cleanup runs LIFO): close the DB handle while still inside the
	// temp dir, before chdirTemp's cleanup changes back out of it.
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := db.Init(true); err != nil {
		t.Fatalf("db.Init() error: %v", err)
	}
	if err := db.Open(true); err != nil {
		t.Fatalf("db.Open() error: %v", err)
	}

	return dir
}

// createPlaceholderFile creates an empty, content-irrelevant file at path
// (db.InsertEngine only checks the path exists via os.Stat) and closes the
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
