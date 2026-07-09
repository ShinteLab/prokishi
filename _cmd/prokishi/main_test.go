package main

import (
	"log/slog"
	"os"
	"path/filepath"
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

func TestIsYes(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{name: "empty string is yes", in: "", want: true},
		{name: "lowercase y is yes", in: "y", want: true},
		{name: "uppercase Y is yes", in: "Y", want: true},
		{name: "n is no", in: "n", want: false},
		{name: "full word yes is not accepted", in: "yes", want: false},
		{name: "arbitrary text is no", in: "nope", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isYes(tt.in); got != tt.want {
				t.Errorf("isYes(%q) = %v, want %v", tt.in, got, tt.want)
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

// TestLoadIniFileExisting exercises only the "file already exists" branch of
// loadIniFile(). The "no file exists, prompt via os.Stdin" branch is skipped
// here because it reassigns process-wide os.Stdin, which is flaky/unsafe to
// exercise in a normal test.
func TestLoadIniFileExisting(t *testing.T) {
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

	// version is "" under `go test` (no -ldflags), so loadIniFile()
	// resolves dev-mode via prokishi.GetRunDir(true) == os.Getwd(),
	// which matches the tempdir we just chdir'd into.
	iniFile = IniFile{}
	if err := loadIniFile(); err != nil {
		t.Fatalf("loadIniFile() error: %v", err)
	}

	if iniFile != want {
		t.Errorf("loadIniFile() populated iniFile = %+v, want %+v", iniFile, want)
	}
}
