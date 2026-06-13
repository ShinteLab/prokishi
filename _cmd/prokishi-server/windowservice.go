package main

import "github.com/wailsapp/wails/v3/pkg/application"

type WindowService struct {
	app           *application.App
	serverService *ServerService
}

func (s *WindowService) ShutdownAndQuit() {
	s.serverService.Stop()
	s.app.Quit()
}
