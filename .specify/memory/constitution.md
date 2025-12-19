<!--
Sync Impact Report
==================
Version change: 1.2.1 → 1.3.0

Modified sections:
  - Technology Stack: 公式 MCP Go SDK の使用を明確化
  - Principle III: SDK のトランスポートを使用することを明記

Added sections: None

Removed sections: None

Templates requiring updates:
  - ✅ plan-template.md (compatible with current principles)
  - ✅ spec-template.md (compatible with current principles)
  - ✅ tasks-template.md (compatible with current principles)

Follow-up TODOs: None

Change rationale: 公式 SDK 使用の明確化（MINOR バージョンアップ）
  - 独自実装から公式 SDK (mcp.SSEClientTransport, mcp.StreamableClientTransport) への移行完了
  - SDK の HTTPClient フィールドにカスタム HTTP クライアント（SOCKS プロキシ経由）を注入
-->

# MCP over SOCKS Constitution

## Core Principles

### I. Stdio Bridge Interface

このプロジェクトは Cursor との通信において標準入出力 (stdio) ベースの MCP サーバーとして振る舞う。

- システムは stdin からの JSON-RPC リクエストを受け付け、stdout に JSON-RPC レスポンスを出力しなければならない (MUST)
- エラーメッセージおよびログは stderr に出力しなければならない (MUST)
- Cursor の MCP 設定において `command` 形式で起動可能でなければならない (MUST)

**根拠**: Cursor は stdio MCP サーバーのみで環境変数設定をサポートしており、SSE/Streamable HTTP MCP では環境変数を渡せない。stdio インターフェースを介することでこの制約を回避する。

### II. SOCKS5 Proxy Routing

すべてのリモート MCP サーバーへの接続は SOCKS5 プロキシを経由しなければならない。

- SOCKS5 プロトコルに準拠した接続を確立しなければならない (MUST)
- 以下のプロキシスキームをサポートしなければならない (MUST):
  - `socks5://` - ローカルで DNS 解決を行い、IP アドレスでプロキシに接続
  - `socks5h://` - プロキシサーバー側（リモート）で DNS 解決を行う
- プロキシ設定（ホスト、ポート、認証情報）はコマンドライン引数から読み込む (MUST)
- プロキシ接続失敗時は明確なエラーメッセージを返さなければならない (MUST)
- SOCKS5 認証（ユーザー名/パスワード）をサポートすべきである (SHOULD)

**根拠**: SOCKS プロキシ経由でのみアクセス可能なネットワーク環境に存在する MCP サーバーへの接続を可能にする。`socks5h://` はプライベートネットワーク内でのみ解決可能なホスト名（例: 内部 DNS）へのアクセスに必須。

### III. Protocol Translation (SDK Integration)

公式 MCP Go SDK を使用して、stdio と SSE/Streamable HTTP 間の MCP プロトコル変換を行う。

- 公式 MCP Go SDK (`github.com/modelcontextprotocol/go-sdk`) のトランスポートを使用しなければならない (MUST)
- SDK の `SSEClientTransport` および `StreamableClientTransport` を使用し、`HTTPClient` フィールドに SOCKS プロキシ経由の HTTP クライアントを注入する (MUST)
- SDK の `jsonrpc.DecodeMessage` / `jsonrpc.EncodeMessage` を使用してメッセージを処理する (MUST)
- MCP メッセージの内容は一切改変せず透過的に転送しなければならない (MUST)

**根拠**: 公式 SDK を使用することで、MCP プロトコルの変更に追従しやすくなり、互換性の問題を最小限に抑えられる。

### IV. Command-Line Configuration

設定はコマンドライン引数から読み込む。

- すべての設定はコマンドライン引数で指定しなければならない (MUST)
- 必須引数: プロキシアドレス（`--proxy`）、リモート MCP サーバー URL（`--server`）
- オプション引数: タイムアウト、ログレベル、トランスポートタイプ
- 引数のバリデーションを行い、不正な値または必須引数の欠落時は起動を中止しなければならない (MUST)
- ヘルプオプション（`--help`）で使用方法を表示しなければならない (MUST)

**根拠**: MVP としてシンプルさを優先し、引数のみの設定方法を採用。Cursor の `args` 設定で直接指定可能。将来的に必要に応じて環境変数・設定ファイルのサポートを追加できる。

### V. Error Handling & Resilience

エラー状態を適切に処理し、可能な限りサービスを継続する。

- ネットワークエラー時は自動的に再接続を試みるべきである (SHOULD)
- 回復不可能なエラーは MCP エラーレスポンスとして返さなければならない (MUST)
- すべてのエラーは stderr にログ出力しなければならない (MUST)
- 接続状態の変化（接続、切断、再接続）をログに記録しなければならない (MUST)

**根拠**: ネットワーク経由の接続は不安定になりうるため、堅牢なエラー処理が必要。

## Technology Stack

プロジェクトの技術スタック:

- **言語**: Go 1.21+
- **SOCKS ライブラリ**: `golang.org/x/net/proxy`（Go 準標準ライブラリ）
- **MCP SDK**: 公式 MCP Go SDK (`github.com/modelcontextprotocol/go-sdk`)
  - `mcp.SSEClientTransport` - SSE トランスポート
  - `mcp.StreamableClientTransport` - Streamable HTTP トランスポート
  - `jsonrpc.DecodeMessage` / `jsonrpc.EncodeMessage` - メッセージのエンコード/デコード
- **ビルドツール**: Go modules (`go mod`)
- **パッケージング**: 単一バイナリとして配布（クロスコンパイル対応）

### 依存関係の方針

- 公式 MCP Go SDK を使用する (MUST)
- SOCKS プロキシ対応には `golang.org/x/net/proxy` を使用する (MUST)
- その他のサードパーティライブラリの追加は、公式に代替がない場合のみ許可される
- 追加する場合は、メンテナンス状況とセキュリティを評価しなければならない

## Development Workflow

開発ワークフロー:

- **テスト**: `go test` によるユニットテストおよび統合テストを作成
- **CI/CD**: GitHub Actions を使用
- **コードスタイル**: `gofmt` および `go vet` を使用
- **静的解析**: `staticcheck` または `golangci-lint` を使用
- **ドキュメント**: README に使用方法、設定例、トラブルシューティングを記載
- **リリース**: セマンティックバージョニングに従い、GoReleaser でビルド・配布

## Governance

- この Constitution はプロジェクトのすべての設計判断において優先される
- 原則の変更は本ドキュメントの更新、レビュー、および移行計画を必要とする
- すべての PR/レビューは原則への準拠を確認しなければならない
- 複雑さの追加は Complexity Tracking セクションで正当化しなければならない
- 開発ガイダンスは `docs/development.md` に記載する（必要に応じて作成）

**Version**: 1.3.0 | **Ratified**: 2025-12-19 | **Last Amended**: 2025-12-19
