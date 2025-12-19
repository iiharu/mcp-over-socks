# Feature Specification: MCP over SOCKS Bridge

**Feature Branch**: `001-mcp-over-socks-bridge`  
**Created**: 2025-12-19  
**Status**: Draft  
**Input**: SOCKS proxy 経由で接続できる SSE/Streamable HTTP MCP を Cursor で使いたい

## User Scenarios & Testing *(mandatory)*

### User Story 1 - SSE MCP サーバーへの SOCKS 経由接続 (Priority: P1)

ユーザーは Cursor から SOCKS プロキシ経由で SSE 形式の MCP サーバーに接続し、ツールを実行できる。

**Why this priority**: これがプロジェクトの核となる機能。SSE は最も一般的な MCP トランスポート形式であり、まずこれを動作させることで MVP が完成する。

**Independent Test**: ローカルで SOCKS プロキシと SSE MCP サーバーをセットアップし、ブリッジ経由でツール呼び出しが成功することを確認できる。

**Acceptance Scenarios**:

1. **Given** SOCKS プロキシが `localhost:1080` で動作中、SSE MCP サーバーが `http://remote:8080/sse` で動作中, **When** ユーザーが `mcp-over-socks --proxy socks5://localhost:1080 --server http://remote:8080/sse` を実行, **Then** ブリッジが起動し、stdin から JSON-RPC リクエストを受け付ける
2. **Given** ブリッジが起動中, **When** Cursor から `tools/list` リクエストが送信される, **Then** リモート MCP サーバーのツール一覧が stdout に返される
3. **Given** ブリッジが起動中, **When** Cursor から `tools/call` リクエストが送信される, **Then** リモート MCP サーバーでツールが実行され、結果が stdout に返される

---

### User Story 2 - 接続エラーのハンドリング (Priority: P2)

ユーザーは接続エラー時に明確なエラーメッセージを受け取り、問題を診断できる。

**Why this priority**: MVP の品質を確保するために必要。接続失敗時にユーザーが原因を特定できなければ、ツールとして使い物にならない。

**Independent Test**: 無効なプロキシアドレスや到達不能なサーバーを指定した場合に、適切なエラーメッセージが表示されることを確認できる。

**Acceptance Scenarios**:

1. **Given** SOCKS プロキシが停止中, **When** ブリッジを起動しようとする, **Then** 「SOCKS プロキシに接続できません」エラーが stderr に出力される
2. **Given** SOCKS プロキシは動作中だがリモートサーバーが停止中, **When** ブリッジを起動しようとする, **Then** 「リモート MCP サーバーに接続できません」エラーが stderr に出力される
3. **Given** 無効な引数が指定された, **When** ブリッジを起動しようとする, **Then** 使用方法のヘルプが表示され、終了コード 1 で終了する

---

### User Story 3 - Streamable HTTP MCP サーバーへの接続 (Priority: P3)

ユーザーは SSE だけでなく、Streamable HTTP 形式の MCP サーバーにも接続できる。

**Why this priority**: SSE が動作した後の拡張機能。一部の MCP サーバーは Streamable HTTP を使用しているため、互換性のために必要。

**Independent Test**: Streamable HTTP MCP サーバーに対してツール呼び出しが成功することを確認できる。

**Acceptance Scenarios**:

1. **Given** Streamable HTTP MCP サーバーが動作中, **When** ブリッジ経由で接続, **Then** SSE と同様にツール呼び出しが成功する

---

### Edge Cases

- プロキシ認証が必要な場合（ユーザー名/パスワード）の動作
- 接続中にプロキシが停止した場合の再接続動作
- 非常に大きなレスポンス（数 MB）を受信した場合の動作
- 複数の同時リクエストが発生した場合の動作

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: システムは stdin から JSON-RPC リクエストを受信し、stdout に JSON-RPC レスポンスを出力しなければならない (MUST)
- **FR-002**: システムは SOCKS5 プロキシ経由でリモート MCP サーバーに接続しなければならない (MUST)
- **FR-003**: システムは SSE (Server-Sent Events) 形式の MCP サーバーをサポートしなければならない (MUST)
- **FR-004**: システムは `--proxy` と `--server` のコマンドライン引数を受け付けなければならない (MUST)
- **FR-005**: システムは `--help` オプションで使用方法を表示しなければならない (MUST)
- **FR-006**: システムはすべてのエラーを stderr に出力しなければならない (MUST)
- **FR-007**: システムは MCP メッセージを一切改変せず透過的に転送しなければならない (MUST)
- **FR-008**: システムは Streamable HTTP 形式の MCP サーバーをサポートすべきである (SHOULD)
- **FR-009**: システムは SOCKS5 認証（ユーザー名/パスワード）をサポートすべきである (SHOULD)

### Key Entities

- **Bridge**: stdio と HTTP/SSE 間のプロトコル変換を行うメインコンポーネント
- **SOCKS Client**: SOCKS5 プロキシ経由で TCP 接続を確立するコンポーネント
- **SSE Transport**: SSE プロトコルで MCP メッセージを送受信するコンポーネント
- **Config**: コマンドライン引数から読み込んだ設定を保持する構造体

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Cursor の MCP 設定で `command` として指定し、リモート MCP サーバーのツールが使用できる
- **SC-002**: SOCKS プロキシ経由でのツール呼び出しのレイテンシが直接接続と比較して +100ms 以内
- **SC-003**: 1時間の連続使用で接続が維持される（自動再接続を含む）
- **SC-004**: すべての MCP メッセージ（tools/list, tools/call, resources/*, prompts/*）が正しく転送される

