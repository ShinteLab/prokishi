# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Project Is

Prokishi is a USI (Universal Shogi Interface) protocol proxy written in Go. It splits a shogi engine into a lightweight client (`prokishi`) and a remote server (`prokishi-server`) connected via gRPC. The client sits between a shogi GUI and the server, forwarding USI commands over the network so the computationally intensive engine runs remotely.

## Build Commands

```powershell
# Development build (reads config from working directory)
go build -ldflags "-X main.version=0.0.0" -o prokishi.exe ./_cmd/prokishi/
go build -ldflags "-X main.version=0.0.0" -o prokishi-server.exe ./_cmd/prokishi-server/

# Release build (reads config from executable directory)
$version = git describe --tags --abbrev=0
go build -ldflags "-X main.version=$version" -o prokishi.exe ./_cmd/prokishi/
go build -ldflags "-X main.version=$version" -o prokishi-server.exe ./_cmd/prokishi-server/
```

Version is injected via ldflags. An empty version string signals dev mode (config read from CWD); any non-empty version signals release mode (config read from the executable's directory).

## Test Commands

```powershell
go test ./...
go test ./usi/...
```

Integration tests require an actual shogi engine binary; most manual testing is done with ShogiGUI.

## Regenerate Protobuf

If `api/api.proto` changes, regenerate the Go bindings:

```bash
protoc --go_out=. --go-grpc_out=. api/api.proto
```

## Architecture

### Data Flow

```
[Shogi GUI]
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
| `_cmd/prokishi/` | Client `main.go` entry point |
| `_cmd/prokishi-server/` | Server `main.go` entry point |

### Server State

The server maintains an in-memory map from connection UUID → engine `Sender`. Each authenticated gRPC connection gets its own engine subprocess. The CSVQ database (stored in `db/` relative to the executable) holds two tables: `engines` (id, path) and `codes` (auth codes).

### Client Configuration

`prokishi.ini` (TOML) is auto-created on first run:

```toml
host = "localhost"
port = 8080
code = ""        # optional auth code
engineId = ""    # engine UUID (required)
logLevel = "warn"
```

### Server CLI Commands

```powershell
# Engine management
prokishi-server engine generate "C:\path\to\engine.exe"  # auto-generate ID
prokishi-server engine register <id> "C:\path\to\engine.exe"
prokishi-server engine          # list all
prokishi-server engine delete <id>

# Auth code management (optional)
prokishi-server code generate
prokishi-server code register <code>
prokishi-server code           # list all
prokishi-server code delete <code>

# Run server (default port 8080)
prokishi-server -p 9090
```
