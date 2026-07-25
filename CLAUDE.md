# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Is

Prokishi is a USI (Universal Shogi Interface) protocol proxy written in Go. It splits a shogi engine into a lightweight client (`prokishi`) and a remote server (`prokishi-server`) connected via gRPC. The client sits between a shogi GUI and the server, forwarding USI commands over the network so the computationally intensive engine runs remotely.

Typical deployment: the server runs on a Windows machine with engine binaries (e.g. Apery, YaneuraOu), and the client runs on macOS with ShogiHome (Electron-based cross-platform GUI). Manual testing is also done with ShogiGUI on Windows.

## Build Commands

### prokishi client (plain Go CLI)

```powershell
go build -ldflags "-X main.version=0.0.0" -o prokishi.exe ./_cmd/prokishi/
```

### prokishi-server (Wails3 desktop app)

```powershell
# Development
cd _cmd/prokishi-server
wails3 dev

# Production build (uses -tags production internally)
cd _cmd/prokishi-server
wails3 build
```

### Version Management

Version is stored in `_cmd/prokishi-server/version` and embedded via `//go:embed version`. Build tags control mode:

- `//go:build !production` → `mode_dev.go` (dev mode: config read from CWD)
- `//go:build production` → `mode_prod.go` (release mode: config read from executable directory)

These align with Wails3's Taskfile which uses `-tags production` for release builds. The client uses `-ldflags "-X main.version=..."` and treats empty version as dev mode.

## Test Commands

```powershell
go test ./...
go test ./usi/...
```

Integration tests require an actual shogi engine binary.

## Regenerate Protobuf

If `api/api.proto` changes, regenerate the Go bindings:

```bash
protoc --go_out=. --go-grpc_out=. api/api.proto
```

## CI/CD

- `.github/workflows/versionup.yml` — On push to main, bumps patch version, creates PR, auto-merges, tags. Uses `MY_GITHUB_TOKEN` (PAT) because `GITHUB_TOKEN`-pushed tags don't trigger downstream workflows.
- `.github/workflows/release.yml` — On tag push, builds server (Windows only, extendable via matrix) and client (Windows/macOS/Linux), creates draft GitHub Release.

## Architecture

### Data Flow

```
[Shogi GUI (ShogiHome / ShogiGUI)]
    ↕ stdin/stdout (USI text protocol)
[prokishi client]
    ↕ gRPC (TCP)
[prokishi-server]
    ↕ stdin/stdout (spawns engine subprocess)
[Engine binary (e.g. Apery, YaneuraOu)]
```

### Key Packages

| Package | Role |
|---|---|
| `prokishi` (root) | Client lifecycle: reads config, connects to server, bridges stdin↔gRPC |
| `usi/` | OS process I/O abstraction — `Receiver` reads stdin, `Sender` wraps an engine subprocess |
| `api/` | Protobuf definitions for three gRPC services: `ConnectionService`, `USISendService`, `USIReceiveService` |
| `server/` | gRPC server — authenticates connections, spawns engine processes, streams output back |
| `db/` | CSVQ-based persistence (SQL queries over CSV files) for engine registrations and auth codes |
| `registry/` | In-memory engine registry mapping connection UUID → engine `Sender` |
| `_cmd/prokishi/` | Client entry point (plain Go CLI) |
| `_cmd/prokishi-server/` | Server entry point (Wails3 desktop app with admin UI) |

### Server Admin UI (Wails3)

The server includes a GUI built with Wails3 + React + MUI. Tabs:

- **マスタ管理** — Engine and auth code CRUD (inline name editing, generate/register/delete)
- **接続モニター** — Live connection monitoring
- **盤面** — Shogi board display (`BoardView.tsx`). Uses `<shogi-board>` / `<shogi-hand>` from `@shinte/web` (repo `core/web`). Renders a SFEN, and can pull the latest `position` command from a connection's USI log; `resolvePosition` applies the `moves` (captures/promotions/drops) to show the current board + hands + side-to-move (move application is display-only, no legality check).
- **サーバ設定** — Server config (port, auth toggle) with start/stop controls, client config export

### 共有フロントパッケージ `@shinte/web`

`core/web` の Web Component `<shogi-board>` と SFEN/USI ロジックを利用する。npm install せずに
参照するため、`vite.config.ts` の `resolve.alias` と `tsconfig.json` の `paths` で
`@shinte/web` → `../../../../core/web` に解決している(`server.fs.allow` にリポジトリルートを追加)。
JSX で使うための型は `src/shogi-board.d.ts`、登録は `App.tsx` の副作用 import (`import '@shinte/web'`)。

### Database Schema

CSVQ (SQL over CSV files) in `db/` relative to the executable:

- `engines.csv` — columns: `id`, `path`, `created_date`, `name`
- `codes.csv` — columns: `code`, `created_date`, `updated_date`, `disabled`, `name`

Migration is handled automatically via `migrateCSVColumn()` on startup — new columns are appended if missing.

### Client Configuration

`prokishi.ini` (TOML) is auto-created on first run:

```toml
host = "localhost"
port = 8080
code = ""        # optional auth code
engineId = ""    # engine UUID (required)
logLevel = "warn"
```

### Wails3 Notes

- Wails3 v3.0.0-alpha.98
- Taskfile.yml uses the namespace format (includes: common, windows, darwin, linux)
- Close confirmation uses frontend MUI Dialog + event emit — never call dialogs from RegisterHook (causes deadlock)
