# Tasks: MCP over SOCKS Bridge

**Input**: Design documents from `/specs/001-mcp-over-socks-bridge/`
**Prerequisites**: plan.md ✓, spec.md ✓, research.md ✓, data-model.md ✓, quickstart.md ✓

**Tests**: Tests are included as they are essential for a CLI tool that bridges network protocols.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go CLI project**: `cmd/`, `internal/`, `tests/` at repository root

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Create project directory structure per implementation plan (`cmd/`, `internal/`, `tests/`)
- [ ] T002 Initialize Go module with `go mod init github.com/iiharu/mcp-over-socks`
- [ ] T003 Add dependencies: `github.com/modelcontextprotocol/go-sdk`, `golang.org/x/net/proxy`
- [ ] T004 [P] Create Makefile with build, test, lint targets in `Makefile`
- [ ] T005 [P] Create .gitignore for Go project in `.gitignore`

**Checkpoint**: Project structure ready for development

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T006 Implement Logger with log levels in `internal/logging/logger.go`
- [ ] T007 [P] Implement Config struct with validation in `internal/config/config.go`
- [ ] T008 Implement CLI flag parsing in `cmd/mcp-over-socks/main.go` (--proxy, --server, --help, --timeout, --log)
- [ ] T009 [P] Implement SOCKS5 Dialer wrapper in `internal/transport/socks.go`

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story 1 - SSE MCP サーバーへの SOCKS 経由接続 (Priority: P1) 🎯 MVP

**Goal**: Cursor から SOCKS プロキシ経由で SSE 形式の MCP サーバーに接続し、ツールを実行できる

**Independent Test**: `echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | mcp-over-socks --proxy socks5://localhost:1080 --server http://remote:8080/sse` でツール一覧が返される

### Implementation for User Story 1

- [ ] T010 [US1] Implement SSEClient struct with Connect, Send, Receive, Close in `internal/transport/sse.go`
- [ ] T011 [US1] Implement SSE event parsing (data: lines) in `internal/transport/sse.go`
- [ ] T012 [US1] Implement Bridge struct with Run method in `internal/bridge/bridge.go`
- [ ] T013 [US1] Integrate Bridge with MCP Server (using mcp.NewServer) in `internal/bridge/bridge.go`
- [ ] T014 [US1] Wire up main.go to create Config, Logger, SSEClient, Bridge and run in `cmd/mcp-over-socks/main.go`
- [ ] T015 [US1] Add stdio transport handling (stdin read loop, stdout write) in `internal/bridge/bridge.go`

### Tests for User Story 1

- [ ] T016 [P] [US1] Unit test for Config validation in `tests/unit/config_test.go`
- [ ] T017 [P] [US1] Unit test for SSE event parsing in `tests/unit/sse_test.go`
- [ ] T018 [US1] Integration test with mock SSE server in `tests/integration/bridge_test.go`

**Checkpoint**: MVP complete - SSE MCP サーバーに SOCKS 経由で接続可能

---

## Phase 4: User Story 2 - 接続エラーのハンドリング (Priority: P2)

**Goal**: 接続エラー時に明確なエラーメッセージを受け取り、問題を診断できる

**Independent Test**: 無効なプロキシアドレスを指定した場合に適切なエラーメッセージが stderr に出力される

### Implementation for User Story 2

- [ ] T019 [US2] Define error types (ErrInvalidConfig, ErrProxyConnection, ErrServerConnection) in `internal/bridge/errors.go`
- [ ] T020 [US2] Add SOCKS proxy connection error handling with user-friendly messages in `internal/transport/socks.go`
- [ ] T021 [US2] Add SSE server connection error handling with user-friendly messages in `internal/transport/sse.go`
- [ ] T022 [US2] Add argument validation errors with help display in `cmd/mcp-over-socks/main.go`
- [ ] T023 [US2] Add connection state logging (connecting, connected, disconnected) in `internal/bridge/bridge.go`

### Tests for User Story 2

- [ ] T024 [P] [US2] Unit test for error messages in `tests/unit/errors_test.go`
- [ ] T025 [US2] Integration test for connection failure scenarios in `tests/integration/error_handling_test.go`

**Checkpoint**: エラーハンドリング完了 - ユーザーは接続問題を診断可能

---

## Phase 5: User Story 3 - Streamable HTTP MCP サーバーへの接続 (Priority: P3)

**Goal**: SSE だけでなく Streamable HTTP 形式の MCP サーバーにも接続できる

**Independent Test**: Streamable HTTP MCP サーバーに対してツール呼び出しが成功する

### Implementation for User Story 3

- [ ] T026 [US3] Implement StreamableHTTPClient in `internal/transport/streamable_http.go`
- [ ] T027 [US3] Add transport type detection (SSE vs Streamable HTTP) based on server response in `internal/transport/detector.go`
- [ ] T028 [US3] Update Bridge to support both transport types in `internal/bridge/bridge.go`
- [ ] T029 [US3] Add --transport flag for manual override (auto, sse, streamable) in `cmd/mcp-over-socks/main.go`

### Tests for User Story 3

- [ ] T030 [P] [US3] Unit test for transport detection in `tests/unit/detector_test.go`
- [ ] T031 [US3] Integration test with mock Streamable HTTP server in `tests/integration/streamable_http_test.go`

**Checkpoint**: 複数トランスポートサポート完了

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] T032 [P] Update README.md with installation, usage, and examples
- [ ] T033 [P] Add GoReleaser configuration for cross-platform builds in `.goreleaser.yml`
- [ ] T034 [P] Add GitHub Actions workflow for CI/CD in `.github/workflows/ci.yml`
- [ ] T035 Run quickstart.md validation (manual test with real SOCKS proxy)
- [ ] T036 Performance profiling and optimization if needed

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can proceed in priority order (P1 → P2 → P3)
  - Or in parallel if staffed
- **Polish (Phase 6)**: Depends on at least US1 (MVP) being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - Builds on US1 patterns but independently testable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - Extends transport layer, independently testable

### Within Each User Story

- Core types and interfaces first
- Implementation before integration
- Tests after implementation (or before if TDD preferred)
- Story complete before moving to next priority

### Parallel Opportunities

- T004, T005: Makefile and .gitignore can be created in parallel
- T007, T009: Config and SOCKS Dialer can be implemented in parallel
- T016, T017: Unit tests within US1 can run in parallel
- T024: Error tests can run in parallel
- T030: Detector tests can run in parallel
- T032, T033, T034: All polish tasks can run in parallel

---

## Parallel Example: User Story 1

```bash
# After Phase 2 Foundation is complete:

# These can run in parallel (different files):
Task T016: "Unit test for Config validation in tests/unit/config_test.go"
Task T017: "Unit test for SSE event parsing in tests/unit/sse_test.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test with real SOCKS proxy and SSE server
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → **MVP Release!**
3. Add User Story 2 → Test independently → Quality Release
4. Add User Story 3 → Test independently → Full Feature Release
5. Each story adds value without breaking previous stories

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- All file paths are relative to repository root

