# Handoff — 2026-07-25 (Session 38)

## Where We Left Off

Session 38 fully resolved both open handoff items from Session 37 and delivered Phase 5 HTTP Smuggling Defenses. The runtime is now security-hardened across three vectors: rate limiting correctness, shutdown connection handling, and HTTP request smuggling.

## Commits This Session
```
5196092 fix(security): ADR-030 global rate limiter + io_uring shutdown 503 + HTTP smuggling defenses (Phase 5)
1affc7f fix: resolve stale compiler cache and verify bucket aliasing bug  [previous session]
```

## What Was Accomplished

### BUG-096 — Rate Limiter (CRITICAL, FIXED)
The rate limiter was per-core, allowing `N × WOLF_RATE_RPS` per IP. Fixed by moving to a single global `WolfEngine.ratelimit` instance shared by all cores (ADR-030). Effective ceiling is now exactly `WOLF_RATE_RPS` on any machine regardless of core count.

### BUG-097 — io_uring Shutdown RST (FIXED)
`on_accept_complete` now checks the shutdown flag first and returns `503 + FIN` instead of `RST` for connections arriving during the shutdown window (ADR-031). This eliminates the 1-in-20,665 RST that showed up in the Session 37 TSAN gauntlet.

### Phase 5 — HTTP Request Smuggling Defenses (COMPLETE)
5 rules implemented in `wolf_http_parser.c` (ADR-032):
- R1: CL + TE coexistence → 400
- R2: TE:chunked sets `ctx->ignore_content_length`
- R3: Non-canonical TE values (`xchunked`, `chunked, trailers`) → 400
- R4: Duplicate CL headers (count ≥ 2) or negative CL → 400
- R5: Bare `\r` not followed by `\n` in raw header bytes → 400
- Unit test: `bench/test_smuggling.c` — 21/21 PASS

## Tests Status
- `go test ./internal/... -count=1` — **13/13 packages PASS** ✅
- `bench/test_smuggling.c` — **21/21 PASS** ✅
- E2E pre-existing failures (NOT regressions):
  - `TestWIRDump`, `TestWIREmitLLVM`, `TestWIRBothFlags` — require `wolf_out/main` native binary
  - `TestWebSocketEcho` — flaky timing (confirmed pre-existing on baseline commit)

## Next Immediate Tasks (Phase 5 continues)

1. **E2E test for smuggling defense (`55_smuggling_defense.wolf`)** — parked this session. Need a Wolf server that intentionally tests the 400 responses to smuggling probes via the E2E harness. Low priority since C-level unit test already covers all 5 rules.

2. **Phase 5B — HTTP/1.1 Pipelining Defenses** — after smuggling defenses, the next attack surface is connection-level request pipelining (overlapping requests on keep-alive connections). Wolf currently does not pipeline but should explicitly reject pipelined requests.

3. **wolf install package registry** — the next major feature milestone per the execution plan.

4. **Wolf LSP + VS Code foundation** — the other unblocked task from the sprint.

## Relevant Files Modified This Session
- `runtime/wolf_http_engine.h` — ADR-030: WolfCore/WolfEngine struct changes, ADR-031 comments
- `runtime/wolf_http_engine.c` — rate limiter init/destroy refactor, shutdown guard, debug print removal
- `runtime/wolf_http_parser.c` — Phase 5 smuggling defense suite (5 rules + validate_raw_crlf)
- `bench/test_smuggling.c` — 21-case unit test suite (NEW)
- `.wolf-vault/RnD/architecture.md` — ADR-030, ADR-031, ADR-032 entries
- `.wolf-vault/RnD/bugs_fixed.md` — BUG-096, BUG-097, BUG-098 entries, ledger updated to 67

## Open Items (None Critical)
- Pre-existing `TestWIR*` failures need `wolf_out/main` native binary to be built — not a regression.
- BUG-052 (wolf_qb_where NULL conn) remains open per previous session decision.
