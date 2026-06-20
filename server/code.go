package server

import (
	"context"
	"fmt"
	"prokishi/db"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

func GenerateCode() error {
	uid := uuid.New()
	return db.InsertCode(uid.String(), "")
}

func RegisterCode(code string) error {
	return db.InsertCode(code, "")
}

func PrintCodes() error {
	codes, err := db.FindCodes(context.Background())
	if err != nil {
		return xerrors.Errorf("db.FindCodes() error: %w", err)
	}

	fmt.Printf("%-40s|%-37s\n", "Code", "Used")
	fmt.Printf("%-40s|%-37s\n", strings.Repeat("-", 40), strings.Repeat("-", 37))
	for _, c := range codes {
		fmt.Printf("%-40s|%37s\n", c.Code, c.Updated)
	}
	return nil
}

func DeleteCode(code string) error {
	return db.DeleteCode(code)
}
