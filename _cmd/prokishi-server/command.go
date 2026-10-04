package main

import (
	"fmt"
)

// エンジンと認証コードの登録は管理画面で行う（リリース版は GUI アプリとして
// ビルドされ、コマンドの出力が表示されないため、コマンド形式の登録は持たない）
func command(args []string) error {

	sub := args[0]

	switch sub {
	case "version":
		fmt.Println("prokishi-server version:", version)
	default:
		return fmt.Errorf("unknow sub command: %s", sub)
	}
	return nil
}
