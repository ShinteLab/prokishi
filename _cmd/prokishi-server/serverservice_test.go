package main

import (
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ShinteLab/prokishi/registry"
)

func TestServerService_GetConfig_MissingFileReturnsDefault(t *testing.T) {
	chdirTemp(t)
	s := &ServerService{dev: true}

	got, err := s.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error: %v", err)
	}
	want := defaultServerConfig()
	if got != want {
		t.Errorf("GetConfig() = %+v, want default %+v", got, want)
	}
}

func TestServerService_SaveConfig_GetConfig_Roundtrip(t *testing.T) {
	chdirTemp(t)
	s := &ServerService{dev: true}

	want := ServerConfig{Host: "example.com", Port: 9999, AutoStart: false}
	if err := s.SaveConfig(want); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	got, err := s.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error: %v", err)
	}
	if got != want {
		t.Errorf("GetConfig() = %+v, want %+v", got, want)
	}
}

func TestServerService_InitConfig_OnlyWritesOnce(t *testing.T) {
	chdirTemp(t)
	s := &ServerService{dev: true}

	if err := s.InitConfig("first-host", 1111); err != nil {
		t.Fatalf("InitConfig() error: %v", err)
	}
	first, err := s.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error: %v", err)
	}
	wantFirst := ServerConfig{Host: "first-host", Port: 1111, AutoStart: true}
	if first != wantFirst {
		t.Fatalf("GetConfig() after first InitConfig = %+v, want %+v", first, wantFirst)
	}

	if err := s.InitConfig("second-host", 2222); err != nil {
		t.Fatalf("InitConfig() (second call) error: %v", err)
	}
	second, err := s.GetConfig()
	if err != nil {
		t.Fatalf("GetConfig() error: %v", err)
	}
	if second != first {
		t.Errorf("InitConfig() overwrote an existing config: got %+v, want unchanged %+v", second, first)
	}
}

func TestServerService_ZeroValue_NoAppNoServer(t *testing.T) {
	s := &ServerService{}

	if s.IsRunning() {
		t.Error("IsRunning() = true on zero-value ServerService, want false")
	}
	if url := s.GetURL(); url != "" {
		t.Errorf("GetURL() = %q, want empty", url)
	}
	state := s.GetState()
	if state.Running || state.URL != "" {
		t.Errorf("GetState() = %+v, want {Running:false URL:\"\"}", state)
	}

	// Stop() on a service that was never started must be a safe no-op
	// (s.cancel is nil) - it must not panic.
	s.Stop()
}

func TestServerService_GetLocalIPs_Smoke(t *testing.T) {
	s := &ServerService{}

	// Must not panic; environment-dependent contents aren't asserted.
	_ = s.GetLocalIPs()
}

// 他プロセスが同じポートを掴んでいるとき、Start() は非同期に流さず
// その場で bind エラーを返し、稼働中扱いにしないこと。
func TestServerService_Start_PortInUseReturnsError(t *testing.T) {
	chdirTemp(t)

	// 先客を作る。ephemeral port を取ってから同じポートを狙う。
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error: %v", err)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port

	s := &ServerService{registry: registry.New(), dev: true}
	if err := s.SaveConfig(ServerConfig{Host: "127.0.0.1", Port: port, AutoStart: false}); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	err = s.Start()
	if err == nil {
		t.Cleanup(s.Stop)
		t.Fatal("Start() on an occupied port: got nil error, want bind error")
	}
	if !strings.Contains(err.Error(), strconv.Itoa(port)) {
		t.Errorf("Start() error = %q, want it to mention port %d", err, port)
	}

	if s.IsRunning() {
		t.Error("IsRunning() = true after a failed Start(), want false")
	}
	if st := s.GetState(); st.Running || st.Err == "" {
		t.Errorf("GetState() = %+v, want Running=false with a non-empty Err", st)
	}
}

func TestServerService_StartStopLifecycle(t *testing.T) {
	chdirTemp(t)
	s := &ServerService{registry: registry.New(), dev: true}

	// Port 0 asks the OS for an ephemeral free port so this test can't
	// collide with a real prokishi-server instance or other test runs.
	if err := s.SaveConfig(ServerConfig{Host: "127.0.0.1", Port: 0, AutoStart: false}); err != nil {
		t.Fatalf("SaveConfig() error: %v", err)
	}

	if err := s.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	// Belt-and-braces cleanup in case an assertion below fails before the
	// explicit Stop() call - Stop() is a safe no-op if already stopped.
	t.Cleanup(s.Stop)

	if !s.IsRunning() {
		t.Error("IsRunning() = false immediately after Start(), want true")
	}

	if err := s.Start(); err == nil {
		t.Error("Start() while already running: got nil error, want error")
	}

	s.Stop()

	deadline := time.Now().Add(5 * time.Second)
	for s.IsRunning() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if s.IsRunning() {
		t.Fatal("IsRunning() still true after Stop() and 5s timeout - server goroutine may have leaked")
	}
	if url := s.GetURL(); url != "" {
		t.Errorf("GetURL() after Stop() = %q, want empty", url)
	}
}
