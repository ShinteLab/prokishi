package prokishi

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ShinteLab/prokishi/internal/logs"
	"golang.org/x/xerrors"
)

// 開発時は作業ディレクトリ、
// リリース動作の場合は実行しているパスを取得
func GetRunDir(d bool) (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", xerrors.Errorf("os.Getwd() error: %w", err)
	}
	//開発モードじゃない場合、実行位置に変更
	if !d {
		exe, err := os.Executable()
		if err != nil {
			return "", xerrors.Errorf("os.Executable() error: %w", err)
		}
		dir = filepath.Dir(exe)
	}
	return dir, nil
}

// ログ。prokishi はライブラリなので、ログの設定（レベル・出力先）は持たず、
// slog.SetDefault も呼ばない。使う側（アプリ）が決める。
//
// ⚠️ クライアント（Run）は USI のやり取りに標準入出力を使う。**標準出力へ書く Logger を
// 渡さないこと**（GUI が USI の応答として読んでしまう）。

// SetLogger は prokishi（server・usi を含む）が使う Logger を差し替える。
// nil を渡すと slog.Default() に戻る（何もしなければ slog.Default()）。
func SetLogger(l *slog.Logger) {
	logs.Set(l)
}

// NewFileLogger は実行位置（GetRunDir）に "<name>.log" を作り、lv 以上を書く Logger を返す。
// ファイルは起動のたびに作り直す（削除の運用をしなくて済むよう、同じ名前にしている）。
//
// ⚠️ **slog.SetDefault はしない。** 既定にするか SetLogger に渡すかはアプリが決める。
// 返す io.Closer で終了時にファイルを閉じる。
func NewFileLogger(lv slog.Level, name string, dev bool) (*slog.Logger, io.Closer, error) {
	dir, err := GetRunDir(dev)
	if err != nil {
		return nil, nil, xerrors.Errorf("GetRunDir() error: %w", err)
	}
	fp, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s.log", name)))
	if err != nil {
		return nil, nil, xerrors.Errorf("os.Create() error: %w", err)
	}
	h := slog.NewTextHandler(fp, &slog.HandlerOptions{Level: lv})
	return slog.New(h), fp, nil
}
