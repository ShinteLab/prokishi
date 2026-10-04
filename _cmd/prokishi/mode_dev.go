//go:build !production

package main

// devMode は開発ビルド時 true。go build / go test（タグなし）で有効。
// 設定ファイルとログはカレントディレクトリに置く。
// リリースビルド（-tags production）では mode_prod.go が代わりに使われる。
const devMode = true
