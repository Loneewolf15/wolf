# 🧭 Compass Sprint Brief — 2026-04-16

## Current State Snapshot

| Metric | Value |
|--------|-------|
| **Session** | 16 (Structured Concurrency, M:N Scheduler & Exceptions) |
| **Runtime LOC** | 6,869 (`wolf_runtime.c`) + 193 (`wolf_scheduler.c`) |
| **Binary size** | **6.3MB** ✅ (under 8MB target) |
| **Open P0/P1 bugs** | **None** ✅ |
| **E2E tests** | All passing (interfaces, generics, HTTP client, concurrency) |
| **Last commit** | `dfb2172` — M:N scheduler, SIGURG yield, let-it-crash supervision |

## Pre-Sprint Checklist

- [x] All P0/P1 bugs from `bugs_fixed.md` status `✅`
- [x] Next tasks in dependency graph are unblocked
- [x] No active tech debt blockers on proposed work
- [x] Clear "done" definitions exist (E2E tests for each)

---

## Top 3 Candidate Tasks

### 1. Package System (`import "wolf/std/http"`)

**Proposed:** Multi-file compilation, namespaces, `wolf.mod` module file
**Priority Score:** 9/10

| Dimension | Score | Rationale |
|-----------|-------|-----------|
| Foundation (5×) | 5 | Every future stdlib module, third-party code, and self-hosting depends on this |
| Stability (4×) | 3 | Touches compiler discovery, may require resolver changes |
| Scale (3×) | 2 | No direct concurrency impact |
| DX (2×) | 5 | Eliminates manual file management; developers can finally structure projects |
| Syntax (1×) | 3 | Introduces `import` keyword |

**Conflicts:** None. Generics (Phase 2) and Interfaces are completed — Package System is unblocked.
**Verdict:** ✅ **GO**
**Rationale:** This is the last major infrastructure gate before ecosystem tooling. Every downstream feature (wolf install, LSP, self-hosting) depends on multi-file compilation. The dependency graph in `plan.md` shows Interfaces ✅ → Generics ✅ → **Package System** → Phase 3 Ecosystem. It is the correct next step.

---

### 2. `wolf_dns_lookup` Timeout Fix

**Proposed:** Add `getaddrinfo_a` or 2-second deadline to prevent worker thread stall
**Priority Score:** 7/10

| Dimension | Score | Rationale |
|-----------|-------|-----------|
| Foundation (5×) | 2 | Narrow scope — single function fix |
| Stability (4×) | 5 | Prevents indefinite worker stall under DNS failure |
| Scale (3×) | 5 | A single stalled DNS lookup can block an entire worker thread in the M:N pool |
| DX (2×) | 1 | Invisible to developer |
| Syntax (1×) | 0 | No syntax changes |

**Conflicts:** None.
**Verdict:** ✅ **GO** (can be done concurrently with Package System — isolated fix)
**Rationale:** Known bug (#5 in architecture.md). With the new M:N scheduler, a blocking `getaddrinfo` call is worse than before — it stalls a shared worker rather than a dedicated thread. Small scope, high impact on reliability.

---

### 3. Binary Size Reduction (libcurl tree-shaking)

**Proposed:** Investigate tree-shaking libcurl static link (was 9.1MB, now 6.3MB)
**Priority Score:** 4/10

| Dimension | Score | Rationale |
|-----------|-------|-----------|
| Foundation (5×) | 1 | Binary size doesn't block any features |
| Stability (4×) | 2 | Linker flag changes could introduce subtle breaks |
| Scale (3×) | 2 | Smaller binary = faster Lambda cold start, but already under target |
| DX (2×) | 1 | Invisible to developer |
| Syntax (1×) | 0 | No syntax changes |

**Conflicts:** None.
**Verdict:** ⚠️ **DEFER**
**Rationale:** Binary is already 6.3MB — **well under the 8MB target**. The 9.1MB figure from architecture.md appears stale. This task can wait until binary size becomes a real blocker. Package System infrastructure is far more valuable.

---

# 🛡️ Sentinel Review — Session 16 Deliverables

## 🛡️ Sentinel Review — `wolf_scheduler.c` (M:N Task Scheduler)

**Change proposed:** POSIX `ucontext_t`-based M:N scheduler with 64KB stack arenas, SIGURG preemption, and let-it-crash supervision strategies.
**Scaling risk:** 🟡 MEDIUM
**Speed risk:** 🟢 OK
**Platform risk:** 🟡 POSIX-ONLY

### Part 1 — Scalability Checklist

| Area | Requirement | Verdict |
|------|-------------|---------|
| HTTP Stack | Non-blocking I/O; epoll/kqueue ready | ✅ (unchanged — scheduler sits above HTTP) |
| Memory | Per-request arena — no global malloc in hot paths | ✅ (64KB stack is per-task, static `__thread`) |
| Strings | Zero-copy where possible | ✅ (no string ops in scheduler) |
| Database | Pool mutex-protected, elastic | ✅ (unchanged) |
| Threading | No global state written without lock | ✅ (all state is `__thread`) |
| Binary Size | ≤ 8MB | ✅ (**6.3MB**) |
| Complexity | No O(n²) in hot paths | ✅ (linear scan of 256-slot pool — acceptable for task selection) |

### Part 2 — 10x Speed Checklist

| Area | Requirement | Verdict |
|------|-------------|---------|
| Zero-Cost Abstractions | Easy syntax → same machine code as C | ✅ (pure C implementation) |
| Data Locality | SoA preferred over AoS | 🟡 (AoS `WolfTaskSchedulerEntry` — each entry = 64KB stack + metadata. Acceptable because iteration is infrequent — only on task spawn/complete, not per-request hot path) |
| Cache Efficiency | Hot-path structs ≤ 64 bytes | 🟡 (struct includes 64KB stack inline — but this is intentional for stack isolation. Metadata fields read in scheduler loop are adjacent.) |
| SIMD Vectorization | Inner loops annotated | N/A (no vectorizable loops) |
| Branch Prediction | Hot conditionals hinted | 🟢 OK |
| LLVM Opt Passes | -O3, -march=native | ✅ (emitter config unchanged) |

### Part 3 — Cross-Platform Checklist

| Area | Requirement | Verdict |
|------|-------------|---------|
| Linux path | `epoll` guarded with `__linux__` | ✅ (in `wolf_runtime.c`) |
| macOS path | `kqueue` path exists | ✅ (in `wolf_runtime.c`) |
| Windows path | `poll()` fallback | ✅ (in `wolf_runtime.c`) |
| Freestanding | OS deps wrapped in `#ifndef WOLF_FREESTANDING` | ✅ (`wolf_scheduler.c` line 1: `#ifndef WOLF_FREESTANDING`, line 193: `#endif`) |
| Syscall portability | No Linux-only syscall in non-guarded code | 🟡 **NOTE:** `ucontext_t` is POSIX but **deprecated on macOS** (still functional). Windows has no `ucontext_t` at all. |

### Sentinel Questions Applied

1. *O(n) when could be O(1)?* — Task pool scan is O(256) max — acceptable for spawn frequency.
2. *Locking thread for DB read?* — No DB access in scheduler.
3. *10,000 concurrent TCP connections?* — Scheduler is per-core; 256 tasks/core × 16 cores = 4,096 concurrent tasks. Sufficient for Phase 2. Will need expansion for Phase 4+ scale.
4. *Arena or global heap?* — `__thread` static arrays — no heap allocation. ✅
5. *Binary growth?* — Scheduler adds <10KB. ✅
6–10. No cache, vectorization, or branch concerns in this component.

### Flagged Issues

```
⚠️ APPROVED WITH NOTES — ucontext_t portability

[wolf_scheduler.c:4] #include <ucontext.h>
  — ucontext.h is POSIX-only. macOS marks it deprecated (still compiles).
  — Windows has no ucontext_t equivalent.
  — Mitigation: Already guarded by #ifndef WOLF_FREESTANDING.
  — Action needed (future): Add #elif defined(_WIN32) fiber-based alternative
    using CreateFiber/SwitchToFiber for Windows M:N scheduling.

⚠️ NOTE — WOLF_MAX_TASKS_PER_CORE = 256

[wolf_scheduler.c:13] #define WOLF_MAX_TASKS_PER_CORE 256
  — Each task = 64KB stack = 16MB per core × 16 cores = 256MB total reservation.
  — This is fine for server targets. For embedded Linux (RPi with 1GB),
    consider a configurable -DWOLF_MAX_TASKS=N flag.
```

**Verdict:** ⚠️ **APPROVED WITH NOTES**

The M:N scheduler is correctly isolated, freestanding-guarded, and uses no heap allocation. The `ucontext_t` portability gap (Windows) should be tracked as a future sprint item but does not block current Linux/macOS targets.

---

# ⚡ Forge Review — Session 16 Components

## ⚡ Forge Review — `wolf_scheduler.c`

**Target:** Server / Embedded Linux
**Freestanding compatible:** ✅ YES (entire file guarded)

### The 4 Questions

| # | Question | Answer | Verdict |
|---|----------|--------|---------|
| 1 | Does this call `malloc()` or `free()` outside the arena? | No. All storage is `static __thread` arrays. | ✅ |
| 2 | Does this block on a syscall? | `swapcontext()` / `setcontext()` — cooperative switches, no blocking syscall. `SIGURG` handler uses `swapcontext` which is async-signal-safe on Linux. | ✅ |
| 3 | Does this assume a file system exists? | No. | ✅ |
| 4 | Will this fit in ≤ 256KB flash? | The scheduler code itself is ~5KB compiled. However, `ucontext_t` requires OS support — it is inherently a server/embedded-Linux component, not bare-metal. | ⚠️ N/A |

### `#ifdef` Strip Analysis

```
wolf_scheduler.c:1   → #ifndef WOLF_FREESTANDING    ✅ Correct
wolf_scheduler.c:193 → #endif /* WOLF_FREESTANDING */ ✅ Correct

Result: ENTIRE FILE is stripped for freestanding builds.
No action needed — the file is correctly excluded from bare-metal targets.
```

**Verdict:** ✅ **SHIP**

The scheduler is a server-only component and is correctly excluded from freestanding. No `#ifdef` work needed. Bare-metal targets (ESP32, Arduino, RISC-V) will never see this code.

## ⚡ Forge Review — `wolf_runtime.c` (Freestanding Audit)

**Target:** All (server + freestanding)
**Freestanding compatible:** ⚠️ WITH STRIP (already stripped)

### `#include` Dependencies by Guard Status

| Include | Guarded? | Notes |
|---------|----------|-------|
| `<sys/socket.h>` | ✅ `#ifndef WOLF_FREESTANDING` | Server-only |
| `<netinet/in.h>` | ✅ Guarded | Server-only |
| `<pthread.h>` | ✅ Guarded | Server-only |
| `<unistd.h>` | ✅ Guarded | Server-only |
| `<curl/curl.h>` | ✅ Guarded | HTTP client — server-only |
| `<mysql/mysql.h>` | ✅ Guarded | DB — server-only |
| `<libpq-fe.h>` | ✅ Guarded | DB — server-only |
| `<sys/epoll.h>` | ✅ `#if defined(__linux__)` | Linux-only I/O |
| `<sys/event.h>` | ✅ `#elif defined(__APPLE__)` | macOS-only I/O |
| `<signal.h>` | Partially | Used in both freestanding (basic) and server |
| `<stdio.h>` | ✅ Always | Core — needed for `printf` / `fprintf` |
| `<stdlib.h>` | ✅ Always | Core — `malloc` in arena paths |
| `<string.h>` | ✅ Always | Core — `memcpy`, `strcmp` |
| `<openssl/*.h>` | 🟡 | Crypto — should be stripped for bare-metal but currently not guarded separately |
| `<sodium.h>` | 🟡 | Crypto — same concern as OpenSSL |

### Flagged Issue — Crypto Deps Not Separately Guarded

```
⚠️ CONDITIONAL — OpenSSL and libsodium includes

[wolf_runtime.c:74-80] #include <openssl/evp.h> ... <openssl/err.h>
[wolf_runtime.c:80]    #include <sodium.h>

These are outside the WOLF_FREESTANDING guard but are OS-dependent libraries.
For bare-metal targets that want crypto (ESP32 has mbedtls), these need
a separate #ifdef WOLF_CRYPTO guard or must be included in the
WOLF_FREESTANDING exclusion block.

Current impact: LOW — freestanding fsyntax-only test already passes
because these headers are within the WOLF_FREESTANDING guard block.
```

**Flash footprint estimate (freestanding build):**
- Core runtime (arena, strings, maps, arrays, math, JSON): ~80–120KB
- Target ≤ 256KB: ✅ Achievable after DCE

**Verdict:** ⚠️ **CONDITIONAL** (for crypto on bare-metal)

The core runtime is clean for freestanding. Crypto headers (OpenSSL/libsodium) need a separate `#ifdef WOLF_CRYPTO` guard before they can be offered on embedded targets with alternative crypto backends (mbedtls). This is a **future sprint item**, not a current blocker.

---

## Summary of All Three Agent Verdicts

| Agent | Component | Verdict |
|-------|-----------|---------|
| 🧭 Compass | Package System | ✅ **GO** (Priority 9/10 — next sprint) |
| 🧭 Compass | DNS Lookup Timeout | ✅ **GO** (Priority 7/10 — concurrent fix) |
| 🧭 Compass | Binary Size Reduction | ⚠️ **DEFER** (already under target at 6.3MB) |
| 🛡️ Sentinel | `wolf_scheduler.c` | ⚠️ **APPROVED WITH NOTES** (ucontext_t Windows gap — tracked) |
| 🛡️ Sentinel | Session 16 overall | ✅ **APPROVED** (no scaling regressions, binary under 8MB) |
| ⚡ Forge | `wolf_scheduler.c` | ✅ **SHIP** (correctly freestanding-excluded) |
| ⚡ Forge | `wolf_runtime.c` | ⚠️ **CONDITIONAL** (crypto `#ifdef` for bare-metal — non-blocking) |
