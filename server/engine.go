package server

import (
	"context"
	"fmt"
	"prokishi/db"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

func GenerateEngineId(p string) error {
	uid := uuid.New()
	id := uid.String()
	err := db.InsertEngine(id, p, "")
	if err != nil {
		return xerrors.Errorf("db.InsertEngine() error: %w", err)
	}
	fmt.Println("Generate EngineID->", id)
	return nil
}

func RegisterEngineId(id string, p string) error {

	err := db.InsertEngine(id, p, "")
	if err != nil {
		return xerrors.Errorf("db.InsertEngine() error: %w", err)
	}
	fmt.Println("Register EngineID->", id)
	return nil
}

func PrintEngineIds() error {
	engines, err := db.FindEngines(context.Background())
	if err != nil {
		return xerrors.Errorf("db.FindEngines() error: %w", err)
	}

	fmt.Printf("%-40s|%-37s\n", "ID", "Created")
	fmt.Printf("%-40s|%37s\n", strings.Repeat("-", 40), strings.Repeat("-", 37))
	for _, e := range engines {
		fmt.Printf("%-40s|%-37s\n%s\n", e.ID, e.Created, e.Path)
	}
	return nil
}

func DeleteEngineId(id string) error {
	return db.DeleteEngine(id)
}
