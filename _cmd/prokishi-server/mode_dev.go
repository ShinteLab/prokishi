//go:build !production

package main

// devMode は開発ビルド時 true。wails3 dev / go build（タグなし）で有効。
// リリースビルド（-tags production）では mode_prod.go が代わりに使われる。
const devMode = true
