# Research: MCP over SOCKS Bridge

**Feature Branch**: `001-mcp-over-socks-bridge`  
**Date**: 2025-12-19

## Research Areas

### 1. MCP Go SDK の使用方法

**Decision**: `github.com/modelcontextprotocol/go-sdk` を使用

**Rationale**: 
- Google と協力して維持されている公式 SDK
- Server と Client の両方の実装をサポート
- `mcp.StdioTransport` で stdio 通信が簡単に実装可能
- SSE クライアント機能も提供

**Alternatives considered**:
- `github.com/mark3labs/mcp-go`: 非公式だが成熟している。公式 SDK が利用可能なため却下。
- 独自実装: MCP プロトコルの複雑さを考慮すると非効率。

**Key findings**:
- Server 作成: `mcp.NewServer()` → `server.Run(ctx, transport)`
- Client 作成: `mcp.NewClient()` → `client.Connect(ctx, transport, nil)`
- stdio transport: `mcp.StdioTransport{}`
- HTTP/SSE transport: カスタム実装が必要（SDK は HTTP クライアントを直接提供しない）

### 2. SOCKS5 プロキシ経由の HTTP 接続

**Decision**: `golang.org/x/net/proxy` を使用

**Rationale**:
- Go の準標準ライブラリ（公式 x リポジトリ）
- SOCKS5 プロトコルを完全サポート
- `http.Transport` の `DialContext` に統合可能
- 認証（ユーザー名/パスワード）もサポート

**Alternatives considered**:
- `github.com/armon/go-socks5`: SOCKS5 サーバー実装。クライアントには不向き。
- 独自 SOCKS5 実装: 準標準ライブラリで十分なため不要。

**Key findings**:
```go
import "golang.org/x/net/proxy"

// SOCKS5 Dialer の作成
dialer, err := proxy.SOCKS5("tcp", "localhost:1080", nil, proxy.Direct)

// http.Transport に統合
transport := &http.Transport{
    DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
        return dialer.Dial(network, addr)
    },
}
client := &http.Client{Transport: transport}
```

### 3. SSE (Server-Sent Events) クライアント実装

**Decision**: 標準 `net/http` + 手動パース

**Rationale**:
- SSE は単純なテキストプロトコル
- 専用ライブラリは不要
- `bufio.Scanner` でストリームを行ごとに読み取り可能

**Alternatives considered**:
- `github.com/r3labs/sse`: 人気の SSE ライブラリだが、サードパーティ依存を避けるため却下。
- `github.com/tmaxmax/go-sse`: 同上。

**Key findings**:
```go
// SSE ストリームの読み取り
resp, _ := client.Get("http://server/sse")
scanner := bufio.NewScanner(resp.Body)
for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "data: ") {
        data := strings.TrimPrefix(line, "data: ")
        // JSON パース
    }
}
```

### 4. stdio ↔ SSE ブリッジアーキテクチャ

**Decision**: MCP Client として動作し、リモートサーバーに接続

**Rationale**:
- ブリッジは「MCP サーバー」として Cursor に見せかける
- 内部では「MCP クライアント」としてリモートサーバーに接続
- メッセージを透過的に転送

**Architecture**:
```
Cursor (MCP Client)
    ↓ stdio (JSON-RPC)
mcp-over-socks (Bridge)
    ↓ SOCKS5 Proxy
    ↓ HTTP/SSE
Remote MCP Server
```

**Key design decisions**:
1. Bridge は MCP Server として実装（`mcp.NewServer()`）
2. 受信したリクエストをリモートサーバーに転送
3. レスポンスをそのまま Cursor に返す
4. Tools, Resources, Prompts すべてをプロキシ

### 5. エラーハンドリング戦略

**Decision**: 接続エラーは即座に終了、リクエストエラーは MCP エラーレスポンス

**Rationale**:
- 起動時の接続エラーは回復不能なため即座に終了
- リクエスト中のエラーは MCP プロトコルに従ってエラーレスポンスを返す
- すべてのエラーは stderr にログ出力

**Error categories**:
1. **Fatal errors** (exit 1):
   - 引数パースエラー
   - SOCKS プロキシ接続失敗
   - 初期 SSE 接続失敗
2. **Recoverable errors** (MCP error response):
   - リクエスト中のタイムアウト
   - リモートサーバーのエラーレスポンス

## Resolved Clarifications

| Item | Resolution |
|------|------------|
| SSE vs Streamable HTTP の優先度 | SSE を P1、Streamable HTTP を P3 で実装 |
| 再接続ロジック | 初期バージョンでは再接続なし、接続断で終了 |
| 認証のサポート | SOCKS5 認証は SHOULD、初期バージョンでは未実装可 |

## Next Steps

1. data-model.md で Config 構造体と Bridge インターフェースを定義
2. quickstart.md で使用例とテスト方法を記載
3. tasks.md で実装タスクを分割

