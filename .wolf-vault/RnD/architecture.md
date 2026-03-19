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
| Redis | wolf_redis_* | ⚠️ In-memory mock only |
| HTTP server | wolf_http_* | ✅ Done |
| JWT | wolf_jwt_* | ✅ Done |
| Memory arena | wolf_req_alloc | 🔴 In progress |

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

### ADR-004: Per-request memory arena (planned)
- **Decision:** Thread-local arena allocator, freed at end of each HTTP request
- **Reason:** Eliminate memory leaks in long-running HTTP server
- **Status:** 🔴 In progress

### ADR-005: Self-hosting roadmap
- **Decision:** Re-implement compiler in Wolf with Go as parity backup
- **Timeline:** Year 1 = Wolf-hosted, Year 1.5 = Go dropped
- **Date:** 2026-03-19

## Performance Targets

| Metric | Target | Current |
|--------|--------|---------|
| Request latency (p50) | < 200ms | Unknown (needs benchmark) |
| Request latency (p99) | < 500ms | Unknown |
| DB query (simple SELECT) | < 10ms | ~10-50ms (pool fixes this) |
| Binary startup time | < 5ms | Unknown |
| Compile time (hello.wolf) | < 1s | ~2-3s (LLVM overhead) |

## Known Bugs / Tech Debt

| # | Issue | Priority | Owner |
|---|-------|----------|-------|
| 1 | Memory leaks — malloc with no free in HTTP handler | 🔴 High | In progress |
| 2 | Redis mock — not real hiredis | 🟡 Medium | Next sprint |
| 3 | db_mutex warning (wolf_db_bind) | 🟢 Low | Cleanup |
| 4 | Surrogate pair unicode (\uD83D\uDE00) not decoded | 🟢 Low | Future |
| 5 | wolf_db_pool_destroy() for graceful shutdown | 🟢 Low | Future |
