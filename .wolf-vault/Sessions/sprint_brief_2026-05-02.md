# Sprint Brief — 2026-05-02

**Prepared by:** Compass  
**Session:** 20 (Wrap-up)

---

## 🐕 Bloodhound Report

**Test sweep result:** All `./internal/...` packages: ✅ PASS (9/9)  
**Open bugs:** None  
**New issues found:** None  

The codebase is in a clean state. No P0/P1 issues were detected. The architecture.md tech debt table still shows `wolf_dns_lookup blocks worker thread` as Medium — but this has now been resolved in this session. That entry should be updated in a future vault cleanup pass.

---

## ⚡ Forge Review — wolf_dns_lookup (New Implementation)

**Target:** server / embedded-linux  
**Freestanding compatible:** ⚠️ WITH STRIP  
**Blocker:** `pthread_create`, `pthread_cond_timedwait`, `malloc` — all OS-dependent  
**Strip path:** The new `wolf_dns_lookup` must remain wrapped inside `#ifndef WOLF_FREESTANDING`. The `#include <netdb.h>` is already outside the guard. For freestanding targets, DNS does not exist at the OS level — static IP tables must be used instead.

**Verdict:** ⚠️ CONDITIONAL  
The pthread-based timeout is correct for server targets. For `--freestanding`, `wolf_dns_lookup` must be a no-op stub returning `""`.

> **Recommendation:** Add `#ifndef WOLF_FREESTANDING` guard around the entire `wolf_dns_ctx_t` struct, `wolf_dns_worker`, and `wolf_dns_lookup` implementation. The `#include <netdb.h>` at the top should also be moved inside the guard.

---

## 🛡️ Sentinel Review — wolf_dns_lookup Timeout

**Reviewed change:** `feat(runtime): implement 2s timeout for wolf_dns_lookup using detached pthread`

| Check | Result |
|---|---|
| No unbounded `malloc` outside arena? | ⚠️ `malloc(sizeof(wolf_dns_ctx_t))` used — justified (ctx lifetime exceeds request arena) |
| No busy-wait loops? | ✅ `pthread_cond_timedwait` used correctly |
| No global state mutation? | ✅ Context is stack-allocated pointer, no globals touched |
| Thread safety? | ✅ Mutex protects `main_waiting` flag; detached worker frees ctx on timeout |
| Memory leak on timeout? | ✅ Worker detects `main_waiting == 0` and calls `free(ctx)` |
| Lock-free hot path? | ⚠️ Uses pthread mutex — acceptable for DNS (not in HTTP hot path) |

**Verdict:** ✅ APPROVED WITH NOTES  
The `malloc` for `wolf_dns_ctx_t` is acceptable — it cannot use the request arena because the detached thread outlives the request context. The `free()` in the worker thread handles cleanup correctly. One concern: if `pthread_mutex_destroy` races with the worker thread still holding the lock on the happy path, use the pattern of `pthread_cond_wait` + signal instead of destroying immediately after `pthread_mutex_unlock`. Current code is safe because the main thread destroys only after the worker sets `main_waiting = 0` and signals.

---

## 🧭 Compass — Sprint 20 Candidates

### Decision Matrix Applied

| Candidate | Roadmap Priority | Dependency Clear? | Complexity | Verdict |
|---|---|---|---|---|
| **Package System v2 `new` dispatch** | P0 | ✅ Yes | Medium | 🟢 GO |
| **DB eager loading (DB-02)** | P1 | ✅ Yes | Medium | 🟢 GO |
| **Binary size / tree-shaking** | P2 | ✅ Yes | High | 🔵 DEFER |

### Sprint 20 Brief

**Top Task: Package System v2 — Dynamic `new` Dispatch**
- **What:** `wolf___compiler_create_model` currently resolves class instantiation via a generated `strcmp` chain. This must be replaced with a dynamic registry (BSS hash map or linked list) seeded during autodiscovery so new classes from packages don't require compiler string table updates.
- **Files:** `internal/compiler/compiler.go`, `internal/emitter/llvm_emitter.go`, possibly `runtime/wolf_runtime.c`
- **Risk:** Medium — touches the autodiscovery pipeline. Sentinel must review.
- **Verdict:** 🟢 GO

**Second Task: DB Eager Loading (DB-02)**
- **What:** Implement `$this->db->with("relation")` eager loading to prevent N+1 queries.
- **Files:** `stdlib/db.go` (if exists), `runtime/wolf_runtime.c`
- **Risk:** Medium — new runtime API surface
- **Verdict:** 🟢 GO (next sprint)

**Deferred: Binary Size**
- At 9.2MB (target 8MB), the overage is from libcurl static linking. Tree-shaking would require musl libc or a custom curl build. Defer until after Package System v2 ships.
- **Verdict:** 🔵 DEFER

---

## 🆕 New Agent: Scout

Scout has been created at `.wolf-vault/Agents/scout.md`. It is the Wolf Pack's documentation writer — invoked via `/scout` whenever stdlib or runtime functions change. Scout always uses Wolf syntax, always includes a code example, and only updates what actually changed.

**Trigger:** `/scout` — "Document this changed stdlib file."

---

## Next Immediate Task
**Package System v2 `new` dispatch** — replace the generated `strcmp` chain in `wolf___compiler_create_model` with a dynamic hash-based class registry.
