package db

import (
	"context"
	"testing"
	"time"
)

func TestInsertCode_SelectCode_Roundtrip(t *testing.T) {
	setupTestDB(t)
	ctx := context.Background()

	if err := InsertCode("CODE1", "first code"); err != nil {
		t.Fatalf("InsertCode() error: %v", err)
	}

	c, err := SelectCode(ctx, "CODE1")
	if err != nil {
		t.Fatalf("SelectCode() error: %v", err)
	}
	if c == nil {
		t.Fatal("SelectCode() returned nil for an inserted code")
	}
	if c.Code != "CODE1" {
		t.Errorf("Code = %q, want %q", c.Code, "CODE1")
	}
	if c.Name != "first code" {
		t.Errorf("Name = %q, want %q", c.Name, "first code")
	}
	if c.Disabled {
		t.Errorf("Disabled = true, want false for a freshly inserted code")
	}
	if d := time.Since(c.Created); d < -time.Minute || d > time.Minute {
		t.Errorf("Created = %v, expected to be close to now", c.Created)
	}
}

func TestSelectCode_NotFound(t *testing.T) {
	setupTestDB(t)
	ctx := context.Background()

	c, err := SelectCode(ctx, "NOPE")
	if err != nil {
		t.Fatalf("SelectCode() error: %v", err)
	}
	if c != nil {
		t.Fatalf("SelectCode() = %+v, want nil for an unknown code", c)
	}
}

func TestCountCodes_ExcludesDisabled(t *testing.T) {
	setupTestDB(t)
	ctx := context.Background()

	if err := InsertCode("CODE1", "a"); err != nil {
		t.Fatalf("InsertCode() error: %v", err)
	}
	if err := InsertCode("CODE2", "b"); err != nil {
		t.Fatalf("InsertCode() error: %v", err)
	}

	n, err := CountCodes(ctx)
	if err != nil {
		t.Fatalf("CountCodes() error: %v", err)
	}
	if n != 2 {
		t.Fatalf("CountCodes() = %d, want 2", n)
	}

	if err := DisableCode("CODE1"); err != nil {
		t.Fatalf("DisableCode() error: %v", err)
	}

	n, err = CountCodes(ctx)
	if err != nil {
		t.Fatalf("CountCodes() error: %v", err)
	}
	if n != 1 {
		t.Fatalf("CountCodes() after disabling one code = %d, want 1", n)
	}
}

func TestDisableCode_EnableCode(t *testing.T) {
	setupTestDB(t)
	ctx := context.Background()

	if err := InsertCode("CODE1", "a"); err != nil {
		t.Fatalf("InsertCode() error: %v", err)
	}
	if err := DisableCode("CODE1"); err != nil {
		t.Fatalf("DisableCode() error: %v", err)
	}

	// SelectCode's WHERE clause filters out disabled codes.
	c, err := SelectCode(ctx, "CODE1")
	if err != nil {
		t.Fatalf("SelectCode() error: %v", err)
	}
	if c != nil {
		t.Fatalf("SelectCode() = %+v, want nil for a disabled code", c)
	}

	if err := EnableCode("CODE1"); err != nil {
		t.Fatalf("EnableCode() error: %v", err)
	}

	c, err = SelectCode(ctx, "CODE1")
	if err != nil {
		t.Fatalf("SelectCode() error: %v", err)
	}
	if c == nil {
		t.Fatal("SelectCode() returned nil after re-enabling the code")
	}
	if c.Disabled {
		t.Errorf("Disabled = true after EnableCode(), want false")
	}
}

func TestFindCodes_IncludesDisabledAndOrdersByUpdatedDateDesc(t *testing.T) {
	setupTestDB(t)
	ctx := context.Background()

	if err := InsertCode("CODE1", "a"); err != nil {
		t.Fatalf("InsertCode() error: %v", err)
	}
	if err := InsertCode("CODE2", "b"); err != nil {
		t.Fatalf("InsertCode() error: %v", err)
	}
	if err := DisableCode("CODE2"); err != nil {
		t.Fatalf("DisableCode() error: %v", err)
	}

	// FindCodes (unlike SelectCode) has no disabled filter, so a disabled
	// code must still show up. Drive updated_date apart (with a real sleep,
	// since DATETIME() resolution is unknown) so ordering is deterministic:
	// CODE1 updated last should sort first.
	if err := UpdateCode("CODE2"); err != nil {
		t.Fatalf("UpdateCode() error: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)
	if err := UpdateCode("CODE1"); err != nil {
		t.Fatalf("UpdateCode() error: %v", err)
	}

	codes, err := FindCodes(ctx)
	if err != nil {
		t.Fatalf("FindCodes() error: %v", err)
	}
	if len(codes) != 2 {
		t.Fatalf("FindCodes() returned %d codes, want 2 (disabled codes must still be listed)", len(codes))
	}
	if codes[0].Code != "CODE1" || codes[1].Code != "CODE2" {
		t.Fatalf("FindCodes() order = [%s, %s], want [CODE1, CODE2] (most recently updated first)", codes[0].Code, codes[1].Code)
	}
}

func TestUpdateCodeName(t *testing.T) {
	setupTestDB(t)
	ctx := context.Background()

	if err := InsertCode("CODE1", "old name"); err != nil {
		t.Fatalf("InsertCode() error: %v", err)
	}
	if err := UpdateCodeName("CODE1", "new name"); err != nil {
		t.Fatalf("UpdateCodeName() error: %v", err)
	}

	c, err := SelectCode(ctx, "CODE1")
	if err != nil {
		t.Fatalf("SelectCode() error: %v", err)
	}
	if c == nil {
		t.Fatal("SelectCode() returned nil")
	}
	if c.Name != "new name" {
		t.Errorf("Name = %q, want %q", c.Name, "new name")
	}
}

func TestDeleteCode(t *testing.T) {
	setupTestDB(t)
	ctx := context.Background()

	if err := InsertCode("CODE1", "a"); err != nil {
		t.Fatalf("InsertCode() error: %v", err)
	}
	if err := DeleteCode("CODE1"); err != nil {
		t.Fatalf("DeleteCode() error: %v", err)
	}

	c, err := SelectCode(ctx, "CODE1")
	if err != nil {
		t.Fatalf("SelectCode() error: %v", err)
	}
	if c != nil {
		t.Fatalf("SelectCode() = %+v, want nil after DeleteCode()", c)
	}

	codes, err := FindCodes(ctx)
	if err != nil {
		t.Fatalf("FindCodes() error: %v", err)
	}
	if len(codes) != 0 {
		t.Fatalf("FindCodes() returned %d codes, want 0 after DeleteCode()", len(codes))
	}
}
