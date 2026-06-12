package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"prokishi"
	"prokishi/db"
	"prokishi/server"

	"golang.org/x/xerrors"
)

var version string

var (
	port    int
	host    string
	verbose bool
)

func init() {
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
		//slogがコンソール時はいらない,設定がまだの場合はいらない
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

	//DB操作モードかを判定
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
		//レベルを確認
		defer prokishi.SetLogFile(lv, "prokishi-server", dev).Close()
	}

	err = server.Run(host, port)
	if err != nil {
		return xerrors.Errorf("server.Run() error: %w", err)
	}
	return nil
}

func command(args []string) error {

	var err error
	sub := args[0]

	switch sub {
	case "version":
		fmt.Println("prokishi-server version:", version)
	case "engine":
		if len(args) >= 2 {
			err = commandEngine(args[1:])
		} else {
			err = server.PrintEngineIds()
		}
	case "code":
		if len(args) >= 2 {
			err = commandCode(args[1:])
		} else {
			err = server.PrintCodes()
		}
	default:
		return fmt.Errorf("unknow sub command: %s", sub)
	}

	if err != nil {
		return xerrors.Errorf("Sub Command[%s] error: %w", sub, err)
	}
	return nil
}

func commandEngine(args []string) error {

	var err error
	mode := args[0]

	switch mode {
	case "generate":
		if len(args) == 2 {
			p := args[1]
			err = server.GenerateEngineId(p)
		} else if len(args) > 2 {
			err = fmt.Errorf("Please separate the engine paths with double quotes.")
		} else {
			err = fmt.Errorf("engine generate mode required path")
		}
	case "register":
		if len(args) == 3 {
			id := args[1]
			p := args[2]
			err = server.RegisterEngineId(id, p)
		} else if len(args) > 3 {
			err = fmt.Errorf("Please separate the engine paths with double quotes.")
		} else {
			err = fmt.Errorf("engine register mode required id,path")
		}
	case "delete":
		if len(args) >= 2 {
			id := args[1]
			err = server.DeleteEngineId(id)
		} else {
			err = fmt.Errorf("engine delete mode required id")
		}
	default:
		return fmt.Errorf("engine command unknown mode: %s", mode)
	}

	if err != nil {
		return xerrors.Errorf("engine mode[%s] error: %w", mode, err)
	}

	return nil
}

func commandCode(args []string) error {
	var err error
	mode := args[0]

	switch mode {
	case "generate":
		err = server.GenerateCode()
	case "register":
		if len(args) >= 2 {
			code := args[1]
			err = server.RegisterCode(code)
		} else {
			err = fmt.Errorf("code register mode required code")
		}
	case "delete":
		if len(args) >= 2 {
			code := args[1]
			err = server.DeleteCode(code)
		} else {
			err = fmt.Errorf("code delete mode required code")
		}
	default:
		return fmt.Errorf("code command unknown mode: %s", mode)
	}

	if err != nil {
		return xerrors.Errorf("code mode[%s] error: %w", mode, err)
	}
	return nil
}
