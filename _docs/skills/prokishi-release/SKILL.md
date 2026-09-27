---
name: prokishi-release
description: prokishi（クライアント）と prokishi-server を配るまでの手順。バージョンの決め方（_cmd/prokishi-server/version と _cmd/version.go）、main へのマージで動く versionup.yml、タグで動く release.yml、下書きリリースの公開、配布対象の OS を増やすとき。「リリースする」「バージョンを上げる」「マイナー / メジャーバージョンを上げる」「タグを打つ」「リリースが作られない」「macOS 版のサーバも配りたい」ときに使う。
---

# リリースする

## バージョンの置き場所

**`_cmd/prokishi-server/version` が唯一の元。** クライアントも同じ番号で配る
（release.yml がこのファイルを読んで `-ldflags "-X main.version=..."` に渡す）。

`_cmd/version.go`（`//go:build ignore` の単独スクリプト）が、次の 3 か所を揃える。

| ファイル | 書き換える箇所 |
|---|---|
| `_cmd/prokishi-server/version` | ファイル全体 |
| `_cmd/prokishi-server/build/config.yml` | `info.version` |
| `_cmd/prokishi-server/frontend/package.json` | `"version"` |

```powershell
go run _cmd/version.go            # 引数なし: version ファイルの値で他の 2 つを揃えるだけ
go run _cmd/version.go 0.2.0      # 指定した番号にする
go run _cmd/version.go -bump      # 対話でパッチ / マイナー / メジャーを選ぶ（人が実行する用）
```

- ⚠️ **エージェントは `-bump` を使わない**（標準入力を待つ）。番号を引数で渡す
- ⚠️ **`version` ファイルだけを手で書き換えない。** `config.yml` と `package.json` がずれる

## 流れ

```
feature ブランチ ──PR──▶ main にマージ
                           │ versionup.yml（PR が merged で閉じたとき）
                           ▼
            タグ v<version> が既にある → パッチ +1 / 無い → そのまま
            version.go で 3 か所を更新、wails3 update build-assets
            chore/version-X ブランチ → PR → 自動マージ → タグ vX を push
                           │ release.yml（v* のタグ push）
                           ▼
            サーバ（Windows）とクライアント（Windows / macOS / Linux）をビルド
            → GitHub Release を**下書き**で作る
```

1. **パッチを上げるだけなら何もしなくてよい。** 前回のタグがあれば versionup が +1 する
2. **マイナー / メジャーを上げるなら**、その PR の中で `go run _cmd/version.go 0.2.0` を実行してコミットしておく。
   タグ `v0.2.0` がまだ無いので、versionup はその番号をそのまま使う
3. PR を main にマージする
4. Actions で VersionUp → Release が通ったのを確かめる
5. GitHub の Releases で下書きを確かめて**手で公開する**（`draft: true` なので自動では公開されない）

## 注意

- ⚠️ **versionup は main への直接 push では動かない。** トリガーは「main 向けの PR がマージされて閉じた」とき
- タイトルが `chore: version` で始まる PR では versionup を動かさない（自分の PR で無限に回らないように）
- `GITHUB_TOKEN` で push したタグは後続のワークフローを起動しないため、versionup は
  `MY_GITHUB_TOKEN`（PAT）を使う。**PAT の期限が切れると versionup が checkout / push で落ちる**
- `-` や `+` を含むタグ（`v0.2.0-rc1` など）では release.yml はビルドしない
- CI の `wails3` は `@latest` を入れる。`go.mod` の Wails（beta.3）より新しい CLI で
  `update build-assets` やビルドをするので、ビルドの失敗や build/ 配下の想定外の差分はまずここを疑う
- CI の `setup-go` は `1.25`、`go.mod` は `go 1.26.1`。ツールチェーンの自動取得で動いている

## ローカルで先に確かめる

```powershell
go test ./...
go build -ldflags "-X main.version=0.0.0" -o prokishi.exe ./_cmd/prokishi/

cd _cmd/prokishi-server
go test ./...
cd frontend; npm install; npm test; cd ..
wails3 build
```

## 配布対象の OS を増やす

`.github/workflows/release.yml` の `build-server` の `matrix` に、コメントアウトしてある
`ubuntu-latest` / `macos-latest` がある。外せば対象になる。

- macOS は `wails3 package` で `.app` を作る分岐がすでにある。ただし**署名・公証はしていない**ので、
  配った `.app` は Gatekeeper に止められる（利用者側で許可が要る）
- Linux は GTK4 / WebKitGTK 6 の開発パッケージを入れる手順がすでにある
- クライアントは 3 OS ともすでに配っている
