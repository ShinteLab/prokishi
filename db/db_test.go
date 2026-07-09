package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mithrandie/csvq-driver"
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

// disabled・name 両方の列が存在しない旧スキーマの codes.csv が、
// initTables() の呼び出しで CSVQ が読み込める正しい形式に移行されることを検証する。
func TestInit_MigratesOldCodesSchema(t *testing.T) {
	dir := t.TempDir()
	dbDir := filepath.Join(dir, "db")
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}

	codesPath := filepath.Join(dbDir, "codes.csv")
	old := "code,created_date,updated_date\nabc123,2024-01-01 00:00:00,2024-01-01 00:00:00\n"
	if err := os.WriteFile(codesPath, []byte(old), 0644); err != nil {
		t.Fatalf("WriteFile() error: %v", err)
	}

	if err := initTables(dbDir); err != nil {
		t.Fatalf("initTables() error: %v", err)
	}

	data, err := os.ReadFile(codesPath)
	if err != nil {
		t.Fatalf("ReadFile() error: %v", err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")

	if lines[0] != CodesColumns {
		t.Fatalf("header = %q, want %q", lines[0], CodesColumns)
	}

	headerCols := len(strings.Split(lines[0], ","))
	for i := 1; i < len(lines); i++ {
		rowCols := len(strings.Split(lines[i], ","))
		if rowCols != headerCols {
			t.Fatalf("row %q has %d fields, want %d (matching header)", lines[i], rowCols, headerCols)
		}
	}

	// CSVQ が実際にテーブルを読み込めることを確認する
	db, err := sql.Open("csvq", dbDir)
	if err != nil {
		t.Fatalf("sql.Open() error: %v", err)
	}
	defer db.Close()

	row := db.QueryRow(CodesSelect + " WHERE code = ?", "abc123")
	var code string
	var created, updated string
	var disabled, name sql.NullString
	if err := row.Scan(&code, &created, &updated, &disabled, &name); err != nil {
		t.Fatalf("CSVQ query failed after migration: %v", err)
	}
	if code != "abc123" {
		t.Errorf("code = %q, want %q", code, "abc123")
	}
}
