# Wolf — Bugs Fixed (Cumulative Log)

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

## Status Ledger
- Total bugs fixed: 16
- E2E tests: 21/21 passing
- Open: memory leaks (BUG-017 in progress), Redis mock, surrogate pairs
