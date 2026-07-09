package db

import (
	"os"
	"testing"
)

// chdirTemp creates a fresh temporary directory, changes the process's
// working directory into it, and arranges for the working directory (and
// any open DB handle) to be restored/closed once the test finishes.
//
// db.Init/db.Open in dev mode (dev=true) resolve their storage location via
// prokishi.GetRunDir(true), which is simply os.Getwd(). Chdir'ing into a
// fresh temp dir is therefore the only way to sandbox the CSVQ-backed DB
// per test.
//
// gDB is an unexported package-level *sql.DB, so tests in this package must
// not run with t.Parallel() - they'd stomp on each other's global state and
// working directory. Go still runs other packages' tests concurrently in
// separate processes, so this only serializes tests within package db.
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

	// Registered after t.TempDir()'s own cleanup, so ours runs first
	// (cleanups run LIFO): close the DB handle and cd back out *before*
	// TempDir tries to remove the directory - on Windows you cannot remove
	// a directory that is still the process's current working directory.
	t.Cleanup(func() {
		_ = Close()
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("os.Chdir(restore) error: %v", err)
		}
	})

	return dir
}

// setupTestDB sandboxes a fresh CSVQ-backed DB (schema created via Init,
// connection opened via Open) in a temp dir and returns that dir's path.
func setupTestDB(t *testing.T) string {
	t.Helper()

	dir := chdirTemp(t)

	if err := Init(true); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if err := Open(true); err != nil {
		t.Fatalf("Open() error: %v", err)
	}

	return dir
}
