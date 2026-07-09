package db

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error: %v", path, err)
	}
	return string(data)
}

func TestInit_CreatesSchemaFiles(t *testing.T) {
	dir := chdirTemp(t)

	if err := Init(true); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	dbDir := filepath.Join(dir, "db")
	if info, err := os.Stat(dbDir); err != nil || !info.IsDir() {
		t.Fatalf("expected db dir to be created at %s: %v", dbDir, err)
	}

	// createTableFile writes exactly the header string, with no trailing
	// newline and no data rows.
	codesContent := readFile(t, filepath.Join(dbDir, "codes.csv"))
	if codesContent != CodesColumns {
		t.Fatalf("codes.csv content = %q, want %q", codesContent, CodesColumns)
	}

	enginesContent := readFile(t, filepath.Join(dbDir, "engines.csv"))
	if enginesContent != EnginesColumns {
		t.Fatalf("engines.csv content = %q, want %q", enginesContent, EnginesColumns)
	}
}

func TestInit_IdempotentOnFreshSchema(t *testing.T) {
	dir := chdirTemp(t)
	dbDir := filepath.Join(dir, "db")

	if err := Init(true); err != nil {
		t.Fatalf("first Init() error: %v", err)
	}
	beforeCodes := readFile(t, filepath.Join(dbDir, "codes.csv"))
	beforeEngines := readFile(t, filepath.Join(dbDir, "engines.csv"))

	if err := Init(true); err != nil {
		t.Fatalf("second Init() error: %v", err)
	}
	afterCodes := readFile(t, filepath.Join(dbDir, "codes.csv"))
	afterEngines := readFile(t, filepath.Join(dbDir, "engines.csv"))

	if beforeCodes != afterCodes {
		t.Fatalf("codes.csv changed on second Init(): before=%q after=%q", beforeCodes, afterCodes)
	}
	if beforeEngines != afterEngines {
		t.Fatalf("engines.csv changed on second Init(): before=%q after=%q", beforeEngines, afterEngines)
	}
}

func TestInit_MigratesOldCodesSchema(t *testing.T) {
	dir := chdirTemp(t)
	dbDir := filepath.Join(dir, "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll() error: %v", err)
	}

	// Pre-existing "old format" codes.csv, missing the disabled/name
	// columns current schema expects, using CRLF line endings.
	oldContent := "code,created_date,updated_date\r\n" +
		"OLDCODE1,2024-01-01 00:00:00,2024-01-01 00:00:00\r\n"
	codesPath := filepath.Join(dbDir, "codes.csv")
	if err := os.WriteFile(codesPath, []byte(oldContent), 0644); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	if err := Init(true); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	migrated := readFile(t, codesPath)
	if strings.Contains(migrated, "\r") {
		t.Fatalf("expected CRLF to be normalized to LF during migration, got %q", migrated)
	}

	lines := strings.Split(strings.TrimSuffix(migrated, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected header + 1 data row, got %d lines: %q", len(lines), migrated)
	}
	if lines[0] != CodesColumns {
		t.Fatalf("header = %q, want %q", lines[0], CodesColumns)
	}
	// NOTE: migrateCodesColumns runs migrateCSVColumn twice in sequence
	// (once for "disabled", once for "name"). Each call trims *any*
	// trailing empty field before appending its own, so the second call
	// strips the blank the first call just added instead of appending a
	// second one. The net effect: the header gains both columns (5
	// fields), but each data row only gains a single trailing empty field
	// (4 fields) - the row ends up one field short of the header. This
	// looks like an unintended quirk of the migration helper, but it is
	// what the current code actually does, so the test pins that behavior
	// rather than an idealized 5-field row.
	wantRow := "OLDCODE1,2024-01-01 00:00:00,2024-01-01 00:00:00,"
	if lines[1] != wantRow {
		t.Fatalf("data row = %q, want %q", lines[1], wantRow)
	}

	// A second Init() must be a no-op: the header already has every
	// expected column, so migrateCSVColumn short-circuits.
	if err := Init(true); err != nil {
		t.Fatalf("second Init() error: %v", err)
	}
	again := readFile(t, codesPath)
	if again != migrated {
		t.Fatalf("second Init() changed an already-migrated file: before=%q after=%q", migrated, again)
	}

	// BUG: because of the field-count mismatch documented above, the
	// migrated file is actually *not* readable through the normal DB API -
	// CSVQ rejects the row outright since it has one fewer field than the
	// header declares. This means migrating a codes.csv that predates both
	// the "disabled" and "name" columns currently breaks every query
	// against the codes table, not just lookups of the migrated row. This
	// test pins that regression so it turns green (and should be updated
	// to assert a successful SelectCode) once migrateCSVColumn is fixed to
	// not swallow a previous call's newly-appended empty field.
	if err := Open(true); err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	_, err := SelectCode(context.Background(), "OLDCODE1")
	if err == nil {
		t.Fatal("SelectCode() unexpectedly succeeded after migration - if migrateCSVColumn was fixed, update this test to assert successful data instead")
	}
	if !strings.Contains(err.Error(), "wrong number of fields") {
		t.Fatalf("SelectCode() error = %v, want a CSVQ \"wrong number of fields\" parse error", err)
	}
}

func TestInit_MigratesOldEnginesSchema(t *testing.T) {
	dir := chdirTemp(t)
	dbDir := filepath.Join(dir, "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll() error: %v", err)
	}

	// Pre-existing "old format" engines.csv, missing the name column,
	// using CRLF line endings.
	oldContent := "id,path,created_date,updated_date\r\n" +
		"OLDENGINE1,C:\\engine.exe,2024-01-01 00:00:00,2024-01-01 00:00:00\r\n"
	enginesPath := filepath.Join(dbDir, "engines.csv")
	if err := os.WriteFile(enginesPath, []byte(oldContent), 0644); err != nil {
		t.Fatalf("os.WriteFile() error: %v", err)
	}

	if err := Init(true); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	migrated := readFile(t, enginesPath)
	if strings.Contains(migrated, "\r") {
		t.Fatalf("expected CRLF to be normalized to LF during migration, got %q", migrated)
	}

	lines := strings.Split(strings.TrimSuffix(migrated, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected header + 1 data row, got %d lines: %q", len(lines), migrated)
	}
	if lines[0] != EnginesColumns {
		t.Fatalf("header = %q, want %q", lines[0], EnginesColumns)
	}
	wantRow := "OLDENGINE1,C:\\engine.exe,2024-01-01 00:00:00,2024-01-01 00:00:00,"
	if lines[1] != wantRow {
		t.Fatalf("data row = %q, want %q", lines[1], wantRow)
	}

	if err := Init(true); err != nil {
		t.Fatalf("second Init() error: %v", err)
	}
	again := readFile(t, enginesPath)
	if again != migrated {
		t.Fatalf("second Init() changed an already-migrated file: before=%q after=%q", migrated, again)
	}
}
