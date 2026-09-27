---
name: prokishi-change-api
description: prokishi のクライアントとサーバの間の gRPC API（api/api.proto）を変えるときの手順。Protobuf の再生成、server/handler.go と client.go の修正、テスト、古いクライアントと新しいサーバ（またはその逆）の組み合わせへの配慮。「api.proto にフィールドを足す」「RPC を足す」「pb.go を作り直す」「protoc を実行する」「クライアントからサーバに新しい情報を送る」ときに使う。
---

# gRPC の API を変える

## 関係するファイル

| ファイル | 中身 |
|---|---|
| `api/api.proto` | 定義。`ConnectionService` / `USISendService` / `USIReceiveService` の 3 つ |
| `api/api.pb.go`, `api/api_grpc.pb.go` | 生成物。**手で編集しない** |
| `server/handler.go` | サーバ側の実装（`Connection` / `Send` / `Receive`） |
| `server/server.go` | サービスの登録（`api.Register...Server`）と `Server` 構造体への埋め込み、認証（`verifyAuthentication`） |
| `client.go`（ルート） | クライアント側の呼び出し（`connect` / `sendServer` / `receiveServer`） |
| `server/handler_test.go` | 本物の gRPC 接続で 3 つのサービスを通すテスト（偽エンジンを使う） |

## 手順

1. `api/api.proto` を直す
2. **`api/` の中で**再生成する（`go_package` が `"../api"` なので、リポジトリ直下で実行すると出力先がずれる）

   ```powershell
   cd api
   protoc --go_out=. --go-grpc_out=. api.proto
   ```

3. サーバ側（`server/handler.go`）とクライアント側（`client.go`）を直す
   - RPC を足したら `server/server.go` の `Server` 構造体への埋め込みと `api.Register...Server` も足す
   - **新しい RPC も `code` を受け取り、`verifyAuthentication` を通すこと**
     （既存の RPC はすべて毎回認証している。接続 ID が要るものは `getEngine` 経由でよい）
   - 管理 UI の接続モニターに出したい情報は `registry`（`AppendLog` など）に流す
4. テストを足して確かめる

   ```powershell
   go vet ./...
   go test ./...
   go test -tags integration ./usi/...
   cd _cmd/prokishi-server; go test ./...   # サーバアプリは入れ子モジュール
   ```

## 新旧の組み合わせ（互換性）

クライアント（macOS など）とサーバ（Windows）は**別々の機械で別々に更新される**。
同じタグから両方を配るが、利用者が片方だけ古いまま使うことは普通に起こる。
**接続時にバージョンを確かめる仕組みは無い**ので、互換性は定義の書き方で保つ。

- ⚠️ **既存フィールドの番号・型を変えない。番号を使い回さない。** 消すときは `reserved` にする
- フィールドを足すのはよい。古い側からは**ゼロ値（空文字・0・false）が届く**ので、
  ゼロ値のときは今までどおり動くように作る
- RPC を足した場合、古いサーバに新しいクライアントが呼ぶと `Unimplemented` エラーになる。
  必須の流れ（接続〜送受信）に新しい RPC を挟むなら、そのエラーのときに旧来の動きに戻すか、
  サーバを先に更新する必要があることをリリースノートに書く
- 既存フィールドの**意味**を変えない（例: `cmd` に USI 以外のものを混ぜない）

## 生成ツールのバージョン

今の生成物は `protoc v4.25.2` / `protoc-gen-go v1.32.0` / `protoc-gen-go-grpc v1.3.0` で作られている
（各 `.pb.go` の先頭に書いてある）。`go.mod` は `google.golang.org/protobuf v1.36.11` /
`google.golang.org/grpc v1.84.0`（ランタイムの方が新しいのは問題ない）。

- 新しい `protoc-gen-go` / `protoc-gen-go-grpc` で生成すると、生成コードがより新しい
  ランタイムを要求して `go.mod` の更新が要ることがある。**生成後に `go build ./...` が通るか確かめ**、
  必要なら `go get` で上げる（上げたら `_cmd/prokishi-server` でも `go mod tidy` とビルドを確かめる）
- ⚠️ ルートで `go mod tidy` すると `github.com/BurntSushi/toml` の require 行が消える
  （使っているのが `go build ./...` の対象外の `_cmd/prokishi` だけのため）。消えたら
  `go get github.com/BurntSushi/toml@<元の版>` で戻す

## クライアントの接続

`client.go` は `grpc.NewClient` を使う（接続は遅延）。最初の `Connection` を
`grpc.WaitForReady(true)` と `connectTimeout`（3 秒）で呼び、つながらなければ
`context deadline exceeded` で失敗する（INSTALL.md のエラー説明はこの文言に依存している）。
- 差分に生成ツールのバージョン行の変化が出るのは問題ない
- `protoc` 本体は PATH に無いことがある（プラグインは `go install` で入るが本体は別途入れる）
