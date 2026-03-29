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
| Pool destroy | wolf_db_pool_destroy() | ✅ Done |
| Redis | wolf_redis_* | ✅ Done (real hiredis) |
| HTTP server | wolf_http_* | ✅ Done |
| Graceful shutdown | SIGTERM/SIGINT handler + cond_var drain | ✅ Done |
| Request timeout | SO_RCVTIMEO + HTTP 408 | ✅ Done |
| SIGPIPE guard | signal(SIGPIPE, SIG_IGN) | ✅ Done |
| File uploads | multipart/form-data → wolf_http_req_file() | ✅ Done (session 5) |
| wolf_file_save | Base64 decode + binary fwrite | ✅ Done (session 5) |
| Metal-Ready | #ifdef WOLF_FREESTANDING guards | ✅ Done (session 5) |
| JWT | wolf_jwt_* | ✅ Done (HMAC-SHA256) |
| Memory arena | wolf_req_alloc / wolf_req_arena_flush | ✅ Done |
| Crypto | OpenSSL + libsodium | ✅ Done |
| MSSQL | wolf_db_* (MSSQL path) | 🔴 Mock only |
| WebSocket | wolf_ws_* | ✅ Done |
| Date / Time | wolf_date_* | ✅ Done (Session 7) |

## Key Design Decisions

### ADR-001: LLVM IR as compilation target (not Go transpile)
- **Decision:** Emit LLVM IR directly, not Go/C source
- **Reason:** True native performance, no GC pauses, sub-200ms request latency goal
- **Date:** Early 2026

### ADR-002: wolf.config (INI format, like php.ini)
- **Decision:** INI-style config, values baked at compile time as -D flags
- **Reason:** Zero runtime overhead, no file reads in server hot path
- **Date:** 2026-03-19

### ADR-003: Connection pool (C), not per-request connections
- **Decision:** WOLF_DB_POOL_SIZE (default 10) pooled connections, mutex+cond_var
- **Reason:** Eliminate ~10-50ms auth handshake per request
- **Date:** 2026-03-19

### ADR-004: Per-request memory arena
- **Decision:** Thread-local arena allocator, freed at end of each HTTP request
- **Reason:** Eliminate memory leaks in long-running HTTP server; zero-cost in hot path
- **Date:** 2026-03-18

### ADR-005: Self-hosting roadmap
- **Decision:** Re-implement compiler in Wolf with Go as parity backup
- **Timeline:** Year 1 = Wolf-hosted, Year 1.5 = Go dropped
- **Date:** 2026-03-19

### ADR-006: Graceful shutdown via sigaction + drain loop
- **Decision:** SIGTERM/SIGINT → flag → `pthread_cond_wait` drain → destroy pool → exit(0)
- **Date:** 2026-03-20

### ADR-007: Request timeout via SO_RCVTIMEO
- **Decision:** Set socket receive timeout on each accepted client fd
- **Default:** 30 seconds, configurable via `-DWOLF_REQUEST_TIMEOUT_SEC=N`
- **Date:** 2026-03-20

### ADR-008: File uploads via arena-allocated multipart parser
- **Decision:** Parse multipart boundary in-process (no temp files, no external libs). Binary data stored directly in request arena. Returned to Wolf script as `{"name","type","size","data"}` JSON with binary-safe base64 encoding.
- **Reasoning:** Keeps the zero-heap-alloc invariant in the HTTP hot path. Base64 overhead is acceptable for the typical image/PDF upload sizes.
- **API:** `wolf_http_req_file(req_id, "field")` → JSON; `wolf_file_save(path, b64)` → persists
- **Limit:** 8 files per request (`WOLF_MAX_UPLOADS`)
- **Date:** 2026-03-25

### ADR-009: WOLF_FREESTANDING core/server split
- **Decision:** Server-only OS deps (socket, pthread, netinet, arpa) guarded by `#ifndef WOLF_FREESTANDING`. Guard covers the entire HTTP server block AND Request/Response API.
- **Core always compiled:** arena, strings, maps, arrays, math, crypto, file I/O, JSON
- **Goal:** Allow the core runtime to compile for bare-metal / OS-less targets with `-DWOLF_FREESTANDING -ffreestanding`
- **Verification:** `clang -DWOLF_FREESTANDING -fsyntax-only runtime/wolf_runtime.c` → zero errors
- **Date:** 2026-03-25

### ADR-010: libcurl for native HTTP Client
- **Decision:** Use `libcurl` easy interface for synchronous GET/POST from Wolf scripts.
- **Reasoning:** Industry standard, handles TLS/SSL, redirects, and complex proxies out-of-the-box.
- **Date:** 2026-03-26

### ADR-011: Hand-rolled RFC 6455 WebSocket framing
- **Reasoning:** Maintains "Metal-Ready" (Zero-OS-Dependency) core. Avoids the 500KB+ overhead of `libwebsockets`.
- **Date:** 2026-03-26

### ADR-012: ISO-8601 Temporal API focus
- **Decision:** Optimize all the stdlib date primitives (`date_to_iso`, `date_create`) for strict ISO-8601 compatibility.
- **Reasoning:** Industry standard for JSON interchange; eliminates timezone ambiguity in distributed worker models.
- **Date:** 2026-03-26

## Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| Request latency (p50) | < 200ms | Unknown (load test pending) |
| Request latency (p99) | < 500ms | Unknown |
| DB query (simple SELECT) | < 10ms | ~10-50ms (pool fixes this) |
| Binary startup time | < 5ms | Unknown |
| Binary size | ≤ 8MB | 4.5MB ✅ |
| Compile time (hello.wolf) | < 1s | ~2-3s (LLVM overhead) |
| Graceful shutdown drain | < 32s | ✅ |

## Known Bugs / Tech Debt

| # | Issue | Priority | Status |
|---|-------|----------|--------|
| 1 | MSSQL mock — no real FreeTDS implementation | 🟡 Medium | Next sprint |
| 2 | wolf_http_req_file limited to 8 uploads/req | 🟢 Low | Increase WOLF_MAX_UPLOADS if needed |
| 3 | wolf_file_save doesn't create parent dirs | 🟢 Low | Use wolf_dir_exists first |
| 4 | Surrogate pair unicode (\\uD83D\\uDE00) not decoded | 🟢 Low | Future |


## Shutdown Sequence (reference)

```
SIGTERM/SIGINT received
    → wolf_signal_handler() sets wolf_shutdown_requested = 1
    → accept() returns EINTR, loop exits
    → New connections get HTTP 503
    → pthread_cond_wait drain (blocks until wolf_active_requests == 0)
    → wolf_db_pool_destroy():
        → WOLF_DB_CLOSE() all slots
        → pthread_cond_broadcast() unblocks pool waiters
        → free credential strings
    → mysql_library_end()  (MySQL driver only)
    → fprintf(stderr, "[Wolf] Shutdown complete.")
    → exit(0)
```

## Local Development Workflow

1. **Work**: Implement changes.
2. **Test**: `./test_local.sh` or `./test_local.sh --no-slow` for iterative work.
3. **Commit & Push**: `./fix_and_push.sh`