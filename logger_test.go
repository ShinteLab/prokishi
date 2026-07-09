package prokishi_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"prokishi"
	"strings"
	"testing"
)

func TestGetRunDirDev(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}
	defer func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("failed to restore cwd: %v", err)
		}
	}()

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("os.Chdir(%q) error: %v", tmp, err)
	}

	wantDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}

	got, err := prokishi.GetRunDir(true)
	if err != nil {
		t.Fatalf("GetRunDir(true) error: %v", err)
	}
	if got != wantDir {
		t.Errorf("GetRunDir(true) = %q, want %q", got, wantDir)
	}
}

func TestGetRunDirRelease(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() error: %v", err)
	}
	want := filepath.Dir(exe)

	got, err := prokishi.GetRunDir(false)
	if err != nil {
		t.Fatalf("GetRunDir(false) error: %v", err)
	}
	if got != want {
		t.Errorf("GetRunDir(false) = %q, want %q", got, want)
	}
}

// withRestoredDefaultLogger saves and restores the slog default logger so
// this test's SetLog/SetLogFile calls don't leak into other tests.
func withRestoredDefaultLogger(t *testing.T) {
	t.Helper()
	orig := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(orig)
	})
}

func TestSetLog(t *testing.T) {
	withRestoredDefaultLogger(t)

	var buf bytes.Buffer
	prokishi.SetLog(slog.LevelWarn, &buf)

	slog.Debug("debug message")
	slog.Info("info message")
	if buf.Len() != 0 {
		t.Errorf("expected no output below Warn level, got: %q", buf.String())
	}

	slog.Warn("warn message")
	if !strings.Contains(buf.String(), "warn message") {
		t.Errorf("expected buffer to contain warn message, got: %q", buf.String())
	}

	buf.Reset()
	slog.Error("error message")
	if !strings.Contains(buf.String(), "error message") {
		t.Errorf("expected buffer to contain error message, got: %q", buf.String())
	}
}

func TestSetLogFile(t *testing.T) {
	withRestoredDefaultLogger(t)

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error: %v", err)
	}
	defer func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatalf("failed to restore cwd: %v", err)
		}
	}()

	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("os.Chdir(%q) error: %v", tmp, err)
	}

	closer := prokishi.SetLogFile(slog.LevelInfo, "testlog", true)
	if closer == nil {
		t.Fatal("SetLogFile() returned nil closer")
	}

	slog.Info("hello from test")

	if err := closer.Close(); err != nil {
		t.Fatalf("closer.Close() error: %v", err)
	}

	wantPath := filepath.Join(tmp, "testlog.log")
	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("expected log file at %q, ReadFile error: %v", wantPath, err)
	}
	if !strings.Contains(string(data), "hello from test") {
		t.Errorf("log file content = %q, want it to contain %q", string(data), "hello from test")
	}
}
