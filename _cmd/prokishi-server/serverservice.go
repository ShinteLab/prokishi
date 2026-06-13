package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"prokishi"
	"prokishi/registry"
	"prokishi/server"
	"strconv"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type ServerConfig struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	AutoStart bool   `json:"autoStart"`
}

type ServerStateEvent struct {
	Running bool   `json:"running"`
	URL     string `json:"url"`
}

func defaultServerConfig() ServerConfig {
	return ServerConfig{Host: "", Port: 8080, AutoStart: true}
}

type ServerService struct {
	app      *application.App
	registry *registry.Registry
	dev      bool

	mu     sync.Mutex
	cancel context.CancelFunc
	url    string
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
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	displayHost := cfg.Host
	if displayHost == "" {
		displayHost = "0.0.0.0"
	}
	url := displayHost + ":" + strconv.Itoa(cfg.Port)
	s.url = url
	s.mu.Unlock()

	s.emit(true, url)

	go func() {
		if err := server.Run(ctx, cfg.Host, cfg.Port, server.WithRegistry(s.registry)); err != nil {
			slog.Error("server.Run error", "err", err)
		}
		s.mu.Lock()
		s.cancel = nil
		s.url = ""
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

func (s *ServerService) emit(running bool, url string) {
	if s.app != nil {
		s.app.Event.Emit("server-state", ServerStateEvent{Running: running, URL: url})
	}
}

// GetState は現在の状態をまとめて返す（フロントエンド初期化用）。
func (s *ServerService) GetState() ServerStateEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ServerStateEvent{Running: s.cancel != nil, URL: s.url}
}
