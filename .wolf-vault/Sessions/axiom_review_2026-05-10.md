## AXIOM Session — 2026-05-10
**Domain probed:** Security (Domain 7) → Memory Management (Domain 1)
**Question asked:** wolf_db_escape is now the single escaping gateway. But it accepts `void* conn`. Inside the MySQL branch, you cast it to `MYSQL*` and call `mysql_real_escape_string`. Who validates that the `void*` is actually a live, connected `MYSQL*` handle and not a stale pointer from a pool slot that was recycled mid-request?
**Answer given:** wolf_db_escape checks `if (!conn)` and returns an empty string. However, it does NOT verify that the conn handle is still connected — it trusts the pool to deliver live connections.
**AXIOM verdict:** CONDITIONALLY SOUND
**Risk logged:** Yes — `wolf_db_escape(stale_conn, val)` will invoke `mysql_real_escape_string` on a dangling handle. MySQL's client lib will likely segfault or return an unescaped string. The pool's `WOLF_DB_PING` guard mitigates this for normal pool churn, but not for a race between connection recycling and an in-flight escape call.
**Next probe:** Domain 1 — Arena Management. `wolf_req_arena_flush` is called after every HTTP handler. But `wolf_db_escape` allocates its output via `wolf_req_alloc`. If `wolf_db_escape` is called outside a request context (e.g., from a background maintenance function or a WebSocket close handler), its allocation lands in a stale or uninitialized per-thread arena. What is the arena state when `wolf_db_escape` is called from `wolf_ws_handle_read_event`'s `close_ws` label — specifically, is `wolf_req_arena_init()` called before the close handler dispatches?

## 🧠 AXIOM Review — wolf_db_escape / Query Builder SQL Injection Fix

**Area:** Security
**Risk Level:** 🔴 CRITICAL (pre-fix) → 🟡 MEDIUM (post-fix)
**Affects:** runtime — wolf_runtime.c, wolf_qb_insert, wolf_qb_update, wolf_qb_where

**The Problem:**
`wolf_db_escape` performs connection-character-set-aware escaping correctly, but its NULL-conn guard returns an empty string rather than aborting the query. An empty string IS injection-safe but produces silent data corruption — a `WHERE id = ''` clause on an INSERT will pass silently.

**The Evidence:**
```c
if (!conn) {
    fprintf(stderr, "[WOLF-QB] wolf_db_escape called with NULL conn ...");
    return wolf_req_strdup("");  // silent data corruption, not a hard abort
}
```

**Real-world Parallel:**
PostgreSQL's `PQescapeStringConn` sets an error flag on the connection object when called with bad input and the caller checks `PQerrorMessage`. Wolf has no equivalent error propagation from escape to query to caller.

**The Fix:**
Replace the empty-string return with a caller-visible error signal. Add a `wolf_qb_last_error` thread-local string. If `wolf_db_escape` is called with NULL conn, set `wolf_qb_last_error = "DB escape called with NULL connection"` and return `""`. Callers like `wolf_qb_insert` should check this flag before executing the query.

**Follow-up question:**
`wolf_req_alloc` is used inside `wolf_db_escape` to allocate the escaped buffer. If `wolf_db_escape` is called from a code path where `wolf_req_arena_init()` has not been called on this thread (e.g., a background goroutine or WS close handler), where does that allocation land?

**Verdict:** ⚠️ CONDITIONALLY SOUND
