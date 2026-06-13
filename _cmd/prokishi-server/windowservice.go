package main

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type WindowService struct {
	app          *application.App
	cancelServer context.CancelFunc
}

// ShutdownAndQuit はフロントエンドの確認ダイアログで承認された後に呼ばれる。
// サーバを停止してアプリケーションを終了する。
func (s *WindowService) ShutdownAndQuit() {
	s.cancelServer()
	s.app.Quit()
}
