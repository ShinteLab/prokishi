package server

import (
	"os"
	"strings"
	"sync"
	"testing"

	"prokishi/db"
	"prokishi/internal/testfakeengine"
	"prokishi/usi"
)

// chdirTemp creates a fresh temporary directory, changes the process's
// working directory into it, and restores the original working directory
// (plus closes the DB handle) once the test finishes. This mirrors the
// pattern in db/testhelper_test.go; that helper is unexported to package db
// and can't be imported here, so it's replicated locally.
func chdirTemp(t *testing.T) string {
	t.Helper()

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("os.Chdir() error: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
		if err := os.Chdir(origDir); err != nil {
			t.Fatalf("os.Chdir(restore) error: %v", err)
		}
	})

	return dir
}

// setupTestDB sandboxes a fresh CSVQ-backed DB (schema created via
// db.Init, connection opened via db.Open) in a temp dir.
func setupTestDB(t *testing.T) {
	t.Helper()

	chdirTemp(t)

	if err := db.Init(true); err != nil {
		t.Fatalf("db.Init() error: %v", err)
	}
	if err := db.Open(true); err != nil {
		t.Fatalf("db.Open() error: %v", err)
	}
}

func TestVerifyAuthentication_NoCodesRegistered(t *testing.T) {
	setupTestDB(t)

	s := &Server{engines: &sync.Map{}}

	// No codes registered at all: verifyAuthentication should be an "open
	// door" and accept any code, including an empty one.
	if !s.verifyAuthentication("") {
		t.Fatalf("verifyAuthentication(\"\") = false, want true when no codes are registered")
	}
	if !s.verifyAuthentication("anything") {
		t.Fatalf("verifyAuthentication(\"anything\") = false, want true when no codes are registered")
	}
}

func TestVerifyAuthentication_ValidCode(t *testing.T) {
	setupTestDB(t)

	const code = "valid-code"
	if err := db.InsertCode(code, ""); err != nil {
		t.Fatalf("db.InsertCode() error: %v", err)
	}

	s := &Server{engines: &sync.Map{}}

	if !s.verifyAuthentication(code) {
		t.Fatalf("verifyAuthentication(%q) = false, want true for a registered code", code)
	}
}

func TestVerifyAuthentication_InvalidCode(t *testing.T) {
	setupTestDB(t)

	if err := db.InsertCode("valid-code", ""); err != nil {
		t.Fatalf("db.InsertCode() error: %v", err)
	}

	s := &Server{engines: &sync.Map{}}

	if s.verifyAuthentication("wrong-code") {
		t.Fatalf("verifyAuthentication(\"wrong-code\") = true, want false for an unregistered code")
	}
}

func TestVerifyAuthentication_DisabledCode(t *testing.T) {
	setupTestDB(t)

	// db.CountCodes (which verifyAuthentication uses to decide whether auth
	// is required at all) only counts *enabled* codes ("disabled IS NULL OR
	// disabled <> 'true'"). If the disabled code under test were the only
	// row in the table, that count would be 0 and verifyAuthentication
	// would take the "no codes registered" open-door branch, passing the
	// test for the wrong reason. Register a second, active code so cnt > 0
	// and the disabled code is actually exercised against db.SelectCode's
	// "disabled" filter.
	const activeCode = "active-code"
	if err := db.InsertCode(activeCode, ""); err != nil {
		t.Fatalf("db.InsertCode() error: %v", err)
	}

	const code = "disabled-code"
	if err := db.InsertCode(code, ""); err != nil {
		t.Fatalf("db.InsertCode() error: %v", err)
	}
	if err := db.DisableCode(code); err != nil {
		t.Fatalf("db.DisableCode() error: %v", err)
	}

	s := &Server{engines: &sync.Map{}}

	if !s.verifyAuthentication(activeCode) {
		t.Fatalf("verifyAuthentication(%q) = false, want true for the still-active code", activeCode)
	}
	if s.verifyAuthentication(code) {
		t.Fatalf("verifyAuthentication(%q) = true, want false for a disabled code", code)
	}
}

// TestVerifyAuthentication_AllCodesDisabled documents a real, non-obvious
// behavior of verifyAuthentication/db.CountCodes: CountCodes only counts
// *enabled* codes, so if every registered code happens to be disabled, the
// active-code count is 0 and verifyAuthentication takes its "no codes
// registered" open-door branch - i.e. disabling every code does not lock
// the server down, it opens it back up to any code (including an empty
// one). This is captured as a test rather than "fixed" here since this
// phase is testing-only.
func TestVerifyAuthentication_AllCodesDisabled(t *testing.T) {
	setupTestDB(t)

	const code = "only-code"
	if err := db.InsertCode(code, ""); err != nil {
		t.Fatalf("db.InsertCode() error: %v", err)
	}
	if err := db.DisableCode(code); err != nil {
		t.Fatalf("db.DisableCode() error: %v", err)
	}

	s := &Server{engines: &sync.Map{}}

	if !s.verifyAuthentication("") {
		t.Fatalf("verifyAuthentication(\"\") = false, want true: with all codes disabled, CountCodes()==0 and verifyAuthentication falls back to open-door")
	}
}

func TestStartEngine_EmptyEngineID(t *testing.T) {
	setupTestDB(t)

	s := &Server{engines: &sync.Map{}}

	connID, engineName, enginePath, err := s.startEngine("")
	if err == nil {
		t.Fatalf("startEngine(\"\") error = nil, want error")
	}
	if !strings.Contains(err.Error(), "EngineId required") {
		t.Fatalf("startEngine(\"\") error = %v, want message containing %q", err, "EngineId required")
	}
	if connID != "" || engineName != "" || enginePath != "" {
		t.Fatalf("startEngine(\"\") = (%q, %q, %q), want all empty on error", connID, engineName, enginePath)
	}
}

func TestStartEngine_UnknownEngineID(t *testing.T) {
	setupTestDB(t)

	s := &Server{engines: &sync.Map{}}

	connID, engineName, enginePath, err := s.startEngine("does-not-exist")
	if err == nil {
		t.Fatalf("startEngine(\"does-not-exist\") error = nil, want error")
	}
	if !strings.Contains(err.Error(), "Engine is Not Found") {
		t.Fatalf("startEngine(\"does-not-exist\") error = %v, want message containing %q", err, "Engine is Not Found")
	}
	if connID != "" || engineName != "" || enginePath != "" {
		t.Fatalf("startEngine(\"does-not-exist\") = (%q, %q, %q), want all empty on error", connID, engineName, enginePath)
	}
}

func TestStartEngine_RegisteredEngine(t *testing.T) {
	setupTestDB(t)

	binPath := testfakeengine.Build(t)

	const engineID = "engine-1"
	const engineName = "Fake Engine"
	if err := db.InsertEngine(engineID, binPath, engineName); err != nil {
		t.Fatalf("db.InsertEngine() error: %v", err)
	}

	s := &Server{engines: &sync.Map{}}

	connID, gotName, gotPath, err := s.startEngine(engineID)
	if err != nil {
		t.Fatalf("startEngine(%q) error: %v", engineID, err)
	}
	if connID == "" {
		t.Fatalf("startEngine(%q) returned empty connID", engineID)
	}
	if gotName != engineName {
		t.Fatalf("startEngine(%q) engineName = %q, want %q", engineID, gotName, engineName)
	}
	if gotPath != binPath {
		t.Fatalf("startEngine(%q) enginePath = %q, want %q", engineID, gotPath, binPath)
	}

	v, ok := s.engines.Load(connID)
	if !ok {
		t.Fatalf("startEngine(%q) did not store the engine under connID %q", engineID, connID)
	}
	sender, ok := v.(*usi.Sender)
	if !ok {
		t.Fatalf("s.engines.Load(%q) value has type %T, want *usi.Sender", connID, v)
	}

	// Ask the fake engine to exit cleanly rather than calling
	// sender.Terminate() ourselves: NewSender's internal goroutine already
	// calls Terminate() once the process exits, and Terminate() is not
	// safe to invoke twice (see usi/sender_integration_test.go).
	t.Cleanup(func() {
		_ = sender.Send("quit")
	})
}
