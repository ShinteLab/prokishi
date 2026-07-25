package server

import (
	"context"
	"testing"

	"shinte/prokishi/db"
)

// These tests confirm GenerateCode/RegisterCode/DeleteCode correctly
// delegate to the db package; db's own logic is already covered in depth
// by db's test suite, so these stay shallow.

func TestGenerateCode(t *testing.T) {
	setupTestDB(t)

	if err := GenerateCode(); err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}

	codes, err := db.FindCodes(context.Background())
	if err != nil {
		t.Fatalf("db.FindCodes() error: %v", err)
	}
	if len(codes) != 1 {
		t.Fatalf("len(codes) = %d, want 1", len(codes))
	}
	if codes[0].Code == "" {
		t.Fatalf("GenerateCode() produced an empty code")
	}
}

func TestRegisterCode(t *testing.T) {
	setupTestDB(t)

	const code = "my-code"
	if err := RegisterCode(code); err != nil {
		t.Fatalf("RegisterCode() error: %v", err)
	}

	got, err := db.SelectCode(context.Background(), code)
	if err != nil {
		t.Fatalf("db.SelectCode() error: %v", err)
	}
	if got == nil {
		t.Fatalf("RegisterCode(%q) did not persist the code", code)
	}
}

func TestDeleteCode(t *testing.T) {
	setupTestDB(t)

	const code = "to-delete"
	if err := RegisterCode(code); err != nil {
		t.Fatalf("RegisterCode() error: %v", err)
	}

	if err := DeleteCode(code); err != nil {
		t.Fatalf("DeleteCode() error: %v", err)
	}

	got, err := db.SelectCode(context.Background(), code)
	if err != nil {
		t.Fatalf("db.SelectCode() error: %v", err)
	}
	if got != nil {
		t.Fatalf("DeleteCode(%q) did not remove the code", code)
	}
}
