# Data Model: MCP over SOCKS Bridge

**Feature Branch**: `001-mcp-over-socks-bridge`  
**Date**: 2025-12-19

## Overview

このプロジェクトはステートレスなブリッジであり、永続化するデータはない。以下は実行時に使用する主要な構造体とインターフェースを定義する。

## Core Types

### Config

コマンドライン引数から構築される設定構造体。

```go
// Config はブリッジの設定を保持する
type Config struct {
    // ProxyAddr は SOCKS5 プロキシのアドレス (e.g., "socks5://localhost:1080")
    ProxyAddr string

    // ServerURL はリモート MCP サーバーの URL (e.g., "http://remote:8080/sse")
    ServerURL string

    // Timeout は HTTP リクエストのタイムアウト（デフォルト: 30s）
    Timeout time.Duration

    // LogLevel はログの詳細度 ("debug", "info", "error")
    LogLevel string
}
```

**Validation Rules**:
- `ProxyAddr` は空でない (MUST)
- `ProxyAddr` は `socks5://` で始まる (MUST)
- `ServerURL` は空でない (MUST)
- `ServerURL` は `http://` または `https://` で始まる (MUST)
- `Timeout` > 0 (デフォルト: 30s)

### Bridge

stdio と SSE 間のプロトコル変換を行うメインコンポーネント。

```go
// Bridge はプロトコル変換を行う
type Bridge struct {
    config    *Config
    sseClient *SSEClient
    logger    *Logger
}

// Bridge のインターフェース
type BridgeInterface interface {
    // Run はブリッジを起動し、終了するまでブロックする
    Run(ctx context.Context) error
}
```

**State Transitions**:
```
[Created] → Run() → [Connecting] → [Running] → [Closed]
                         ↓
                    [Error] → exit(1)
```

### SSEClient

SSE プロトコルでリモート MCP サーバーと通信するクライアント。

```go
// SSEClient は SOCKS5 経由で SSE サーバーに接続する
type SSEClient struct {
    httpClient *http.Client
    serverURL  string
    dialer     proxy.Dialer
}

// SSEClient のインターフェース
type SSEClientInterface interface {
    // Connect は SSE ストリームに接続する
    Connect(ctx context.Context) error

    // Send は JSON-RPC リクエストを送信する
    Send(ctx context.Context, request []byte) error

    // Receive は JSON-RPC レスポンスを受信するチャネルを返す
    Receive() <-chan []byte

    // Close は接続を閉じる
    Close() error
}
```

### Logger

stderr へのログ出力を担当。

```go
// Logger は stderr にログを出力する
type Logger struct {
    level  LogLevel
    writer io.Writer // os.Stderr
}

type LogLevel int

const (
    LogLevelError LogLevel = iota
    LogLevelInfo
    LogLevelDebug
)
```

## Message Flow

### Request Flow (Cursor → Remote Server)

```
1. stdin から JSON-RPC リクエスト読み取り
2. Bridge がリクエストを受信
3. SSEClient 経由でリモートサーバーに送信
4. レスポンス待ち
```

### Response Flow (Remote Server → Cursor)

```
1. SSEClient が SSE イベントを受信
2. data: フィールドから JSON-RPC レスポンスを抽出
3. Bridge が stdout に出力
```

## Error Types

```go
// ErrInvalidConfig は設定のバリデーションエラー
var ErrInvalidConfig = errors.New("invalid configuration")

// ErrProxyConnection は SOCKS プロキシへの接続エラー
var ErrProxyConnection = errors.New("failed to connect to SOCKS proxy")

// ErrServerConnection はリモート MCP サーバーへの接続エラー
var ErrServerConnection = errors.New("failed to connect to MCP server")

// ErrTimeout はリクエストタイムアウト
var ErrTimeout = errors.New("request timeout")
```

## Relationships

```
┌─────────────────────────────────────────────────────────┐
│                         main.go                         │
│  - flag パース → Config 作成                            │
│  - Bridge 作成 → Run()                                  │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                        Bridge                           │
│  - MCP Server として動作                                │
│  - リクエストを SSEClient に委譲                        │
│  - レスポンスを stdout に出力                           │
└─────────────────────────────────────────────────────────┘
              │                            │
              ▼                            ▼
┌──────────────────────┐      ┌──────────────────────────┐
│      SSEClient       │      │         Logger           │
│  - SOCKS5 Dialer     │      │  - stderr 出力           │
│  - HTTP Client       │      │  - ログレベル制御        │
│  - SSE パース        │      │                          │
└──────────────────────┘      └──────────────────────────┘
```

