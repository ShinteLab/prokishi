package main

import (
	"context"
	"fmt"
	"os"
	"prokishi/db"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
	"golang.org/x/xerrors"
)

type EngineItem struct {
	ID      string `json:"id"`
	Path    string `json:"path"`
	Created string `json:"created"`
}

type CodeItem struct {
	Code    string `json:"code"`
	Created string `json:"created"`
	Used    string `json:"used"`
}

type AdminService struct{}

func (a *AdminService) ListEngines() ([]EngineItem, error) {
	engines, err := db.FindEngines(context.Background())
	if err != nil {
		return nil, xerrors.Errorf("db.FindEngines() error: %w", err)
	}
	items := make([]EngineItem, 0, len(engines))
	for _, e := range engines {
		items = append(items, EngineItem{
			ID:      e.ID,
			Path:    e.Path,
			Created: e.Created.Format("2006-01-02 15:04:05"),
		})
	}
	return items, nil
}

func (a *AdminService) RegisterEngine(path string) (string, error) {
	id := uuid.New().String()
	if err := db.InsertEngine(id, path); err != nil {
		return "", xerrors.Errorf("db.InsertEngine() error: %w", err)
	}
	return id, nil
}

func (a *AdminService) DeleteEngine(id string) error {
	return db.DeleteEngine(id)
}

func (a *AdminService) ListCodes() ([]CodeItem, error) {
	codes, err := db.FindCodes(context.Background())
	if err != nil {
		return nil, xerrors.Errorf("db.FindCodes() error: %w", err)
	}
	items := make([]CodeItem, 0, len(codes))
	for _, c := range codes {
		used := ""
		if !c.Updated.IsZero() {
			used = c.Updated.Format("2006-01-02 15:04:05")
		}
		items = append(items, CodeItem{
			Code:    c.Code,
			Created: c.Created.Format("2006-01-02 15:04:05"),
			Used:    used,
		})
	}
	return items, nil
}

func (a *AdminService) GenerateCode() (string, error) {
	code := uuid.New().String()
	if err := db.InsertCode(code); err != nil {
		return "", xerrors.Errorf("db.InsertCode() error: %w", err)
	}
	return code, nil
}

func (a *AdminService) RegisterCode(code string) error {
	return db.InsertCode(code)
}

func (a *AdminService) DeleteCode(code string) error {
	return db.DeleteCode(code)
}

// SaveClientConfig はクライアント設定ファイル (prokishi.ini) をファイル保存ダイアログで書き出す。
func (a *AdminService) SaveClientConfig(host string, port int, code string, engineId string, logLevel string) error {
	content := fmt.Sprintf("host = %q\nport = %d\ncode = %q\nengineId = %q\nlogLevel = %q\n",
		host, port, code, engineId, logLevel)

	path, err := application.Get().Dialog.SaveFile().
		SetMessage("クライアント設定ファイルの保存先を選択してください").
		SetFilename("prokishi.ini").
		AddFilter("INI ファイル (*.ini)", "*.ini").
		AddFilter("すべてのファイル", "*.*").
		PromptForSingleSelection()
	if err != nil {
		return xerrors.Errorf("SaveFileDialog error: %w", err)
	}
	if path == "" {
		return nil // キャンセル
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return xerrors.Errorf("WriteFile error: %w", err)
	}
	return nil
}

func (a *AdminService) SelectEnginePath() (string, error) {
	path, err := application.Get().Dialog.OpenFile().
		SetTitle("エンジンを選択").
		AddFilter("実行ファイル (*.exe)", "*.exe").
		AddFilter("すべてのファイル", "*.*").
		PromptForSingleSelection()
	if err != nil {
		return "", xerrors.Errorf("OpenFileDialog error: %w", err)
	}
	return path, nil
}
