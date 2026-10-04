package main

import (
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ShinteLab/prokishi"
	"golang.org/x/xerrors"
)

const iniFileName = "prokishi.ini"

// prokishi.ini が無かったので雛形を作ったことを表す
var errIniCreated = errors.New("prokishi.ini が無かったため雛形を作成しました")

// バージョンは _cmd/prokishi-server/version がマスタで、_cmd/version.go が同じ値をここへ書く
//
//go:embed version
var versionEmbed string
var version = strings.TrimSpace(versionEmbed)

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

// logFile は run が開いたログファイル。**main がエラーを書いてから閉じる**
// （run の中で閉じると、終了の理由がログに残らない）。
var logFile io.Closer

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		msg := fmt.Sprintf("%+v", err)
		slog.Error(msg)
		fmt.Fprintln(os.Stderr, msg)
		closeLog()
		os.Exit(1)
	}
	closeLog()
}

func closeLog() {
	if logFile != nil {
		logFile.Close()
	}
}

func run() error {

	dev := devMode

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

	// ログは実行位置の prokishi.log へ。⚠️ 標準出力は USI のやり取りなので、ログを向けない
	// （ファイルを作れなければ slog の既定のまま＝標準エラー）。
	lv, ok := parseLogLevel(iniFile.Level)
	if logger, fp, err := prokishi.NewFileLogger(lv, "prokishi", dev); err != nil {
		slog.Error("ログファイルを作れません", "err", err)
	} else {
		slog.SetDefault(logger)
		logFile = fp
	}
	if !ok {
		slog.Warn("logLevel を読めないので warn で動かします", "logLevel", iniFile.Level)
	}

	// 開発ビルドでは id name に "Development" と出す（従来どおり）
	v := version
	if dev {
		v = ""
	}

	err = prokishi.Run(iniFile.Host,
		iniFile.Port,
		prokishi.Code(iniFile.Code),
		prokishi.Engine(iniFile.EngineId),
		prokishi.Version(v))
	if err != nil {
		return xerrors.Errorf("prokishi.Run() error: %w", err)
	}
	return nil
}

// parseLogLevel は prokishi.ini の logLevel を読む。読めない値は Warn にし、ok を false で返す
// （黙って Warn にすると、綴りを間違えたことに気づけない）。空は既定の Warn として ok。
func parseLogLevel(lv string) (slog.Level, bool) {
	v := strings.ToLower(lv)
	switch v {
	case "dbg", "debug":
		return slog.LevelDebug, true
	case "info", "information":
		return slog.LevelInfo, true
	case "warn", "warning", "":
		return slog.LevelWarn, true
	case "err", "error":
		return slog.LevelError, true
	default:
		return slog.LevelWarn, false
	}
}

func loadIniFile() error {

	dir, err := prokishi.GetRunDir(devMode)
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
