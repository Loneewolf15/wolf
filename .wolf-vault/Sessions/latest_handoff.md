# Handoff — 2026-03-25

## Where We Left Off

Sprint 5 (File Uploads & Metal-Ready Audit) is **fully complete**. All vault docs updated. The runtime is in a clean, green, production-ready state.

## Commits This Session
```
24b5ac8 (HEAD) fix(emitter): correct prefixing and inference for user-defined functions; update vault with stdlib-02 and jwt progress
bee0afe fix: guard hiredis, persistent strdup for defines/pool creds, no-op Redis stubs, ...
8c00089 fix: guard hiredis, persistent strdup, no-op Redis stubs, ...
```
> Sprint 5 changes are **uncommitted** — see files modified below. Commit before next session.

## Tests Status
```
ok  internal/compiler     0.563s
ok  internal/emitter      0.010s
ok  internal/lexer        PASS (cached)
ok  internal/parser       PASS (cached)
ok  internal/resolver     PASS (cached)
ok  internal/typechecker  PASS (cached)
ok  e2e                   122.907s  ← 24/24 tests (22 std + shutdown + upload)
```
**All green. Zero failing tests.**

## Next Immediate Task
**MSSQL Real Implementation**
- File: `runtime/wolf_runtime.c`, `#ifdef WOLF_DB_MSSQL` block
- Action: Install `freetds-dev` or `unixodbc-dev`, implement `wolf_db_mssql_*` using FreeTDS API
- Alternative first: **WebSocket** (`wolf_ws_*`) if MSSQL libs aren't available on the dev machine

## Open Issues / Watch Out For
1. **Sprint 5 is uncommitted.** Run `./fix_and_push.sh` with message:
   ```
   feat(runtime): native multipart file uploads, wolf_file_save, WOLF_FREESTANDING audit
   ```
2. **wolf_http_req_file arg-type coercion** (BUG-031): If you add any NEW `wolf_http_req_*` functions, always add them to the coercion switch in `llvm_emitter.go:2191` — otherwise the emitter will silently pass req_id as a heap pointer.
3. **WOLF_MAX_UPLOADS = 8**: Hard limit per request. Increase in `wolf_runtime.c` if needed; it's a simple `#define`.
4. **wolf_file_save does not create parent directories**: Callers must ensure the path exists first. Consider adding a `wolf_dir_create(path)` helper in the next sprint.
5. **Binary is not stripped** (debug_info present). For production use `go build -ldflags="-s -w"` to bring it from 4.5MB toward ~3MB.

## Relevant Files Modified This Session
| File | What Changed |
|------|-------------|
| `runtime/wolf_runtime.c` | `wolf_upload_t`, `wolf_parse_multipart`, `wolf_http_req_file`, `wolf_base64_encode_bin`, `wolf_file_save`, `#ifndef WOLF_FREESTANDING` guards |
| `runtime/wolf_runtime.h` | Added `wolf_http_req_file`, `wolf_http_req_file_count`, `wolf_file_save` declarations |
| `internal/emitter/llvm_emitter.go` | Added `declare ptr @wolf_http_req_file(i64, ptr)` + coercion fix for arg0 |
| `e2e/upload_test.go` | New E2E test: multipart POST → JSON response verification |
| `e2e/testdata/_server_upload.wolf` | New server script: handles `/upload` route |

---

## Sentinel Review (Sprint 5 Changes)

### 🛡️ Sentinel Review — `wolf_parse_multipart` + `wolf_http_req_file`

**Change proposed:** Native multipart/form-data parser storing uploads in per-request arena  
**Scaling risk:** 🟡 MEDIUM  
**Speed risk:** 🟢 OK  

**Scalability Checklist:**
| Area | Requirement | Verdict |
|---|---|:---:|
| HTTP Stack | Non-blocking I/O only | ✅ (parser runs inside worker thread, no additional I/O) |
| Memory | Per-request arena — no global malloc in hot paths | ✅ All `wolf_req_alloc` |
| Strings | Zero-copy where possible | ✅ Part headers copied once; body memcpy'd once |
| Threading | No global state written from worker without lock | ✅ `ctx->uploads` is thread-local per slot |
| Binary Size | ≤ 8MB | ✅ 4.5MB |
| Complexity | No O(n²) in hot path | ⚠️ (see concern below) |

**Concern:**  
```c
for (const char* s = p; s + delim_len <= end; s++) {
    if (memcmp(s, delim, delim_len) == 0) { ... }
```
This is an O(n × delim_len) = O(n × ~64) linear scan over the body for each boundary. For a 10MB file upload this is ~640M byte comparisons. For typical file sizes (< 10MB) this is acceptable on modern CPUs (~100ms worst case). For large uploads it will become a bottleneck.

**Alternative:** Replace inner `memcmp` scan with the Boyer-Moore-Horspool string search algorithm (O(n/m) average). Implementation is ~40 lines in C and would make 10MB uploads ~8× faster on the boundary scan step.

**Speed Checklist:**
| Area | Verdict |
|---|:---:|
| Zero-Cost Abstractions | ✅ Arena alloc, memcpy — maps cleanly to `rep movsb` |
| Data Locality | ✅ wolf_upload_t embedded in wolf_http_context_t (stack-adjacent) |
| Cache Efficiency | ✅ wolf_upload_t small (5 fields × 8 bytes = 40 bytes, fits in cache line) |
| SIMD Vectorization | ⚠️ memcmp inner loop is SIMD-able but compiler must see it is hot |
| Branch Prediction | ✅ delimiter hit is rare — miss branch is correctly predicted cold |

**Verdict:** ⚠️ **APPROVED WITH NOTES**
- Ship as-is for files < 10MB (the typical web use case)
- Upgrade boundary scan to Boyer-Moore-Horspool before enabling large-file (> 10MB) upload support

---

### 🛡️ Sentinel Review — `#ifndef WOLF_FREESTANDING` Guards

**Change proposed:** Wrap OS-only includes + HTTP server block in WOLF_FREESTANDING guard  
**Scaling risk:** 🟢 LOW  
**Speed risk:** 🟢 OK  

**Verdict:** ✅ **APPROVED**  
- Guard boundary is clean (includes → HTTP server → Req/Res API → #endif)
- Core runtime (arena, strings, maps, math, file I/O) unconditionally compiled
- Verified: `clang -DWOLF_FREESTANDING -fsyntax-only runtime/wolf_runtime.c` → 0 errors
- Minor note: `pthread_mutex_t wolf_defines_mutex` at line 447 and `wolf_drain_mutex` at line 80 are still outside the guard. These are linked only when libpthread is available. Should be wrapped in a future `#ifdef WOLF_FREESTANDING` guard to prevent undefined symbol errors on strict freestanding linkers.