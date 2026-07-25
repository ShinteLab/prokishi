package server

import (
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/ShinteLab/prokishi/db"
	"github.com/ShinteLab/prokishi/internal/testfakeengine"
	"github.com/ShinteLab/prokishi/usi"
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

// verifyAuthentication's model: when useAuth is false, every code
// (including empty) is accepted regardless of what's in the DB. When
// useAuth is true, the code must resolve via db.SelectCode, which itself
// excludes disabled codes and returns nil for anything unregistered.

func TestVerifyAuthentication_AuthDisabled(t *testing.T) {
	setupTestDB(t)

	s := &Server{engines: &sync.Map{}, useAuth: false}

	// useAuth is off: any code, including an empty or unregistered one, is
	// accepted regardless of DB contents.
	if !s.verifyAuthentication("") {
		t.Fatalf("verifyAuthentication(\"\") = false, want true when useAuth is disabled")
	}
	if !s.verifyAuthentication("anything") {
		t.Fatalf("verifyAuthentication(\"anything\") = false, want true when useAuth is disabled")
	}
}

func TestVerifyAuthentication_ValidCode(t *testing.T) {
	setupTestDB(t)

	const code = "valid-code"
	if err := db.InsertCode(code, ""); err != nil {
		t.Fatalf("db.InsertCode() error: %v", err)
	}

	s := &Server{engines: &sync.Map{}, useAuth: true}

	if !s.verifyAuthentication(code) {
		t.Fatalf("verifyAuthentication(%q) = false, want true for a registered code", code)
	}
}

func TestVerifyAuthentication_InvalidCode(t *testing.T) {
	setupTestDB(t)

	if err := db.InsertCode("valid-code", ""); err != nil {
		t.Fatalf("db.InsertCode() error: %v", err)
	}

	s := &Server{engines: &sync.Map{}, useAuth: true}

	if s.verifyAuthentication("wrong-code") {
		t.Fatalf("verifyAuthentication(\"wrong-code\") = true, want false for an unregistered code")
	}
}

func TestVerifyAuthentication_DisabledCode(t *testing.T) {
	setupTestDB(t)

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

	s := &Server{engines: &sync.Map{}, useAuth: true}

	if !s.verifyAuthentication(activeCode) {
		t.Fatalf("verifyAuthentication(%q) = false, want true for the still-active code", activeCode)
	}
	if s.verifyAuthentication(code) {
		t.Fatalf("verifyAuthentication(%q) = true, want false for a disabled code", code)
	}
}

// TestVerifyAuthentication_NoCodesRegistered_AuthEnabled documents that,
// unlike an emergent "open door" behavior based on code count, useAuth=true
// with an empty codes table denies everything: db.SelectCode finds nothing
// for any code, including an empty one.
func TestVerifyAuthentication_NoCodesRegistered_AuthEnabled(t *testing.T) {
	setupTestDB(t)

	s := &Server{engines: &sync.Map{}, useAuth: true}

	if s.verifyAuthentication("") {
		t.Fatalf("verifyAuthentication(\"\") = true, want false: useAuth is on and no codes are registered")
	}
	if s.verifyAuthentication("anything") {
		t.Fatalf("verifyAuthentication(\"anything\") = true, want false: useAuth is on and no codes are registered")
	}
}

func TestStartEngine_EmptyEngineID(t *testing.T) {
	setupTestDB(t)

	s := &Server{engines: &sync.Map{}}

	connID, engineName, enginePath, pid, err := s.startEngine("")
	if err == nil {
		t.Fatalf("startEngine(\"\") error = nil, want error")
	}
	if !strings.Contains(err.Error(), "EngineId required") {
		t.Fatalf("startEngine(\"\") error = %v, want message containing %q", err, "EngineId required")
	}
	if connID != "" || engineName != "" || enginePath != "" || pid != 0 {
		t.Fatalf("startEngine(\"\") = (%q, %q, %q, %d), want all zero on error", connID, engineName, enginePath, pid)
	}
}

func TestStartEngine_UnknownEngineID(t *testing.T) {
	setupTestDB(t)

	s := &Server{engines: &sync.Map{}}

	connID, engineName, enginePath, pid, err := s.startEngine("does-not-exist")
	if err == nil {
		t.Fatalf("startEngine(\"does-not-exist\") error = nil, want error")
	}
	if !strings.Contains(err.Error(), "Engine is Not Found") {
		t.Fatalf("startEngine(\"does-not-exist\") error = %v, want message containing %q", err, "Engine is Not Found")
	}
	if connID != "" || engineName != "" || enginePath != "" || pid != 0 {
		t.Fatalf("startEngine(\"does-not-exist\") = (%q, %q, %q, %d), want all zero on error", connID, engineName, enginePath, pid)
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

	connID, gotName, gotPath, pid, err := s.startEngine(engineID)
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
	if pid <= 0 {
		t.Fatalf("startEngine(%q) pid = %d, want a positive process id", engineID, pid)
	}

	v, ok := s.engines.Load(connID)
	if !ok {
		t.Fatalf("startEngine(%q) did not store the engine under connID %q", engineID, connID)
	}
	sender, ok := v.(*usi.Sender)
	if !ok {
		t.Fatalf("s.engines.Load(%q) value has type %T, want *usi.Sender", connID, v)
	}
	if sender.Pid() != pid {
		t.Fatalf("sender.Pid() = %d, want %d (matching startEngine's returned pid)", sender.Pid(), pid)
	}

	// Ask the fake engine to exit cleanly rather than calling
	// sender.Terminate() ourselves: NewSender's internal goroutine already
	// calls Terminate() once the process exits, and Terminate() is not
	// safe to invoke twice (see usi/sender_integration_test.go).
	t.Cleanup(func() {
		_ = sender.Send("quit")
	})
}
