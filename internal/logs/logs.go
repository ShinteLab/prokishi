// Package logs は prokishi のパッケージ（ルート・server・usi）が共有する Logger の置き場。
// 差し替えの口は prokishi.SetLogger（使う側に見せるのはそちらだけ）。
package logs

import (
	"log/slog"
	"sync/atomic"
)

var current atomic.Pointer[slog.Logger]

// Set は Logger を差し替える。nil で slog.Default() に戻る。
func Set(l *slog.Logger) {
	current.Store(l)
}

// L は今の Logger を返す。slog.Default() は呼ぶたびに引く（アプリが後から
// slog.SetDefault したときに追従するため）。
func L() *slog.Logger {
	if l := current.Load(); l != nil {
		return l
	}
	return slog.Default()
}
