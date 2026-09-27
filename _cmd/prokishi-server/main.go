package main

import (
	"embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"github.com/ShinteLab/prokishi"
	"github.com/ShinteLab/prokishi/db"
	"github.com/ShinteLab/prokishi/registry"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"golang.org/x/xerrors"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed version
var versionEmbed string

var version = strings.TrimSpace(versionEmbed)

var (
	port    int
	host    string
	verbose bool
)

func init() {
	application.RegisterEvent[bool]("request-close")
	application.RegisterEvent[registry.Connection]("conn-added")
	application.RegisterEvent[string]("conn-removed")
	application.RegisterEvent[registry.USILogEvent]("usi-log")
	application.RegisterEvent[ServerStateEvent]("server-state")

	flag.IntVar(&port, "p", 8080, "prokishi-server port (initial default)")
	flag.StringVar(&host, "s", "", "prokishi-server host (initial default)")
	flag.BoolVar(&verbose, "v", false, "verbose")
}

var consoleLog = true

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		msg := fmt.Sprintf("run() error:\n%+v", err)
		if !consoleLog {
			slog.Error(msg)
		}
		fmt.Fprintln(os.Stderr, msg)
	}
}

func run() error {
	dev := devMode

	err := db.Init(dev)
	if err != nil {
		if !errors.Is(err, db.AlreadyErr) {
			return xerrors.Errorf("db.Init() error: %w", err)
		}
	}

	db.Open(dev)
	defer db.Close()

	args := flag.Args()
	if len(args) != 0 {
		err := command(args)
		if err != nil {
			return xerrors.Errorf("command() error: %w", err)
		}
		return nil
	}

	lv := slog.LevelInfo
	if verbose {
		lv = slog.LevelDebug
	} else if !dev {
		lv = slog.LevelWarn
	}

	if dev {
		prokishi.SetLog(lv, os.Stdout)
	} else if fp := prokishi.SetLogFile(lv, "prokishi-server", dev); fp != nil {
		consoleLog = false
		defer fp.Close()
	}

	reg := registry.New()

	serverSvc := &ServerService{registry: reg, dev: dev}

	// 初回起動時のみフラグ値をファイルに保存
	if err := serverSvc.InitConfig(host, port); err != nil {
		slog.Warn("InitConfig failed", "err", err)
	}

	return runUI(serverSvc, reg)
}

func runUI(serverSvc *ServerService, reg *registry.Registry) error {
	winSvc := &WindowService{serverService: serverSvc}
	debugSvc := &DebugService{registry: reg}
	systemSvc := &SystemService{registry: reg}

	app := application.New(application.Options{
		Name:        "prokishi-server",
		Description: "Prokishi Server Admin",
		Services: []application.Service{
			application.NewService(&AdminService{}),
			application.NewService(systemSvc),
			application.NewService(winSvc),
			application.NewService(debugSvc),
			application.NewService(serverSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	winSvc.app = app
	serverSvc.app = app

	reg.SetEmit(func(event string, data any) {
		app.Event.Emit(event, data)
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     "prokishi-server",
		Frameless: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(18, 18, 18),
		Width:            900,
		Height:           600,
		MinWidth:         600,
		MinHeight:        400,
		URL:              "/",
	})

	win.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		event.Cancel()
		app.Event.Emit("request-close", true)
	})

	// autoStart の場合はサーバを起動
	cfg, _ := serverSvc.GetConfig()
	if cfg.AutoStart {
		if err := serverSvc.Start(); err != nil {
			slog.Error("autoStart failed", "err", err)
		}
	}

	// Ctrl+C などの OS シグナルでアプリを終了させる（wails3 dev 終了時など）
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit
		serverSvc.Stop()
		app.Quit()
	}()

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
