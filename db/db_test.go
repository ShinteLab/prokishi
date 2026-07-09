package db

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
