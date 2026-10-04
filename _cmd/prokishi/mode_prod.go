//go:build production

package main

// devMode はリリースビルド時 false（-tags production が付く場合）。
// 設定ファイルとログは実行ファイルのディレクトリに置く。
// 開発ビルドでは mode_dev.go が代わりに使われる。
const devMode = false
