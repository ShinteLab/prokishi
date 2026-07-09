package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"prokishi"

	_ "github.com/mithrandie/csvq-driver"
	"golang.org/x/xerrors"
)

var gDB *sql.DB
var AlreadyErr = fmt.Errorf("already database error")

type scanner interface {
	Scan(dest ...any) error
}

func getRow(ctx context.Context, sql string, args ...interface{}) (*sql.Row, error) {
	if gDB == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return gDB.QueryRowContext(ctx, sql, args...), nil
}

func getRows(ctx context.Context, sql string, args ...interface{}) (*sql.Rows, error) {
	if gDB == nil {
		return nil, fmt.Errorf("db is nil")
	}
	return gDB.QueryContext(ctx, sql, args...)
}

func run(sql string, args ...interface{}) error {

	if gDB == nil {
		return fmt.Errorf("db is nil")
	}
	stmt, err := gDB.Prepare(sql)
	if err != nil {
		return xerrors.Errorf("db.Prepare() error: %w", err)
	}

	_, err = stmt.Exec(args...)
	if err != nil {
		return xerrors.Errorf("stmt.Exec() error: %w", err)
	}
	return nil
}

func Open(dev bool) error {
	dir, err := getPath(dev)
	if err != nil {
		return xerrors.Errorf("getPath() error: %w", err)
	}

	db, err := sql.Open("csvq", dir)
	if err != nil {
		return xerrors.Errorf("sql.Open() error: %w", err)
	}

	if gDB != nil {
		gDB.Close()
	}
	gDB = db
	return nil
}

func Close() error {
	return gDB.Close()
}

func getPath(dev bool) (string, error) {
	dir, err := prokishi.GetRunDir(dev)
	if err != nil {
		return "", xerrors.Errorf("GetRunDir() error: %w", err)
	}
	return filepath.Join(dir, "db"), nil
}

// 開発モード時は別位置に作成
func Init(dev bool) error {

	dir, err := getPath(dev)
	if err != nil {
		return xerrors.Errorf("getPath() error: %w", err)
	}
	return initTables(dir)
}

func initTables(dir string) error {

	os.Mkdir(dir, 0666)

	cp := filepath.Join(dir, "codes.csv")
	if _, err := os.Stat(cp); err != nil {
		err = createTableFile(cp, CodesColumns)
		if err != nil {
			return xerrors.Errorf("createTables(codes) error: %w", err)
		}
	} else {
		migrateCodesColumns(cp)
	}

	ep := filepath.Join(dir, "engines.csv")
	if _, err := os.Stat(ep); err != nil {
		err = createTableFile(ep, EnginesColumns)
		if err != nil {
			return xerrors.Errorf("createTables(engines) error: %w", err)
		}
	} else {
		migrateEnginesColumns(ep)
	}

	return nil
}

func migrateCSVColumn(path, column string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// CRLF 正規化
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	trailingNewline := strings.HasSuffix(content, "\n")
	content = strings.TrimSuffix(content, "\n")
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return
	}

	// ヘッダーに列名が既にあれば何もしない
	for _, col := range strings.Split(lines[0], ",") {
		if strings.TrimSpace(col) == column {
			return
		}
	}

	lines[0] += "," + column
	numCols := len(strings.Split(lines[0], ","))

	// 各データ行の列数をヘッダーに合わせる（既存フィールドは除去せず不足分のみ追加）
	for i := 1; i < len(lines); i++ {
		parts := strings.Split(lines[i], ",")
		for len(parts) < numCols {
			parts = append(parts, "")
		}
		lines[i] = strings.Join(parts, ",")
	}

	result := strings.Join(lines, "\n")
	if trailingNewline {
		result += "\n"
	}
	os.WriteFile(path, []byte(result), 0644)
}

func migrateCodesColumns(path string) {
	migrateCSVColumn(path, "disabled")
	migrateCSVColumn(path, "name")
}
func migrateEnginesColumns(path string) { migrateCSVColumn(path, "name") }

func createTableFile(fn string, cols string) error {
	fp, err := os.Create(fn)
	if err != nil {
		return xerrors.Errorf("os.Open() error: %w", err)
	}
	defer fp.Close()

	_, err = fp.Write([]byte(cols))
	if err != nil {
		return xerrors.Errorf("Write() error: %w", err)
	}
	return nil
}
