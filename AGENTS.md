# AGENTS.md

このリポジトリで作業するコーディングエージェント向けの指示。
リポジトリ全体の方針はルート（shinte）の `AGENTS.md` を参照。

**このファイルには概要と制約だけを置く。** 作業の手順は `_docs/skills/` にある。

| 置き場所 | 中身 | 読むとき |
|---|---|---|
| スキル `prokishi-change-api`（`_docs/skills/`） | gRPC の API を変える手順（再生成・サーバとクライアントの修正・新旧の互換性） | `api/api.proto` を触るとき |
| スキル `prokishi-release`（`_docs/skills/`） | バージョンの上げ方と versionup → タグ → release の流れ | リリース・バージョンを上げるとき |

## 概要

Prokishi は Go で書いた USI（Universal Shogi Interface）プロトコルのプロキシ。将棋エンジンを
軽量なクライアント（`prokishi`）とリモートのサーバ（`prokishi-server`）に分け、gRPC でつなぐ。
クライアントは将棋 GUI とサーバの間に入り、USI コマンドをネットワーク越しに転送するので、
重いエンジンはリモートで動く。

想定する構成: サーバはエンジンのバイナリ（Apery、やねうら王など）がある Windows 機で動かし、
クライアントは macOS の ShogiHome（Electron 製のクロスプラットフォーム GUI）から使う。
Windows の ShogiGUI でも手動で動作確認する。

## モジュール構成

このディレクトリは単独の Go モジュール `github.com/ShinteLab/prokishi`。`core` への Go の依存は無い
（フロントだけが `@shinte/web` を相対パスで使う）。`_cmd/prokishi-server/` の Wails3 アプリは
入れ子の別モジュール（`prokishi-server`）で、`replace github.com/ShinteLab/prokishi => ../../` で
このモジュールを取り込んでいる。タグはまだ打っていないので、この replace は外さないこと。
`go` コマンドは基本このディレクトリで実行する（サーバアプリ本体は `_cmd/prokishi-server/` で実行）。

## ビルド

### prokishi クライアント（素の Go CLI）

```powershell
go build -tags production -o prokishi.exe ./_cmd/prokishi/   # リリース用（タグなしは開発用）
```

### prokishi-server（Wails3 デスクトップアプリ）

```powershell
# 開発
cd _cmd/prokishi-server
wails3 dev

# リリースビルド（内部で -tags production を使う）
cd _cmd/prokishi-server
wails3 build
```

### 開発モードとリリースモード

設定・DB・ログの置き場所がモードで変わる（解決は `prokishi.GetRunDir(dev)`、`logger.go`）。

サーバもクライアントもビルドタグで切り替える。`!production` → `mode_dev.go`（カレントディレクトリ）、
`production` → `mode_prod.go`（実行ファイルのディレクトリ）。Wails3 の Taskfile がリリースビルドで
`-tags production` を使うのに合わせていて、クライアントは release.yml が `-tags production` を付ける。
開発用のクライアントは `id name` に付けるバージョンを `Development` にする。

バージョン番号は各アプリの `version` ファイルを `//go:embed` する（`_cmd/prokishi-server/version` がマスタ）。
`prokishi version` / `prokishi-server version` で表示できる。扱いは `prokishi-release` を参照。

## テスト

```powershell
go test ./...
go test ./_cmd/prokishi/      # _cmd 配下は ./... に含まれない

# 結合テスト（ビルドタグ integration）
go test -tags integration ./usi/...

# サーバアプリ（入れ子モジュール）
cd _cmd/prokishi-server
go test ./...

# フロントエンド（vitest）
cd _cmd/prokishi-server/frontend
npm test
```

エンジンを起動するテストは本物のエンジンを使わず、`internal/testfakeengine` が
テスト時にビルドする偽エンジン（stdin を返すだけのプログラム）を使う。

## API とリリース

- `api/api.proto` を変えるとき → `prokishi-change-api`。⚠️ 生成物（`api/*.pb.go`）を手で編集しないこと。
  クライアントとサーバは別々に更新されるので、既存フィールドの番号・型・意味を変えないこと
- リリース → `prokishi-release`。`versionup.yml`（main 向け PR のマージ）→ タグ → `release.yml`（下書きの Release）

## アーキテクチャ

### データの流れ

```
[将棋 GUI（ShogiHome / ShogiGUI）]
    ↕ stdin/stdout（USI のテキストプロトコル）
[prokishi クライアント]
    ↕ gRPC（TCP）
[prokishi-server]
    ↕ stdin/stdout（エンジンを子プロセスとして起動）
[エンジンのバイナリ（Apery、やねうら王など）]
```

### 主なパッケージ

| パッケージ | 役割 |
|---|---|
| `prokishi`（ルート） | クライアントの流れ: 設定を読み、サーバに接続し、stdin と gRPC を橋渡しする |
| `usi/` | OS プロセスの I/O の抽象化 — `Receiver` は stdin を読み、`Sender` はエンジンの子プロセスを包む |
| `api/` | 3 つの gRPC サービス（`ConnectionService`、`USISendService`、`USIReceiveService`）の Protobuf 定義 |
| `server/` | gRPC サーバ — 接続を認証し、エンジンのプロセスを起動し、出力をストリームで返す |
| `db/` | CSVQ（CSV ファイルに SQL を投げる）による永続化。エンジンの登録と認証コード |
| `registry/` | メモリ上のエンジン登録簿。接続の UUID → エンジンの `Sender` |
| `internal/testfakeengine` | テスト専用。偽エンジンをビルドする（本物のエンジン無しでプロセス起動を試す） |
| `_cmd/prokishi/` | クライアントのエントリポイント（素の Go CLI） |
| `_cmd/prokishi-server/` | サーバのエントリポイント（管理 UI 付きの Wails3 デスクトップアプリ） |

### サーバの管理 UI（Wails3）

Wails3 + React + MUI で作った GUI。タイトルバーのメニューで画面を切り替える（`TitleBar.tsx`。起動時はモニター）。

- **モニター**（`MonitorView.tsx`） — 接続と USI のやり取りを実時間で監視する。
- **盤面** — 将棋盤の表示（`BoardView.tsx`）。`@shinte/web`（リポジトリ `core/web`）の
  `<shogi-board>` / `<shogi-hand>` を使う。SFEN を描画するほか、接続の USI ログから最新の
  `position` コマンドを取り込める。`resolvePosition` が `moves`（駒取り・成り・打ち）を適用して
  現在の盤・持ち駒・手番を表示する（手の適用は表示用で、合法性は検査しない）。
- **マスタ管理**（`MasterView.tsx`） — 画面内のタブが 3 つある。
  - サーバ設定: ホスト・ポート・自動起動・認証の有無、起動・停止、クライアント設定（`prokishi.ini`）の書き出し。
    設定は `server-config.json` に保存する。⚠️ 起動引数の `-p` / `-s` はこのファイルが無いとき（初回）しか効かない
  - エンジン管理 / 認証コード管理: CRUD（名前のインライン編集、生成・登録・削除、コードの停止・有効化）
- **システム監視**（`SystemView.tsx`） — CPU 使用率（全体・コア別・エンジンのプロセス別）。

認証コードは `useAuth` がオンのときだけ検査する（既定はオフ）。停止したコードは `db.SelectCode` で弾かれる。

⚠️ エンジン・認証コードの登録は管理画面だけで行う。**コマンド形式の登録処理を足さないこと**
（リリース版は `-H windowsgui` でビルドされ、コマンドの出力が表示されない）。
サブコマンドは `version` だけ（`command.go`）。

### 共有フロントパッケージ `@shinte/web`

`core/web` の Web Component `<shogi-board>` と SFEN/USI ロジックを利用する。npm install せずに
参照するため、`vite.config.ts` の `resolve.alias` と `tsconfig.json` の `paths` で
`@shinte/web` → `../../../../core/web` に解決している（`server.fs.allow` にリポジトリルートを追加）。
JSX で使うための型は `src/shogi-board.d.ts`、登録は `App.tsx` の副作用 import（`import '@shinte/web'`）。

### データベースのスキーマ

CSVQ（CSV ファイルに対する SQL）。置き場所は `GetRunDir` の下の `db/`
（開発モードはカレントディレクトリ、リリースモードは実行ファイルのディレクトリ）。

- `engines.csv` — 列: `id`, `path`, `created_date`, `name`
- `codes.csv` — 列: `code`, `created_date`, `updated_date`, `disabled`, `name`

起動時に `migrateCSVColumn()` が移行する。足りない列があれば末尾に追加する。

### クライアントの設定

`prokishi.ini`（TOML）は実行ファイルと同じ場所（開発モードはカレントディレクトリ）に置く。
無ければ下の雛形を作り、`errIniCreated` で終了する。

⚠️ クライアントの標準入出力は将棋 GUI との USI のやり取りそのもの。**標準入力を読んで問い合わせたり、
標準出力に USI 以外を書いたりしないこと**（メッセージは標準エラーかログへ）。

```toml
host = "localhost"
port = 8080
code = ""        # 認証コード（任意）
engineId = ""    # エンジンの UUID（必須）
logLevel = "warn"
```

### Wails3 の注意点

- Wails3 は v3.0.0-beta.3（`_cmd/prokishi-server/go.mod`）。手元の `wails3` CLI とは版が
  ずれていることがあるので、bindings などの生成物を作り直すときは注意する（ルートの `AGENTS.md` 参照）。
- Taskfile.yml は名前空間形式（includes: common, windows, darwin, linux）。
- 終了確認はフロントの MUI Dialog とイベントの emit（`request-close`）で行う。
  `RegisterHook` の中からダイアログを呼ばないこと（デッドロックする）。
