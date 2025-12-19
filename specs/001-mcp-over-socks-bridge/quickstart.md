# Quickstart: MCP over SOCKS Bridge

**Feature Branch**: `001-mcp-over-socks-bridge`  
**Date**: 2025-12-19

## Prerequisites

- Go 1.21 以上がインストールされていること
- SOCKS5 プロキシが利用可能であること（例: ssh -D, Shadowsocks）
- リモート MCP サーバーが SSE エンドポイントを提供していること

## Installation

### ソースからビルド

```bash
# リポジトリをクローン
git clone https://github.com/iiharu/mcp-over-socks.git
cd mcp-over-socks

# ビルド
go build -o mcp-over-socks ./cmd/mcp-over-socks

# インストール（オプション）
go install ./cmd/mcp-over-socks
```

### リリースバイナリ

```bash
# macOS (Apple Silicon)
curl -L https://github.com/iiharu/mcp-over-socks/releases/latest/download/mcp-over-socks-darwin-arm64 -o mcp-over-socks
chmod +x mcp-over-socks

# Linux (x86_64)
curl -L https://github.com/iiharu/mcp-over-socks/releases/latest/download/mcp-over-socks-linux-amd64 -o mcp-over-socks
chmod +x mcp-over-socks
```

## Usage

### 基本的な使い方

```bash
mcp-over-socks --proxy socks5://localhost:1080 --server http://remote-server:8080/sse
```

### コマンドラインオプション

```
Usage: mcp-over-socks [options]

Required:
  --proxy   SOCKS5 プロキシの URL (例: socks5://localhost:1080)
  --server  リモート MCP サーバーの URL (例: http://remote:8080/sse)

Optional:
  --timeout  リクエストタイムアウト (デフォルト: 30s)
  --log      ログレベル: debug, info, error (デフォルト: info)
  --help     このヘルプを表示
```

## Cursor での設定

### mcp.json の設定

`~/.cursor/mcp.json` に以下を追加:

```json
{
  "mcpServers": {
    "remote-mcp": {
      "command": "/path/to/mcp-over-socks",
      "args": [
        "--proxy", "socks5://localhost:1080",
        "--server", "http://your-mcp-server.example.com/sse"
      ]
    }
  }
}
```

### SSH SOCKS プロキシの使用

SSH 経由で SOCKS プロキシを作成する場合:

```bash
# 別のターミナルで SSH プロキシを起動
ssh -D 1080 -N user@remote-host
```

## Testing

### ローカルでのテスト

```bash
# ユニットテスト
go test ./...

# 統合テスト（モックサーバー使用）
go test -tags=integration ./tests/integration/...
```

### 手動テスト

1. SOCKS プロキシを起動:
   ```bash
   ssh -D 1080 -N user@jump-host
   ```

2. テスト用 MCP サーバーを起動（リモートで）:
   ```bash
   # リモートマシンで
   npx @modelcontextprotocol/server-everything
   ```

3. ブリッジを起動:
   ```bash
   mcp-over-socks --proxy socks5://localhost:1080 --server http://remote:3000/sse
   ```

4. JSON-RPC リクエストを送信:
   ```bash
   echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | mcp-over-socks --proxy socks5://localhost:1080 --server http://remote:3000/sse
   ```

## Troubleshooting

### 接続エラー

**症状**: `failed to connect to SOCKS proxy`

**対処法**:
1. SOCKS プロキシが起動していることを確認
2. プロキシアドレスが正しいことを確認
3. ファイアウォールがポートをブロックしていないことを確認

### タイムアウト

**症状**: `request timeout`

**対処法**:
1. `--timeout` オプションを増やす
2. ネットワーク接続を確認
3. リモートサーバーが応答していることを確認

### ログの確認

デバッグモードで詳細なログを出力:

```bash
mcp-over-socks --proxy socks5://localhost:1080 --server http://remote:8080/sse --log debug 2>debug.log
```

## Example: Using with Claude MCP Server

```json
{
  "mcpServers": {
    "claude-remote": {
      "command": "/usr/local/bin/mcp-over-socks",
      "args": [
        "--proxy", "socks5://127.0.0.1:1080",
        "--server", "https://internal-claude-server.corp.example.com/mcp/sse"
      ]
    }
  }
}
```

