## Sprint Brief — 2026-04-15

**Session context:** Phase 2 active. STDLIB-06 (HTTP Client + URL Utilities) complete. All `internal/...` tests green. Binary 9.1MB.

---

### Candidate 1 — Apply P3 Bloodhound Fix: `wolf_http_req_client_ip` guard

**Proposed:** Wrap `const char* wolf_http_req_client_ip(int64_t req_id);` forward declaration (line 162, `wolf_runtime.c`) in `#ifndef WOLF_FREESTANDING`.
**Priority Score:** 7/10
**Conflicts:** None — one-line change, no regressions possible.
**Verdict:** ✅ GO

**Rationale:** This is the only open bug from `bugs_fixed.md` and the handoff. It is a carry-over from last session. Per Compass rule 2, open bugs must be cleared before new feature work. This is P3 (bare-metal build break), not P0/P1, but it is trivially small and unblocks the Forge freestanding audit. Estimated time: < 5 minutes. Ship it first.

---

### Candidate 2 — Interfaces / Traits (Phase 4, LANG-04)

**Proposed:** Implement `interface` / `implements` keywords end-to-end: token → AST node → parser → resolver → LLVM vtable dispatch.
**Priority Score:** 8/10
**Conflicts:** None — Closures ✅, Error Handling ✅, HTTP Client ✅ all shipped. Interfaces is the next dependency-unblocked Phase 4 item.
**Verdict:** ✅ GO

**Rationale:** All prerequisite Phase 2 stdlib work (STDLIB-01 through STDLIB-06, STDLIB-08) is done. Interfaces are the highest-DX language feature remaining and are required before Generics can land. The prior implementation session (conv `1ecb6412`) produced a complete design — vtable struct (`i64, ptr, ptr…`) with LLVM opaque ptrs. This sprint should implement lexer tokens, IR nodes, parser, resolver type-check, and LLVM dispatch. Done definition: `39_interfaces.wolf` E2E test passes.

---

### Candidate 3 — STDLIB-07: File System Completion (copy, move, scan_dir, path_join)

**Proposed:** Complete file system stdlib — missing `copy`, `move`, `scan_dir`, `path_join`.
**Priority Score:** 5/10
**Conflicts:** None — but lower DX impact than interfaces.
**Verdict:** ⚠️ DEFER

**Rationale:** Lower strategic value than interfaces this sprint. Language completeness (interfaces → generics) is the critical path to Phase 3 and the public launch. File system completeness is valuable but non-blocking for any downstream feature. Queue behind interfaces.

---

## Session Order

1. ✅ GO — Fix P3 Bloodhound bug (< 5 min, clear the ledger)
2. ✅ GO — Implement Interfaces / Traits end-to-end
3. ⚠️ DEFER — STDLIB-07 file system (next session if time allows)
