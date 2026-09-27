package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"github.com/ShinteLab/prokishi"
	"strings"

	"github.com/BurntSushi/toml"
	"golang.org/x/xerrors"
)

const iniFileName = "prokishi.ini"

// prokishi.ini が無かったので雛形を作ったことを表す
var errIniCreated = errors.New("prokishi.ini が無かったため雛形を作成しました")

// go build -ldflags "-X main.version=?"
var version string

type IniFile struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Code     string `toml:"code"`
	EngineId string `toml:"engineId"`
	Level    string `toml:"logLevel"`
}

var iniFile IniFile

func init() {
}

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		msg := fmt.Sprintf("%+v", err)
		slog.Error(msg)
		fmt.Fprintln(os.Stderr, msg)
		os.Exit(1)
	}
}

func run() error {

	dev := version == ""

	args := flag.Args()
	if len(args) >= 1 {
		sub := args[0]
		if sub == "version" {
			fmt.Println("prokishi version:", version)
			return nil
		}
	}

	err := loadIniFile()
	if err != nil {
		return xerrors.Errorf("loadIniFile() error: %w", err)
	}

	lv := parseLogLevel(iniFile.Level)
	if fp := prokishi.SetLogFile(lv, "prokishi", dev); fp != nil {
		defer fp.Close()
	}

	err = prokishi.Run(iniFile.Host,
		iniFile.Port,
		prokishi.Code(iniFile.Code),
		prokishi.Engine(iniFile.EngineId),
		prokishi.Version(version))
	if err != nil {
		return xerrors.Errorf("prokishi.Run() error: %w", err)
	}
	return nil
}

func parseLogLevel(lv string) slog.Level {
	v := strings.ToLower(lv)
	switch v {
	case "dbg", "debug":
		return slog.LevelDebug
	case "info", "information":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "err", "error":
		return slog.LevelError
	default:
		return slog.LevelWarn
	}
	return slog.LevelInfo
}

func loadIniFile() error {

	//versionに値が入っているかで開発環境を判定
	dir, err := prokishi.GetRunDir(version == "")
	if err != nil {
		return fmt.Errorf("実行位置の取得に失敗しました")
	}

	// 標準入出力は将棋ソフトとの USI のやり取りに使うので、ここで問い合わせない。
	// 雛形だけ作って終了し、engineId などを設定してもらう
	p := filepath.Join(dir, iniFileName)
	if _, err := os.Stat(p); err != nil {
		err := createIniFile(p)
		if err != nil {
			return fmt.Errorf("createIniFile error: %w", err)
		}
		return fmt.Errorf("%w: %s を設定してから起動し直してください", errIniCreated, p)
	}
	_, err = toml.DecodeFile(p, &iniFile)
	if err != nil {
		return fmt.Errorf("%s の解析に失敗しました", p)
	}

	return nil
}

func createIniFile(p string) error {

	fp, err := os.Create(p)
	if err != nil {
		return xerrors.Errorf("os.Create() error: %w", err)
	}
	defer fp.Close()

	var f IniFile
	f.Host = "localhost"
	f.Port = 8080
	f.Code = ""
	f.EngineId = ""
	f.Level = "warn"

	err = toml.NewEncoder(fp).Encode(&f)
	if err != nil {
		return xerrors.Errorf("toml.Encode() error: %w", err)
	}
	return nil
}
