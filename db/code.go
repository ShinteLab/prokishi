package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/xerrors"
)

const CodesColumns = "code,created_date,updated_date,disabled"
const CodesSelect = "SELECT code,DATETIME(created_date),DATETIME(updated_date),disabled FROM codes"
const CodesCount = "SELECT COUNT(code) FROM codes WHERE disabled IS NULL OR disabled <> 'true'"

type Code struct {
	Code     string
	Created  time.Time
	Updated  time.Time
	Disabled bool
}

func createCode(row scanner) (*Code, error) {
	var c Code
	var disabled sql.NullString
	err := row.Scan(&c.Code, &c.Created, &c.Updated, &disabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, xerrors.Errorf("Scan() error: %w", err)
	}
	c.Disabled = disabled.Valid && disabled.String == "true"
	return &c, nil
}

func CountCodes(ctx context.Context) (int, error) {
	row, err := getRow(ctx, CodesCount)
	if err != nil {
		return -1, xerrors.Errorf("getRow(count code) error: %w", err)
	}

	num := 0
	err = row.Scan(&num)
	if err != nil {
		return -1, xerrors.Errorf("Scan() error: %w", err)
	}
	return num, nil
}

func SelectCode(ctx context.Context, code string) (*Code, error) {
	s := CodesSelect + " WHERE code = ? AND (disabled IS NULL OR disabled <> 'true')"
	row, err := getRow(ctx, s, code)
	if err != nil {
		return nil, xerrors.Errorf("getRow(code) error: %w", err)
	}
	if row != nil {
		return createCode(row)
	}
	return nil, nil
}

func FindCodes(ctx context.Context) ([]*Code, error) {
	s := CodesSelect + " ORDER BY updated_date DESC"
	rows, err := getRows(ctx, s)
	if err != nil {
		return nil, xerrors.Errorf("getRows(code) error: %w", err)
	}
	codes := make([]*Code, 0)
	for rows.Next() {
		c, err := createCode(rows)
		if err != nil {
			return nil, xerrors.Errorf("createCode() error: %w", err)
		} else if c == nil {
			break
		}
		codes = append(codes, c)
	}
	return codes, nil
}

func InsertCode(code string) error {
	now := time.Now()
	zero := time.Time{}
	s := fmt.Sprintf("INSERT INTO codes (%s) VALUES (?,?,?,?)", CodesColumns)
	err := run(s, code, now, zero, "")
	if err != nil {
		return xerrors.Errorf("run() error: %w", err)
	}
	return nil
}

func DisableCode(code string) error {
	s := "UPDATE codes SET disabled = 'true' WHERE code = ?"
	if err := run(s, code); err != nil {
		return xerrors.Errorf("run() error: %w", err)
	}
	return nil
}

func EnableCode(code string) error {
	s := "UPDATE codes SET disabled = '' WHERE code = ?"
	if err := run(s, code); err != nil {
		return xerrors.Errorf("run() error: %w", err)
	}
	return nil
}

func UpdateCode(code string) error {
	now := time.Now()
	s := "UPDATE codes SET updated_date = ? WHERE code = ?"
	err := run(s, now, code)
	if err != nil {
		return xerrors.Errorf("run() error: %w", err)
	}
	return nil
}

func DeleteCode(code string) error {
	s := "DELETE FROM codes WHERE code = ?"
	err := run(s, code)
	if err != nil {
		return xerrors.Errorf("run() error: %w", err)
	}
	return nil
}
