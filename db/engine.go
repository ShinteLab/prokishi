package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/xerrors"
)

const EnginesColumns = "id,path,created_date,updated_date,name"
const EnginesSelect = "SELECT id,path,DATETIME(created_date),DATETIME(updated_date),name FROM engines"

type Engine struct {
	ID      string
	Path    string
	Created time.Time
	Updated time.Time
	Name    string
}

func createEngine(row scanner) (*Engine, error) {
	var e Engine
	var name sql.NullString
	err := row.Scan(&e.ID, &e.Path, &e.Created, &e.Updated, &name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, xerrors.Errorf("Scan() error: %w", err)
	}
	e.Name = name.String
	return &e, nil
}

func SelectEngine(ctx context.Context, id string) (*Engine, error) {
	s := EnginesSelect + " WHERE id = ?"
	row, err := getRow(ctx, s, id)
	if err != nil {
		return nil, xerrors.Errorf("getRow(engine) error: %w", err)
	}

	if row != nil {
		return createEngine(row)
	}
	return nil, nil
}

func FindEngines(ctx context.Context) ([]*Engine, error) {
	s := EnginesSelect + " ORDER BY updated_date DESC"
	rows, err := getRows(ctx, s)
	if err != nil {
		return nil, xerrors.Errorf("getRows(engine) error: %w", err)
	}
	engines := make([]*Engine, 0)
	for rows.Next() {
		e, err := createEngine(rows)
		if err != nil {
			return nil, xerrors.Errorf("createEngine() error: %w", err)
		} else if e == nil {
			break
		}
		engines = append(engines, e)
	}
	return engines, nil
}

func InsertEngine(id string, path string, name string) error {

	if _, err := os.Stat(path); err != nil {
		return xerrors.Errorf("os.Stat() error: %w", err)
	}

	now := time.Now()
	s := fmt.Sprintf("INSERT INTO engines (%s) VALUES (?,?,?,?,?)", EnginesColumns)
	err := run(s, id, path, now, now, name)
	if err != nil {
		return xerrors.Errorf("run() error: %w", err)
	}
	return nil
}

func UpdateEngineName(id string, name string) error {
	now := time.Now()
	s := "UPDATE engines SET name = ?, updated_date = ? WHERE id = ?"
	if err := run(s, name, now, id); err != nil {
		return xerrors.Errorf("run() error: %w", err)
	}
	return nil
}

func DeleteEngine(id string) error {
	s := "DELETE FROM engines WHERE id = ?"
	err := run(s, id)
	if err != nil {
		return xerrors.Errorf("run() error: %w", err)
	}
	return nil
}
