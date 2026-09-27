package main

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want slog.Level
	}{
		{name: "dbg alias", in: "dbg", want: slog.LevelDebug},
		{name: "debug", in: "debug", want: slog.LevelDebug},
		{name: "info", in: "info", want: slog.LevelInfo},
		{name: "information alias", in: "information", want: slog.LevelInfo},
		{name: "warn", in: "warn", want: slog.LevelWarn},
		{name: "warning alias", in: "warning", want: slog.LevelWarn},
		{name: "err alias", in: "err", want: slog.LevelError},
		{name: "error", in: "error", want: slog.LevelError},
		{name: "unknown defaults to warn", in: "bogus", want: slog.LevelWarn},
		{name: "empty defaults to warn", in: "", want: slog.LevelWarn},
		{name: "case insensitive", in: "DEBUG", want: slog.LevelDebug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseLogLevel(tt.in); got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestCreateIniFile(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, iniFileName)

	if err := createIniFile(p); err != nil {
		t.Fatalf("createIniFile(%q) error: %v", p, err)
	}

	var got IniFile
	if _, err := toml.DecodeFile(p, &got); err != nil {
		t.Fatalf("toml.DecodeFile(%q) error: %v", p, err)
	}

	want := IniFile{
		Host:     "localhost",
		Port:     8080,
		Code:     "",
		EngineId: "",
		Level:    "warn",
	}
	if got != want {
		t.Errorf("createIniFile() wrote %+v, want %+v", got, want)
	}
}

// TestVersionEmbedded checks that the embedded version file is used and
// matches the master (_cmd/prokishi-server/version), which _cmd/version.go
// keeps in sync.
func TestVersionEmbedded(t *testing.T) {
	if version == "" {
		t.Fatal("embedded version is empty")
	}
	master, err := os.ReadFile(filepath.Join("..", "prokishi-server", "version"))
	if err != nil {
		t.Fatalf("read master version: %v", err)
	}
	if want := strings.TrimSpace(string(master)); version != want {
		t.Errorf("version = %q, want %q (run `go run _cmd/version.go` to sync)", version, want)
	}
}

// TestLoadIniFileMissing checks that a missing prokishi.ini is created as a
// template and reported as errIniCreated, without reading os.Stdin (stdin is
// the USI stream from the shogi GUI).
func TestLoadIniFileMissing(t *testing.T) {
	skipUnlessDevMode(t)
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Fatalf("failed to restore cwd: %v", err)
		}
	}()

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("os.Chdir(%q) error: %v", tmp, err)
	}

	err = loadIniFile()
	if !errors.Is(err, errIniCreated) {
		t.Fatalf("loadIniFile() error = %v, want errIniCreated", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, iniFileName)); err != nil {
		t.Errorf("template %s was not created: %v", iniFileName, err)
	}
}

// TestLoadIniFileExisting exercises the "file already exists" branch of
// loadIniFile().
func TestLoadIniFileExisting(t *testing.T) {
	skipUnlessDevMode(t)
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}
	defer func() {
		if err := os.Chdir(origWd); err != nil {
			t.Fatalf("failed to restore cwd: %v", err)
		}
	}()

	origIniFile := iniFile
	defer func() { iniFile = origIniFile }()

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("os.Chdir(%q) error: %v", tmp, err)
	}

	want := IniFile{
		Host:     "example.com",
		Port:     1234,
		Code:     "AUTHCODE",
		EngineId: "engine-uuid",
		Level:    "debug",
	}

	p := filepath.Join(tmp, iniFileName)
	fp, err := os.Create(p)
	if err != nil {
		t.Fatalf("os.Create(%q) error: %v", p, err)
	}
	if err := toml.NewEncoder(fp).Encode(&want); err != nil {
		fp.Close()
		t.Fatalf("toml.Encode() error: %v", err)
	}
	if err := fp.Close(); err != nil {
		t.Fatalf("fp.Close() error: %v", err)
	}

	// devMode is true under `go test` (no -tags production), so loadIniFile()
	// resolves prokishi.GetRunDir(true) == os.Getwd(), which matches the
	// tempdir we just chdir'd into.
	iniFile = IniFile{}
	if err := loadIniFile(); err != nil {
		t.Fatalf("loadIniFile() error: %v", err)
	}

	if iniFile != want {
		t.Errorf("loadIniFile() populated iniFile = %+v, want %+v", iniFile, want)
	}
}

// skipUnlessDevMode skips tests that rely on dev mode resolving files from
// the current directory (they chdir into a tempdir).
func skipUnlessDevMode(t *testing.T) {
	t.Helper()
	if !devMode {
		t.Skip("requires dev mode (built without -tags production)")
	}
}
