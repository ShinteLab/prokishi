# _docs/skills

このリポジトリの**作業手順**を置く場所。

| ディレクトリ | 中身 |
|---|---|
| `prokishi-change-api/` | gRPC の API（`api/api.proto`）を変える。再生成・サーバとクライアントの修正・新旧の組み合わせへの配慮 |
| `prokishi-release/` | バージョンを上げて配る。`version` ファイル・`_cmd/version.go`・versionup → タグ → release の流れ |

- **ただの Markdown**（`SKILL.md`）。どのコーディングエージェントでも、人が読んでもよい
- ⚠️ **リポジトリに `.claude/` を置かない**（Claude で使う前提になるため。`.gitignore` で無視している）
- Claude Code でスキルとして使うなら、手元で `.claude/skills/<名前>` から
  ここへジャンクション（シンボリックリンク）を張る。例（PowerShell、リポジトリ直下で）:

  ```powershell
  New-Item -ItemType Junction -Path .claude\skills\prokishi-release -Target (Resolve-Path _docs\skills\prokishi-release)
  ```

- ⚠️ **編集するのはここ**（git で管理しているのはこちらだけ）
