# Wolf Architecture — R&D Reference

## Compiler Pipeline

```
.wolf source
    ↓ Lexer          internal/lexer/lexer.go
    ↓ Parser         internal/parser/parser.go
    ↓ Resolver       internal/resolver/resolver.go
    ↓ TypeChecker    internal/typechecker/typechecker.go
    ↓ IR Emitter     internal/emitter/emitter.go  → WIR (Wolf IR)
    ↓ LLVM Emitter   internal/emitter/llvm_emitter.go → .ll file
    ↓ llc / clang    → .o object file
    ↓ clang linker + wolf_runtime.o → native binary
```

## Runtime Architecture

```
wolf_runtime.c       (linked into every binary)
    ↑ wolf_config_runtime.h   (compile-time constants from wolf.config)
    ↑ wolf_runtime.h          (public API declarations)
```

### Runtime Subsystems
| Subsystem | File section | Status |
|-----------|-------------|--------|
| Print / IO | wolf_print_* | ✅ Done |
| String ops | wolf_string_* | ✅ Done |
| Arrays | wolf_array_* | ✅ Done |
| Maps | wolf_map_* | ✅ Done |
| Typed values | wolf_value_t / wolf_val_* | ✅ Done |
| JSON encode/decode | wolf_json_* | ✅ Done (unicode fixed) |
| MySQL pool | wolf_pool_* | ✅ Done |
| Pool destroy | wolf_db_pool_destroy() | ✅ Done (session 3) |
| Redis | wolf_redis_* | ✅ Done (real hiredis, session 2) |
| HTTP server | wolf_http_* | ✅ Done |
| Graceful shutdown | SIGTERM/SIGINT handler + drain | ✅ Done (session 3) |
| Request timeout | SO_RCVTIMEO + HTTP 408 | ✅ Done (session 3) |
| SIGPIPE guard | signal(SIGPIPE, SIG_IGN) | ✅ Done (session 3) |
| JWT | wolf_jwt_* | ✅ Done (stub — needs real HMAC) |
| Memory arena | wolf_req_alloc / wolf_req_arena_flush | ✅ Done |
| MSSQL | wolf_db_* (MSSQL path) | 🔴 Mock only |
| File uploads | multipart/form-data | ⬜ Not started |
| WebSocket | wolf_ws_* | ⬜ Not started |

## Key Design Decisions

### ADR-001: LLVM IR as compilation target (not Go transpile)
- **Decision:** Emit LLVM IR directly, not Go/C source
- **Reason:** True native performance, no GC pauses, sub-200ms request latency goal
- **Date:** Early 2026

### ADR-002: wolf.config (INI format, like php.ini)
- **Decision:** INI-style config, values baked at compile time as -D flags
- **Reason:** Zero runtime overhead, no file reads in server hot path, security (creds not in deployable artifact)
- **Date:** 2026-03-19

### ADR-003: Connection pool (C), not per-request connections
- **Decision:** WOLF_DB_POOL_SIZE (default 10) pooled connections, mutex+cond_var
- **Reason:** Eliminate ~10-50ms auth handshake per request, hit sub-200ms target
- **Implementation:** wolf_pool_acquire → mysql_ping health-check → auto-reconnect
- **Date:** 2026-03-19

### ADR-004: Per-request memory arena
- **Decision:** Thread-local arena allocator, freed at end of each HTTP request
- **Reason:** Eliminate memory leaks in long-running HTTP server
- **Status:** ✅ Done
- **Date:** 2026-03-18

### ADR-005: Self-hosting roadmap
- **Decision:** Re-implement compiler in Wolf with Go as parity backup
- **Timeline:** Year 1 = Wolf-hosted, Year 1.5 = Go dropped
- **Date:** 2026-03-19

### ADR-006: Graceful shutdown via sigaction + drain loop
- **Decision:** SIGTERM/SIGINT → flag → drain in-flight → destroy pool → exit(0)
- **Reason:** Production deployments require zero dropped requests on restart
- **Implementation:** `wolf_shutdown_requested` flag + `wolf_active_requests` atomic counter; drain busy-waits with 100ms sleeps up to `(WOLF_REQUEST_TIMEOUT_SEC + 2) * 10` iterations
- **Date:** 2026-03-20

### ADR-007: Request timeout via SO_RCVTIMEO
- **Decision:** Set socket receive timeout on each accepted client fd
- **Reason:** Slow/malicious clients should not hold worker threads indefinitely
- **Default:** 30 seconds, configurable via `-DWOLF_REQUEST_TIMEOUT_SEC=N`
- **Date:** 2026-03-20

## Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| Request latency (p50) | < 200ms | Unknown (load test pending) |
| Request latency (p99) | < 500ms | Unknown |
| DB query (simple SELECT) | < 10ms | ~10-50ms (pool fixes this) |
| Binary startup time | < 5ms | Unknown |
| Compile time (hello.wolf) | < 1s | ~2-3s (LLVM overhead) |
| Graceful shutdown drain | < 32s | ✅ Guaranteed by drain loop |

## Known Bugs / Tech Debt

| # | Issue | Priority | Status |
|---|-------|----------|--------|
| 1 | MSSQL mock — no real FreeTDS implementation | 🟡 Medium | Next sprint |
| 2 | JWT stub — uses fake HMAC (sha256 djb2 hash) | 🟡 Medium | Future |
| 3 | wolf_db_pool_destroy drain is busy-wait | 🟢 Low | Acceptable |
| 4 | Surrogate pair unicode (\uD83D\uDE00) not decoded | 🟢 Low | Future |
| 5 | wolf_sprintf only handles one %s argument | 🟢 Low | Cleanup |
| 6 | MSSQL unused variable warnings in C compiler | 🟢 Low | Cleanup |

## Shutdown Sequence (reference)

```
SIGTERM/SIGINT received
    → wolf_signal_handler() sets wolf_shutdown_requested = 1
    → accept() returns EINTR, loop exits
    → New connections get HTTP 503
    → Drain loop: nanosleep(100ms) × up to (timeout+2)×10 iterations
    → wolf_db_pool_destroy():
        → WOLF_DB_CLOSE() all slots
        → pthread_cond_broadcast() unblocks pool waiters
        → free credential strings
    → mysql_library_end()  (MySQL driver only)
    → fprintf(stderr, "[Wolf] Shutdown complete.")
    → exit(0)
```

**Note on Testability:** The server readiness message ("🐺 Wolf HTTP Server running...") is followed by an `fflush(stdout)` to ensure external test runners can immediately detect the "up" state.

## Local Development Workflow

To ensure stability and consistent CI passing, every feature addition or bug fix should follow this local loop:

1. **Work**: Implement the code changes.
2. **Test**: Run `./test_local.sh`. 
   - This performs `gofmt` checks, builds the binary, runs all unit/integration tests (with coverage), and executes the e2e test suite.
   - Use `./test_local.sh --no-slow` to skip long-running Python/ML environment setups during iterative work.
3. **Commit & Push**: Once `test_local.sh` passes, run `./fix_and_push.sh`.
   - This script standardizes the commit message and ensures all required files (runtime, tests, CI configs) are staged and pushed to `origin/main`.