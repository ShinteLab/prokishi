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
	// migrateCodesColumns runs migrateCSVColumn twice in sequence (once for
	// "disabled", once for "name"). Each call now pads every data row out to
	// the header's column count instead of stripping-then-appending, so a
	// row migrated across both calls ends up with one empty field per
	// missing column - matching the header exactly.
	wantRow := "OLDCODE1,2024-01-01 00:00:00,2024-01-01 00:00:00,,"
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

	// The migrated file must actually be readable through the normal DB
	// API - this pins the fix for the field-count mismatch that used to
	// make CSVQ reject every row in a codes.csv predating both the
	// "disabled" and "name" columns.
	if err := Open(true); err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	got, err := SelectCode(context.Background(), "OLDCODE1")
	if err != nil {
		t.Fatalf("SelectCode() error after migration: %v", err)
	}
	if got == nil || got.Code != "OLDCODE1" {
		t.Fatalf("SelectCode() = %+v, want code OLDCODE1", got)
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

func TestMigrateCSVColumn_AddsNewColumn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	os.WriteFile(path, []byte("id,name\n1,alice\n2,bob\n"), 0644)

	migrateCSVColumn(path, "age")

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")

	if lines[0] != "id,name,age" {
		t.Errorf("header = %q, want %q", lines[0], "id,name,age")
	}
	if lines[1] != "1,alice," {
		t.Errorf("row1 = %q, want %q", lines[1], "1,alice,")
	}
	if lines[2] != "2,bob," {
		t.Errorf("row2 = %q, want %q", lines[2], "2,bob,")
	}
}

func TestMigrateCSVColumn_ExistingColumnNoOp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	original := "id,name\n1,alice\n"
	os.WriteFile(path, []byte(original), 0644)

	migrateCSVColumn(path, "name")

	data, _ := os.ReadFile(path)
	if string(data) != original {
		t.Errorf("file changed: got %q, want %q", string(data), original)
	}
}

func TestMigrateCSVColumn_SequentialMigrations(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	os.WriteFile(path, []byte("code,created_date\nabc,2024-01-01\n"), 0644)

	migrateCSVColumn(path, "disabled")
	migrateCSVColumn(path, "name")

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")

	if lines[0] != "code,created_date,disabled,name" {
		t.Errorf("header = %q, want %q", lines[0], "code,created_date,disabled,name")
	}
	if lines[1] != "abc,2024-01-01,," {
		t.Errorf("row = %q, want %q", lines[1], "abc,2024-01-01,,")
	}
}

func TestMigrateCSVColumn_CRLFLineEndings(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	os.WriteFile(path, []byte("id,name\r\n1,alice\r\n"), 0644)

	migrateCSVColumn(path, "age")

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")

	if lines[0] != "id,name,age" {
		t.Errorf("header = %q, want %q", lines[0], "id,name,age")
	}
}

func TestMigrateCSVColumn_EmptyDataRows(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	os.WriteFile(path, []byte("id,name\n"), 0644)

	migrateCSVColumn(path, "age")

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")

	if lines[0] != "id,name,age" {
		t.Errorf("header = %q, want %q", lines[0], "id,name,age")
	}
	if len(lines) != 1 {
		t.Errorf("expected 1 line (header only), got %d", len(lines))
	}
}

func TestCreateTableFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "engines.csv")

	err := createTableFile(path, EnginesColumns)
	if err != nil {
		t.Fatalf("createTableFile() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	if string(data) != EnginesColumns {
		t.Errorf("content = %q, want %q", string(data), EnginesColumns)
	}
}

func TestInitTables(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "db")

	err := initTables(dbDir)
	if err != nil {
		t.Fatalf("initTables() error: %v", err)
	}

	codesPath := filepath.Join(dbDir, "codes.csv")
	if _, err := os.Stat(codesPath); err != nil {
		t.Errorf("codes.csv not created: %v", err)
	}

	enginesPath := filepath.Join(dbDir, "engines.csv")
	if _, err := os.Stat(enginesPath); err != nil {
		t.Errorf("engines.csv not created: %v", err)
	}

	// 2回目はマイグレーションパス
	err = initTables(dbDir)
	if err != nil {
		t.Fatalf("initTables() second call error: %v", err)
	}
}
