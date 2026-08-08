package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"github.com/ShinteLab/prokishi"
	"github.com/ShinteLab/prokishi/registry"
	"github.com/ShinteLab/prokishi/server"
	"strconv"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type ServerConfig struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	AutoStart bool   `json:"autoStart"`
	UseAuth   bool   `json:"useAuth"`
}

type ServerStateEvent struct {
	Running bool   `json:"running"`
	URL     string `json:"url"`
	// Err は直近の起動／稼働失敗の理由。成功時は空。
	// autoStart のようにフロントが Start() の戻り値を見られない
	// 経路でも失敗を画面に出せるようにするためのもの。
	Err string `json:"err"`
}

func defaultServerConfig() ServerConfig {
	return ServerConfig{Host: "", Port: 8080, AutoStart: true}
}

type ServerService struct {
	app      *application.App
	registry *registry.Registry
	dev      bool

	mu      sync.Mutex
	cancel  context.CancelFunc
	url     string
	lastErr string
}

func (s *ServerService) configPath() (string, error) {
	dir, err := prokishi.GetRunDir(s.dev)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "server-config.json"), nil
}

// GetConfig はファイルから設定を読む。ファイルがなければデフォルト値を返す。
func (s *ServerService) GetConfig() (ServerConfig, error) {
	path, err := s.configPath()
	if err != nil {
		return defaultServerConfig(), err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return defaultServerConfig(), nil
	}
	if err != nil {
		return defaultServerConfig(), err
	}
	var cfg ServerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultServerConfig(), err
	}
	return cfg, nil
}

// SaveConfig は設定をファイルに書く。
func (s *ServerService) SaveConfig(cfg ServerConfig) error {
	path, err := s.configPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// InitConfig はファイルが存在しない場合だけ初期値を書く（初回起動時のフラグ移行）。
func (s *ServerService) InitConfig(host string, port int) error {
	path, err := s.configPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return s.SaveConfig(ServerConfig{Host: host, Port: port, AutoStart: true})
	}
	return nil
}

func (s *ServerService) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancel != nil
}

func (s *ServerService) GetURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.url
}

func (s *ServerService) Start() error {
	cfg, err := s.GetConfig()
	if err != nil {
		return err
	}

	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return fmt.Errorf("already running")
	}
	s.mu.Unlock()

	// bind は同期的に行い、ポート使用中などのエラーを呼び出し元
	// （＝UI の起動ボタン）へそのまま返す。
	listener, err := server.Listen(cfg.Host, cfg.Port)
	if err != nil {
		slog.Error("server listen error", "host", cfg.Host, "port", cfg.Port, "err", err)
		// xerrors のスタックフレームは UI に出しても読めないので根本原因だけ使う。
		startErr := fmt.Errorf("ポート %d を開けませんでした: %s", cfg.Port, rootCause(err))
		// emit はしない。Start() の戻り値で呼び出し元に伝わるため、
		// イベント経由で通知すると UI が二重にエラーを出す。
		// autoStart 経路はフロント初期化時の GetState() で lastErr を拾う。
		s.setLastErr(startErr.Error())
		return startErr
	}
	s.setLastErr("")

	s.mu.Lock()
	if s.cancel != nil {
		// Start() が同時に呼ばれた場合。取得済みのリスナは捨てる。
		s.mu.Unlock()
		listener.Close()
		return fmt.Errorf("already running")
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	displayHost := cfg.Host
	if displayHost == "" {
		displayHost = "0.0.0.0"
	}
	// Port 0 指定時に OS が割り当てた実ポートを表示するため、
	// 設定値ではなくリスナの実アドレスからポートを取る。
	actualPort := cfg.Port
	if tcpAddr, ok := listener.Addr().(*net.TCPAddr); ok {
		actualPort = tcpAddr.Port
	}
	url := displayHost + ":" + strconv.Itoa(actualPort)
	s.url = url
	s.mu.Unlock()

	s.emit(true, url)

	go func() {
		serveErr := server.Serve(ctx, listener, server.WithRegistry(s.registry), server.WithAuth(cfg.UseAuth))
		if serveErr != nil {
			slog.Error("server.Serve error", "err", serveErr)
		}
		s.mu.Lock()
		s.cancel = nil
		s.url = ""
		if serveErr != nil {
			s.lastErr = "サーバが停止しました: " + rootCause(serveErr).Error()
		}
		s.mu.Unlock()
		s.emit(false, "")
	}()

	return nil
}

func (s *ServerService) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// rootCause はラップの一番内側のエラーを返す。
func rootCause(err error) error {
	for {
		next := errors.Unwrap(err)
		if next == nil {
			return err
		}
		err = next
	}
}

func (s *ServerService) setLastErr(msg string) {
	s.mu.Lock()
	s.lastErr = msg
	s.mu.Unlock()
}

func (s *ServerService) emit(running bool, url string) {
	if s.app == nil {
		return
	}
	s.mu.Lock()
	lastErr := s.lastErr
	s.mu.Unlock()
	s.app.Event.Emit("server-state", ServerStateEvent{Running: running, URL: url, Err: lastErr})
}

// GetLocalIPs はループバックを除く IPv4 アドレス一覧を返す。
func (s *ServerService) GetLocalIPs() []string {
	var result []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return result
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if ip.To4() != nil {
				result = append(result, ip.String())
			}
		}
	}
	return result
}

// GetState は現在の状態をまとめて返す（フロントエンド初期化用）。
func (s *ServerService) GetState() ServerStateEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ServerStateEvent{Running: s.cancel != nil, URL: s.url, Err: s.lastErr}
}
