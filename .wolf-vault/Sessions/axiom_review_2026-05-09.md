# 🧠 AXIOM Security & Technical Audit — 2026-05-09

> Full-stack audit: Security, Memory, Concurrency, I/O, SQL, Crypto.
> Evidence-based. Every finding cites file:line.

---

## FINDING 1 — 🔴 CRITICAL: `wolf_arena_alloc` inner-slab overflow leaks silently

**Area:** Memory Management | **Affects:** `runtime/wolf_http_engine.c:351`

When a single allocation inside an acquired arena exceeds `WOLF_ARENA_SLAB_SIZE`, `wolf_arena_alloc` falls back to `calloc(1, size)`. That pointer is returned to the caller but **is never registered**. When `wolf_arena_reset()` runs at request end, the slab pointer resets to `pos=0` — but the overflow `calloc` is **permanently leaked**.

```c
// wolf_http_engine.c:351 — the comment is lying
if (arena->pos + size > arena->cap) {
    /* Overflow — fall back to malloc (tracked separately) */
    return calloc(1, size);   // ← tracked where?
}
```

**Fix:** Add `void** overflows; int overflow_count` to `WolfArena`. In `wolf_arena_alloc`, when falling back to `calloc`, append to the overflow list. In `wolf_arena_reset`, iterate and `free` each overflow pointer before reset.

**Verdict:** 🚫 ARCHITECTURALLY UNSOUND

---

## FINDING 2 — 🔴 CRITICAL: MSSQL `wolf_db_bind` is a SQL injection stub

**Area:** Security | **Affects:** `runtime/wolf_runtime.c:1498`

The MSSQL branch does zero escaping — raw `strcpy` of unvalidated user input wrapped in single quotes:

```c
#elif defined(WOLF_DB_MSSQL)
    char *escaped = wolf_req_alloc(strlen(value) * 2 + 1);
    strcpy(escaped, value);   // ← no escaping
```

Input `"'; DROP TABLE users; --"` passes through untouched.

**Fix:** Implement single-quote doubling (correct MSSQL escaping):
```c
const char *src = value; char *dst = escaped;
while (*src) { if (*src == '\'') *dst++ = '\''; *dst++ = *src++; }
*dst = '\0';
```
Or gate with `#error "MSSQL not production-ready"` until a driver exists.

**Verdict:** 🚫 ARCHITECTURALLY UNSOUND

---

## FINDING 3 — 🔴 CRITICAL: WebSocket arena freed before WS poller uses it

**Area:** Memory + Concurrency | **Affects:** `runtime/wolf_http_engine.c:731–738`

After WebSocket handshake, the engine calls `wolf_core_free_ctx(ctx)` which resets the arena — but the WS poller still holds the fd and expects the connection state (`ctx->ws_key`, header strings) to remain alive:

```c
if (ctx->is_websocket) {
    wolf_engine_ws_handshake(ctx);
    wolf_core_free_ctx(ctx);   // arena freed HERE — ws_key now dangling
    /* Don't close fd — WS poller owns it */
    continue;
}
```

**Fix:** Before freeing `ctx`, copy all WS session state into a `malloc`'d `WolfWSSession` struct that the WS poller owns and frees on connection close. Never reset the arena while a live fd is pointing into it.

**Verdict:** 🚫 ARCHITECTURALLY UNSOUND

---

## FINDING 4 — 🟡 MEDIUM: JWT length comparison is a timing oracle

**Area:** Security | **Affects:** `runtime/wolf_runtime.c:4891–4895`

`wolf_hash_equals` short-circuits on length mismatch before reaching `sodium_memcmp`. An attacker can distinguish "wrong length" from "wrong content" by measuring response time — leaking the expected signature length and reducing brute-force search space:

```c
if (kl != ul) return 0;   // ← timing branch — early exit leaks length
return sodium_memcmp(known, user, kl) == 0 ? 1 : 0;
```

**Fix:** XOR the lengths into the result and always run `sodium_memcmp` over `min(kl, ul)` bytes, accumulating the result without early exit.

**Verdict:** ⚠️ CONDITIONALLY SOUND

---

## FINDING 5 — 🟡 MEDIUM: Argon2id at `INTERACTIVE` — below OWASP threshold

**Area:** Security | **Affects:** `runtime/wolf_runtime.c:4904–4906`

`crypto_pwhash_OPSLIMIT_INTERACTIVE` = 3 iterations, 64 MB. OWASP 2023 recommends at minimum **MODERATE** (3 iterations, 256 MB) for server-side password hashing. On a RTX 4090, INTERACTIVE-preset Argon2id yields ~4 hashes/sec for an offline attacker against a leaked database.

**Fix:** Default to `crypto_pwhash_OPSLIMIT_MODERATE` / `crypto_pwhash_MEMLIMIT_MODERATE`. Expose `[crypto] argon2_strength = moderate` in `wolf.config` so high-traffic apps can tune down with explicit intent.

**Verdict:** ⚠️ CONDITIONALLY SOUND

---

## FINDING 6 — 🟡 MEDIUM: Arena pool scan has no memory fence at init

**Area:** Concurrency | **Affects:** `runtime/wolf_http_engine.c:328`

`in_use = 0` on arena init is a plain store (main thread). The worker thread reads `in_use` without an `__ATOMIC_ACQUIRE` load. On weakly-ordered CPUs (ARM64), the worker can see a stale `in_use = 1` and skip all arenas, forcing an unnecessary overflow malloc for every request until the CPU cache coherence catches up.

**Fix:** `__atomic_store_n(&pool->arenas[i].in_use, 0, __ATOMIC_RELEASE)` on init; `__atomic_load_n(&pool->arenas[i].in_use, __ATOMIC_ACQUIRE)` on scan. One line per site.

**Verdict:** ⚠️ CONDITIONALLY SOUND

---

## FINDING 7 — 🟡 MEDIUM: SQL layer is string interpolation, not real prepared statements

**Area:** Security | **Affects:** `runtime/wolf_runtime.c:1486–1516`

`wolf_db_prepare` does not call `mysql_stmt_prepare`. It stores the SQL string and `wolf_db_bind` manually escape-interpolates values via `wolf_internal_str_replace`. This is PHP 2003 pattern. String escaping has a documented charset confusion attack vector: if the connection character set is not pinned to `utf8mb4`, multi-byte sequences can break `mysql_real_escape_string` and re-open injection.

**Immediate mitigation:** Verify `mysql_set_character_set(conn, "utf8mb4")` is called after every connection open (check the pool init path).  
**Full fix:** Implement the `MYSQL_STMT` / `mysql_stmt_bind_param` path for true server-side parameterisation.

**Verdict:** ⚠️ CONDITIONALLY SOUND

---

## FINDING 8 — 🟢 LOW: `wolf_encrypt` derives symmetric key with plain SHA-256

**Area:** Security | **Affects:** `runtime/wolf_runtime.c:4934–4938`

A user-supplied password is hashed with a single SHA-256 pass to derive the XSalsa20-Poly1305 key. SHA-256 is not a KDF — it provides no memory hardness, no salting, and no iteration. A 6-char password with SHA-256 has ~39 bits of effective entropy regardless of the output size.

**Fix:** If key is user-provided: use `crypto_pwhash` (Argon2id) to derive the 32-byte key. If key is a machine secret (env var): use HKDF (`crypto_auth_hmacsha256`) to stretch it.

**Verdict:** ⚠️ CONDITIONALLY SOUND

---

## FINDING 9 — 🟢 LOW: OOM path calls `pthread_exit` — held locks never released

**Area:** Concurrency | **Affects:** `runtime/wolf_runtime.c:884`

On `malloc` failure, `wolf_req_alloc` calls `pthread_exit(NULL)`. If the handler holds a DB pool slot when OOM occurs, that slot is never returned — the pool permanently loses one connection. Under repeated OOM events (e.g., memory pressure spike), the pool drains to zero and all subsequent requests block indefinitely.

**Fix:** Set a thread-local `wolf_req_oom = 1` flag and check it after each `wolf_req_alloc`. Return from the handler normally; the worker loop detects the flag and sends 503 after proper cleanup.

**Verdict:** ⚠️ CONDITIONALLY SOUND

---

## FINDING 10 — 🟢 LOW: Closure address validation is x86-64 Linux hardcoded

**Area:** Portability | **Affects:** `runtime/wolf_http_engine.c:756`

```c
if ((uintptr_t)(void*)fn < 0x400000UL || (uintptr_t)(void*)fn > 0x7fffffffffffUL)
```
This hardcodes the Linux x86-64 user-space text segment range. On ARM64 Linux (`0x0000FFFFFFFFFFFF` top), macOS (ASLR at `0x100000000`+), or any 32-bit target, this check misfires.

**Fix:** Embed a compile-time magic cookie (`uint32_t sentinel = 0xW0LFC105`) in `wolf_closure_t` at construction. Verify `closure->sentinel == 0xW0LFC105` at dispatch. Platform-independent, O(1), catches stale pointers that happen to land in the valid range.

**Verdict:** ⚠️ CONDITIONALLY SOUND

---

## Prioritized Remediation Table

| Priority | Finding | Effort | Block |
|:---:|:---|:---:|:---:|
| **P0** | #3 WebSocket use-after-free | Medium | Ship |
| **P0** | #1 Arena inner-slab overflow leak | Small | Ship |
| **P0** | #2 MSSQL SQL injection stub | Trivial | Ship |
| **P1** | #7 String interpolation vs real prepared statements | Large | Production |
| **P1** | #4 JWT timing oracle | Trivial | Production |
| **P1** | #5 Argon2id MODERATE default | Trivial | Production |
| **P1** | #6 Arena pool memory fence | Trivial | Production |
| **P2** | #8 SHA-256 key derivation | Small | Beta |
| **P2** | #9 pthread_exit on OOM | Small | Beta |
| **P2** | #10 Closure check portability | Small | Beta |

---

## AXIOM Verdict

Wolf's crypto choices are **architecturally correct** — `sodium_memcmp`, `randombytes_buf`, Argon2id, XSalsa20-Poly1305 are all right calls. The gaps are **implementation-level**, not design-level.

Three ship-blockers exist. Fix them in order: **#3 → #1 → #2**. The WebSocket arena bug is the most dangerous because it produces silent memory corruption that only manifests under concurrent WS connections — the kind of bug that passes all unit tests and explodes in production.

*Next AXIOM probe domain: Benchmark Methodology — p99 latency, coordinated omission, and how to make Wolf's speed claims defensible.*
