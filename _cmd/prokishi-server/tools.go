//go:build tools

package main

// フロントエンドが使う共有パッケージ @shinte/web（core/web）の版を go.mod で決めるための import。
// アプリのビルドには入らない（ビルドタグ tools）。go mod tidy で require が消えないように置いてある。
// フロントは frontend/scripts/shinte-web.mjs がこのモジュールの web をコピーして使う。
import _ "github.com/ShinteLab/core/web"
