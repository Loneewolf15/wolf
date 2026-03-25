# Wolf Bugs Fixed — Cumulative Log

## Session 2026-03-25 (Session 5 — File Uploads & Metal-Ready Audit)

### BUG-031: LLVM emitter passes req_id as ptr instead of i64 for wolf_http_req_file
- **Class:** P1 🟠 Runtime Stability
- **Root cause:** The emitter's arg-type coercion switch at `llvm_emitter.go:2191` had `wolf_http_req_file` and `wolf_http_req_file_count` missing from the HTTP functions case. All args defaulted to `ptr`. The Wolf handler passed `$req_id: int` which the emitter sent as a raw heap pointer (`140283295567792`) instead of the integer slot index (`0`).
- **Symptom:** `wolf_http_req_file` returned `""` (400 response) on every request. C log showed `req_id=140283295567792` hitting the `req_id >= MAX_CONCURRENT_REQUESTS` guard.
- **Fix:** Added `"wolf_http_req_file"` and `"wolf_http_req_file_count"` to the HTTP case in `emitCallExpr` arg-type switch. arg0 → `i64`.
- **File:** `internal/emitter/llvm_emitter.go`

### BUG-032: wolf_base64_encode truncates binary data at null bytes
- **Class:** P2 🟡 Functional Correctness
- **Root cause:** `wolf_base64_encode(const char* s)` uses `strlen(s)` for length, which stops at the first null byte. Binary file content (images, PDFs) can contain embedded nulls.
- **Symptom:** Uploaded binary files would have truncated base64 output; decoded data would be shorter than expected.
- **Fix:** Added `wolf_base64_encode_bin(const char* data, size_t len)` that uses the explicit `size` from `wolf_upload_t`. `wolf_http_req_file` now calls this instead of `wolf_base64_encode`.
- **File:** `runtime/wolf_runtime.c`

### BUG-033: wolf_parse_multipart used strtok_r destructively on body_start pointer
- **Class:** P2 🟡 Functional Correctness
- **Root cause:** Original `parse_http_request` used `strstr(raw_req, "\r\n\r\n")` and then `strtok_r` on the header section to parse headers. With the body_start now computed by byte-scan (null-terminating at `\r\n\r\n`), the body pointer was computed correctly but only AFTER the null write. No data was actually lost but the original approach was fragile.
- **Fix:** Rewrote body split to use a byte-scan loop that null-terminates `raw_req[i]` and stores the body length, then passes `body_start` + `body_len` to the multipart parser so it never relies on null-termination of binary body data.
- **File:** `runtime/wolf_runtime.c`

---

## Session 2026-03-25 (Session 4 — Technical Debt)

### BUG-029: Graceful shutdown busy-wait
- The graceful shutdown drain logic utilized `nanosleep()` in a 100ms interval loop.
- Fix: Converted the system to use a POSIX condition variable (`pthread_cond_t wolf_drain_cond`) allowing the main thread to sleep idly via `pthread_cond_timedwait` until all HTTP workers successfully exit.

### BUG-030: MSSQL Mock unused variables
- The `WOLF_DB_MSSQL` conditional mock functions did not utilize function parameters, leaving generic warnings.
- Fix: Added `(void)stmt;` standard silencing casts to the MSSQL macros.

---

## Session 2026-03-20 (Session 3)

### BUG-018: wolf_db_pool_destroy() missing — pool leaked on exit
### BUG-019: No request timeout — slow clients could hold workers indefinitely
### BUG-020: SIGPIPE not ignored — broken client socket killed process
### BUG-021: Double-free in HTTP context during cleanup
### BUG-022: JSON string encoding regression — strings rendered as []
### BUG-023: E2E test suite hangs and timeouts
### BUG-024: @main redefinition conflict with Wolf func main()
### BUG-025: Fragile regex JSON parsing in URL Shortener
### BUG-026: Sequential SQL parameter binding corruption
### BUG-027: wolf_sprintf strictly typed args causing segfault
### BUG-028: Unicode Surrogate Pairs incorrectly parsed in JSON Decoder

---

## Session 2026-03-19 (Session 2)
### (No numbered bugs — feature additions: Redis, multi-DB)

---

## Session 2026-03-18 (Session 1)
### BUG-001 through BUG-017 (see original log)

---

## Status Ledger

- Total bugs fixed: **33** (BUG-001 through BUG-033)
- E2E tests: **24/24 passing** (22 standard + graceful shutdown + file upload)
- Open: None
- Next Bloodhound Sweep: Monitor for P2 issues with `Content-Type` header case-sensitivity in `wolf_parse_multipart` (strncasecmp used — OK), and base64 padding edge cases in `wolf_file_save`.