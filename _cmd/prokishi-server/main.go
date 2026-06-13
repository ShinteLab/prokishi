package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"prokishi"
	"prokishi/db"
	"prokishi/registry"
	"prokishi/server"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"golang.org/x/xerrors"
)

//go:embed all:frontend/dist
var assets embed.FS

var version string

var (
	port    int
	host    string
	verbose bool
)

func init() {
	application.RegisterEvent[string]("time")
	application.RegisterEvent[bool]("request-close")
	application.RegisterEvent[registry.Connection]("conn-added")
	application.RegisterEvent[string]("conn-removed")
	application.RegisterEvent[registry.USILogEvent]("usi-log")

	flag.IntVar(&port, "p", 8080, "prokishi-server port")
	flag.StringVar(&host, "s", "", "prokishi-server name(default empty)")
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
		fmt.Fprintf(os.Stderr, msg+"\n")
	}
}

func run() error {
	dev := version == ""

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
		defer prokishi.SetLog(lv, os.Stdout)
	} else {
		consoleLog = false
		defer prokishi.SetLogFile(lv, "prokishi-server", dev).Close()
	}

	reg := registry.New()

	ctx, cancelServer := context.WithCancel(context.Background())

	go func() {
		if err := server.Run(ctx, host, port, server.WithRegistry(reg)); err != nil {
			slog.Error("server.Run() error", "error", err)
		}
	}()

	return runUI(cancelServer, reg)
}

func runUI(cancelServer context.CancelFunc, reg *registry.Registry) error {
	winSvc := &WindowService{cancelServer: cancelServer}
	debugSvc := &DebugService{registry: reg}

	app := application.New(application.Options{
		Name:        "prokishi-server",
		Description: "Prokishi Server Admin",
		Services: []application.Service{
			application.NewService(&GreetService{}),
			application.NewService(&AdminService{}),
			application.NewService(winSvc),
			application.NewService(debugSvc),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	winSvc.app = app

	// アプリ起動後にレジストリのイベント emit を有効化
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

	go func() {
		for {
			now := time.Now().Format(time.RFC1123)
			app.Event.Emit("time", now)
			time.Sleep(time.Second)
		}
	}()

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
	return nil
}
