# Implementation Plan: MCP over SOCKS Bridge

**Branch**: `001-mcp-over-socks-bridge` | **Date**: 2025-12-19 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-mcp-over-socks-bridge/spec.md`

## Summary

SOCKS5 プロキシ経由で SSE/Streamable HTTP MCP サーバーに接続するブリッジを作成する。ローカルでは stdio MCP サーバーとして動作し、Cursor から透過的に使用可能。Go 言語で実装し、公式 MCP Go SDK と golang.org/x/net/proxy を使用する。

## Technical Context

**Language/Version**: Go 1.21+  
**Primary Dependencies**: 
- `github.com/modelcontextprotocol/go-sdk` (公式 MCP Go SDK)
- `golang.org/x/net/proxy` (SOCKS5 プロキシ)
- 標準ライブラリ (`net/http`, `bufio`, `encoding/json`, `flag`)

**Storage**: N/A（ステートレス）  
**Testing**: `go test` によるユニットテスト・統合テスト  
**Target Platform**: macOS, Linux, Windows（クロスコンパイル対応）  
**Project Type**: Single CLI application  
**Performance Goals**: レイテンシ +100ms 以内（直接接続比）  
**Constraints**: 単一バイナリ、外部依存なし（実行時）  
**Scale/Scope**: 単一ユーザー向け CLI ツール

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Stdio Bridge Interface | ✅ PASS | stdin/stdout で JSON-RPC、stderr でログ |
| II. SOCKS5 Proxy Routing | ✅ PASS | golang.org/x/net/proxy 使用、socks5:// および socks5h:// サポート |
| III. Protocol Translation (SDK Integration) | ✅ PASS | 公式 MCP Go SDK の SSEClientTransport/StreamableClientTransport 使用 |
| IV. Command-Line Configuration | ✅ PASS | --proxy, --server, --help |
| V. Error Handling & Resilience | ✅ PASS | エラーは stderr、接続状態ログ |

**Technology Stack Compliance**:
- ✅ Go 1.21+
- ✅ 公式 MCP Go SDK
- ✅ golang.org/x/net/proxy（準標準ライブラリ）
- ✅ 単一バイナリ配布

## Project Structure

### Documentation (this feature)

```text
specs/001-mcp-over-socks-bridge/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
└── tasks.md             # Phase 2 output (/speckit.tasks)
```

### Source Code (repository root)

```text
cmd/
└── mcp-over-socks/
    └── main.go          # エントリーポイント、CLI 引数処理

internal/
├── bridge/
│   ├── bridge.go        # stdio ↔ HTTP/SSE ブリッジのメインロジック（公式 MCP Go SDK 使用）
│   └── errors.go        # エラー型定義
├── config/
│   └── config.go        # コマンドライン引数のパース、バリデーション（socks5:// および socks5h:// サポート）
├── transport/
│   └── socks.go         # SOCKS5 プロキシ経由の Dialer（socks5:// および socks5h:// サポート）
└── logging/
    └── logger.go        # stderr へのログ出力

tests/
├── integration/
│   └── bridge_test.go   # 統合テスト（モック SSE サーバー使用）
└── unit/
    ├── config_test.go
    └── transport_test.go

go.mod
go.sum
README.md
Makefile                 # ビルド、テスト、リリース用
```

**Structure Decision**: Single CLI application として `cmd/` と `internal/` の標準 Go プロジェクト構成を採用。

## Complexity Tracking

> **No violations identified** - Design follows constitution principles.
