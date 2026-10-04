package prokishi_test

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ShinteLab/prokishi"
	"github.com/ShinteLab/prokishi/internal/logs"
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

// NewFileLogger はファイルに書く Logger を返すだけで、slog.Default() は変えないこと。
// （ライブラリから既定の Logger を書き換えない約束の歯止め）
func TestNewFileLogger(t *testing.T) {
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

	before := slog.Default()
	logger, closer, err := prokishi.NewFileLogger(slog.LevelInfo, "testlog", true)
	if err != nil {
		t.Fatalf("NewFileLogger() error: %v", err)
	}
	if slog.Default() != before {
		t.Error("NewFileLogger() must not change slog.Default()")
	}

	logger.Debug("debug message")
	logger.Info("hello from test")
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
	if strings.Contains(string(data), "debug message") {
		t.Errorf("log file should not contain messages below Info: %q", string(data))
	}
}

// SetLogger で渡した Logger に出し、nil で slog.Default() に戻ること。
func TestSetLogger(t *testing.T) {
	var buf bytes.Buffer
	prokishi.SetLogger(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { prokishi.SetLogger(nil) })

	logs.L().Info("from library")
	if !strings.Contains(buf.String(), "from library") {
		t.Errorf("SetLogger should replace the logger: %q", buf.String())
	}
	prokishi.SetLogger(nil)
	if got := logs.L(); got != slog.Default() {
		t.Error("SetLogger(nil) should fall back to slog.Default()")
	}
}
