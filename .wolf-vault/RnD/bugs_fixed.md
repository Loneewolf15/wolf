# Wolf — Bugs Fixed (Cumulative Log)

## Session 2026-03-20 (Session 3)

### BUG-017: No graceful shutdown — SIGTERM killed process mid-request
- `wolf_http_serve()` ran infinite `while(1)` loop with no signal handling
- SIGTERM/SIGINT caused immediate process death, dropping in-flight requests
- DB pool connections leaked on unclean exit
- Fix: `sigaction()` for SIGTERM/SIGINT sets `wolf_shutdown_requested`; accept loop checks flag; drain loop waits for `wolf_active_requests == 0`; calls `wolf_db_pool_destroy()` then `exit(0)`

### BUG-018: wolf_db_pool_destroy() missing — pool leaked on exit
- Threads blocked in `wolf_pool_acquire()` would hang forever on shutdown
- Pool credential strings (`wolf_pool_host` etc) were never freed
- Fix: `wolf_db_pool_destroy()` closes all slots, broadcasts `wolf_pool_cond` to unblock waiting threads, frees credential strings, resets `wolf_pool_inited`

### BUG-019: No request timeout — slow clients could hold workers indefinitely
- `http_worker()` called `read()` with no timeout — a slow/malicious client could stall a worker thread forever
- Fix: `SO_RCVTIMEO` set to `WOLF_REQUEST_TIMEOUT_SEC` (default 30s) on each client socket; timeout returns HTTP 408; configurable via `-DWOLF_REQUEST_TIMEOUT_SEC=N`

### BUG-020: SIGPIPE not ignored — broken client socket killed process
- Writing to a client that disconnected mid-response sent SIGPIPE to the process, terminating it
- Fix: `signal(SIGPIPE, SIG_IGN)` added in `wolf_http_serve()` before the accept loop

### BUG-021: Double-free in HTTP context during cleanup

- `free_http_context()` and `wolf_http_res_write()` manually called `free()` on fields (method, path, body) that were already registered in the per-request arena.
- This caused a `double free detected in tcache 2` crash when `wolf_req_arena_flush()` was called at the end of a request.
- Fix: Removed redundant manual `free()` calls; let the arena handle all request-lifetime memory cleanup.

### BUG-022: JSON string encoding regression — strings rendered as []
- Loose heuristic in `wolf_json_encode_value` fallback loop misidentified strings starting with certain data as empty `wolf_array_t` objects.
- Fix: Implemented stricter alignment (8-byte boundary) and realistic capacity/size bounds for `wolf_array_t` and `wolf_map_t` fallback checks.

### BUG-023: E2E test suite hangs and timeouts
- `TestEndToEnd` attempted to run long-running server Wolf files as standard CLI scripts.
- Startup timeout (5s) in `TestGracefulShutdown` and overall E2E timeout (120s) were too short for the now-larger runtime compilation.
- Fix: Renamed server test files to `_server_*.wolf`; increased timeouts to 30s/600s; added `fflush(stdout)` to runtime readiness message.

### BUG-024: @main redefinition conflict with Wolf func main()
- The LLVM emitter generated `@main` for top-level `InitStmts` but also let Wolf-defined `func main()` become `@main` (unprefixed).
- This caused an "invalid redefinition of function 'main'" error during compilation.
- Fix: Universal `wolf_` prefix applied to all Wolf-defined global functions in `emitFunction` and `emitCallExpr`; `func main()` now correctly becomes `@wolf_main`.

### BUG-025: Fragile regex JSON parsing in URL Shortener
- URL Shortener example used `wolf_preg_replace` with a complex regex for JSON extraction, which failed on Postman payloads due to POSIX whitespace handles (`\s`) and lack of backreference replacement support.
- Fix: Refactored `url_shortener.wolf` to use the robust `json_decode()` built-in.

### BUG-026: Sequential SQL parameter binding corruption
- `wolf_db_bind()` with `?` as the parameter name replaced ALL `?` in the query with the first bound value because `wolf_internal_str_replace` lacked a replacement limit.
- This caused both `short_id` and `original_url` to be stored with the same value.
- Fix: Modified `wolf_internal_str_replace` to support a `limit` parameter; `wolf_db_bind` now uses a limit of 1 to ensure sequential binding.

---

## Session 2026-03-19 (Session 2)

### (No numbered bugs — feature additions)
- Real Redis via hiredis (thread-local `redisContext*`)
- Multi-DB driver support (Postgres/MSSQL) with compiler flag auto-selection
- AgSkill and BackendTemplate removed from repo

---

## Session 2026-03-18 (Commits 73818ba, 208de88)

### BUG-001: gofmt CI failure
- Go code not formatted → pre-commit hook + gofmt -w .

### BUG-002: For loop type inference (10_for, 19_nested_loops)
- collectLocalVars in llvm_emitter.go didn't scan ForStmt.Init
- Loop variable $i got type `ptr` instead of `i64`
- Fix: scan Init in collectLocalVars; infer type regardless of operator

### BUG-003: Foreach duplicate alloca (12_foreach)
- emitRange was emitting `%v = alloca ptr` inside loop body every iteration
- Fix: removed alloca from emitRange — collectLocalVars handles it

### BUG-004: JSON integer encoding (07_stdlib_json)
- Map values stored as strings → age:1 became "age":"1"
- Fix: wolf_map_set_int/float/bool typed variants

### BUG-005: JSON key ordering (07_stdlib_json)
- Keys in insertion order, expected alphabetical
- Fix: bubble sort by key in wolf_json_encode_map

### BUG-006: Typed value system in runtime
- No way to distinguish ints from strings in maps/arrays
- Fix: wolf_value_t tagged union, wolf_val_make/int/float/bool

### BUG-007: Map value retrieval (15_maps)
- wolf_map_get returned raw wolf_value_t* → wolf_print_str printed garbage
- Fix: wolf_map_get unwraps tagged values back to strings

### BUG-008: Array typed values (12_foreach, 17_arrays, 21_json_arrays)
- emitSliceLit used emitArgAsString for all elements
- Fix: wolf_val_int/float/bool for typed elements; wolf_array_get unwraps

### BUG-009: Real json_decode implementation
- json_decode was a stub returning raw JSON → segfault on $data["name"]
- Fix: full recursive JSON parser (objects, arrays, strings, numbers, bools, null)

### BUG-010: Dynamic JSON buffers
- Fixed 4096-byte buffers in encode functions → silent memory corruption
- Fix: wolf_strbuf_t dynamic buffer with automatic doubling

### BUG-011: DB segfault on startup
- db_connect from main thread; MySQL client not thread-safe
- Fix: mysql_library_init in wolf_http_serve, mysql_thread_init/end in worker

### BUG-012: String interpolation for function calls
- {count($x)} not interpolated
- Fix: lexer detects {letter} and routes to scanInterpExpression

### BUG-013: JSON array support (21_json_arrays)
- Arrays not properly encoded/decoded
- Fix: wolf_json_decode handles [...] arrays; dynamic buffer encode

### BUG-014: sendResponse data field returning {} (CRITICAL)
- json_decode stored strings as raw char* → json_encode misidentified type
- Fix: wrap strings in wolf_val_make(WOLF_TYPE_STRING) in json_decode

### BUG-015: json_decode missing \uXXXX unicode
- Only \n \t \r handled → international characters broken
- Fix: full RFC 7159 escape support; dynamic buffer (was fixed char[4096])

### BUG-016: {$this->method()} not interpolated
- scanInterpContent stopped at ( — method calls silently dropped
- Fix: paren-depth tracking in scanInterpContent after ->method

---

## Status Ledger

- Total bugs fixed: 26 (BUG-001 through BUG-026)
- E2E tests: 22/22 passing (including shutdown and URL shortener integration)
- Open: MSSQL mock (unused variable warnings), surrogate pair unicode (\uD83D\uDE00)