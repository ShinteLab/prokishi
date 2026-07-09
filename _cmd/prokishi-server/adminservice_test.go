package main

import (
	"path/filepath"
	"testing"

	"prokishi/db"
)

// NOTE: AdminService.SaveClientConfig and AdminService.SelectEnginePath are
// intentionally NOT covered here - both call application.Get().Dialog...,
// which requires a live, running Wails application context (application.Get
// panics/returns nil outside of app.Run()). They cannot be constructed or
// exercised in a plain `go test` process.

func TestAdminService_ListEngines_Empty(t *testing.T) {
	setupTestDB(t)
	a := &AdminService{}

	got, err := a.ListEngines()
	if err != nil {
		t.Fatalf("ListEngines() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(ListEngines()) = %d, want 0", len(got))
	}
}

func TestAdminService_RegisterEngine_ListEngines_Roundtrip(t *testing.T) {
	dir := setupTestDB(t)
	a := &AdminService{}

	enginePath := filepath.Join(dir, "engine.exe")
	createPlaceholderFile(t, enginePath)

	id, err := a.RegisterEngine(enginePath, "MyEngine")
	if err != nil {
		t.Fatalf("RegisterEngine() error: %v", err)
	}
	if id == "" {
		t.Fatal("RegisterEngine() returned empty id")
	}

	got, err := a.ListEngines()
	if err != nil {
		t.Fatalf("ListEngines() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(ListEngines()) = %d, want 1", len(got))
	}
	if got[0].ID != id {
		t.Errorf("got[0].ID = %q, want %q", got[0].ID, id)
	}
	if got[0].Name != "MyEngine" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "MyEngine")
	}
	if got[0].Path != enginePath {
		t.Errorf("got[0].Path = %q, want %q", got[0].Path, enginePath)
	}
	if got[0].Created == "" {
		t.Error("got[0].Created is empty")
	}
}

func TestAdminService_RegisterEngine_NonexistentPathFails(t *testing.T) {
	dir := setupTestDB(t)
	a := &AdminService{}

	_, err := a.RegisterEngine(filepath.Join(dir, "does-not-exist.exe"), "MyEngine")
	if err == nil {
		t.Fatal("RegisterEngine() with nonexistent path: got nil error, want error")
	}
}

func TestAdminService_UpdateEngineName(t *testing.T) {
	dir := setupTestDB(t)
	a := &AdminService{}

	enginePath := filepath.Join(dir, "engine.exe")
	createPlaceholderFile(t, enginePath)
	id, err := a.RegisterEngine(enginePath, "Original")
	if err != nil {
		t.Fatalf("RegisterEngine() error: %v", err)
	}

	if err := a.UpdateEngineName(id, "Renamed"); err != nil {
		t.Fatalf("UpdateEngineName() error: %v", err)
	}

	got, err := a.ListEngines()
	if err != nil {
		t.Fatalf("ListEngines() error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Renamed" {
		t.Errorf("ListEngines() = %+v, want name %q", got, "Renamed")
	}
}

func TestAdminService_DeleteEngine(t *testing.T) {
	dir := setupTestDB(t)
	a := &AdminService{}

	enginePath := filepath.Join(dir, "engine.exe")
	createPlaceholderFile(t, enginePath)
	id, err := a.RegisterEngine(enginePath, "ToDelete")
	if err != nil {
		t.Fatalf("RegisterEngine() error: %v", err)
	}

	if err := a.DeleteEngine(id); err != nil {
		t.Fatalf("DeleteEngine() error: %v", err)
	}

	got, err := a.ListEngines()
	if err != nil {
		t.Fatalf("ListEngines() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(ListEngines()) after delete = %d, want 0", len(got))
	}
}

func TestAdminService_GenerateCode_ListCodes_Roundtrip(t *testing.T) {
	setupTestDB(t)
	a := &AdminService{}

	code, err := a.GenerateCode("MyCode")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}
	if code == "" {
		t.Fatal("GenerateCode() returned empty code")
	}

	got, err := a.ListCodes()
	if err != nil {
		t.Fatalf("ListCodes() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(ListCodes()) = %d, want 1", len(got))
	}
	if got[0].Code != code {
		t.Errorf("got[0].Code = %q, want %q", got[0].Code, code)
	}
	if got[0].Name != "MyCode" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "MyCode")
	}
	if got[0].Disabled {
		t.Error("got[0].Disabled = true, want false for freshly generated code")
	}
	if got[0].Created == "" {
		t.Error("got[0].Created is empty")
	}
	// A never-used code has a zero-value Updated timestamp in the DB; the
	// service layer should blank that out rather than format the zero time.
	if got[0].Used != "" {
		t.Errorf("got[0].Used = %q, want empty for a never-used code", got[0].Used)
	}
}

func TestAdminService_ListCodes_UsedFormatting(t *testing.T) {
	setupTestDB(t)
	a := &AdminService{}

	code, err := a.GenerateCode("MyCode")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}

	// Simulate the code having been used for authentication: the server
	// package calls db.UpdateCode on successful auth (server/handler.go),
	// which sets updated_date to now. AdminService itself never calls this,
	// so we drive the db package directly to set up the "used" state and
	// verify AdminService.ListCodes formats it correctly.
	if err := db.UpdateCode(code); err != nil {
		t.Fatalf("db.UpdateCode() error: %v", err)
	}

	got, err := a.ListCodes()
	if err != nil {
		t.Fatalf("ListCodes() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(ListCodes()) = %d, want 1", len(got))
	}
	if got[0].Used == "" {
		t.Error("got[0].Used is empty, want a formatted timestamp after db.UpdateCode")
	}
}

func TestAdminService_RegisterCode(t *testing.T) {
	setupTestDB(t)
	a := &AdminService{}

	if err := a.RegisterCode("manual-code", "Manual"); err != nil {
		t.Fatalf("RegisterCode() error: %v", err)
	}

	got, err := a.ListCodes()
	if err != nil {
		t.Fatalf("ListCodes() error: %v", err)
	}
	if len(got) != 1 || got[0].Code != "manual-code" || got[0].Name != "Manual" {
		t.Errorf("ListCodes() = %+v, want code %q name %q", got, "manual-code", "Manual")
	}
}

func TestAdminService_UpdateCodeName(t *testing.T) {
	setupTestDB(t)
	a := &AdminService{}

	code, err := a.GenerateCode("Original")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}

	if err := a.UpdateCodeName(code, "Renamed"); err != nil {
		t.Fatalf("UpdateCodeName() error: %v", err)
	}

	got, err := a.ListCodes()
	if err != nil {
		t.Fatalf("ListCodes() error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Renamed" {
		t.Errorf("ListCodes() = %+v, want name %q", got, "Renamed")
	}
}

func TestAdminService_DisableCode_EnableCode(t *testing.T) {
	setupTestDB(t)
	a := &AdminService{}

	code, err := a.GenerateCode("MyCode")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}

	if err := a.DisableCode(code); err != nil {
		t.Fatalf("DisableCode() error: %v", err)
	}
	got, err := a.ListCodes()
	if err != nil {
		t.Fatalf("ListCodes() error: %v", err)
	}
	if len(got) != 1 || !got[0].Disabled {
		t.Errorf("ListCodes() after DisableCode = %+v, want Disabled=true", got)
	}

	if err := a.EnableCode(code); err != nil {
		t.Fatalf("EnableCode() error: %v", err)
	}
	got, err = a.ListCodes()
	if err != nil {
		t.Fatalf("ListCodes() error: %v", err)
	}
	if len(got) != 1 || got[0].Disabled {
		t.Errorf("ListCodes() after EnableCode = %+v, want Disabled=false", got)
	}
}

func TestAdminService_DeleteCode(t *testing.T) {
	setupTestDB(t)
	a := &AdminService{}

	code, err := a.GenerateCode("MyCode")
	if err != nil {
		t.Fatalf("GenerateCode() error: %v", err)
	}

	if err := a.DeleteCode(code); err != nil {
		t.Fatalf("DeleteCode() error: %v", err)
	}

	got, err := a.ListCodes()
	if err != nil {
		t.Fatalf("ListCodes() error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len(ListCodes()) after delete = %d, want 0", len(got))
	}
}
