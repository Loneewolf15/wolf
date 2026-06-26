# Handoff — 2026-06-26

## Where We Left Off
Sprint 11 is **fully shipped and committed**. All three primary tasks are complete and verified with native E2E tests.

## Commits This Session
```
3e728da feat(sprint-11): lexical closures + stdlib polish (slug, truncate, pipeline)
9daf84f feat(explain): implement wolf explain AI error diagnostic tool
```

## Tests Status
- `go test ./internal/... ./stdlib/...` — **all 18 packages passing, 0 failures**
- E2E `45_closures_mutation.wolf` — **PASS** (closures mutate outer scope correctly)
- E2E `46_stdlib_polish.wolf` — **PASS** (slug, truncate, pipeline all work)

## What Was Done This Session
1. **Lexical Closures (Variable Capture)** — Confirmed `escape_analysis.go` correctly identifies captured variables and lifts them onto the request arena (`wolf_req_alloc`). The full LLVM emission path for closures via `emitFuncLit` was verified and working end-to-end.

2. **`Strings::slug` and `Strings::truncate`** — Native C runtime implementations added to `wolf_runtime.c`. LLVM `declare` headers added to `llvm_emitter.go`. Go fallbacks added to `stdlib/strings.go`.

3. **`pipeline` and `retry` (Higher-Order Stdlib)** — `stdlib/higher_order.wolf` committed and auto-discovered by extending `legacyAutoDiscover` to scan the `stdlib/` directory automatically.

## Next Immediate Task
**Sprint 12 Priority 1:** `wolf dev` hot reload server
- Watch mode: detect `.wolf` file changes → recompile → kill/restart running binary
- File: `cmd/wolf/commands/dev.go` — already exists but has no watch logic
- Use `fsnotify` or poll-based watching (already in go.mod?)

**Sprint 12 Priority 2:** First-class static method references
- Problem: `[Strings::trim, Strings::slug]` emits string literals instead of function pointers
- Fix needed in `ir_emitter.go`: `ClosureExpr`/`StaticCall` used in array context must emit a lambda wrapper automatically
- This would make `pipeline("str", [Strings::trim, Strings::slug])` work without manual lambdas

## Open Issues / Watch Out For
- **`TestFileUpload` E2E still fails** (400 Bad Request) — known issue, not a blocker
- **Static method refs in arrays** aren't first-class yet. The `pipeline` test uses explicit lambda wrappers (`var $trim_fn = func($s) { return Strings::trim($s) }`) as a workaround.
- The autodiscovery addition (`stdlib/` in scan dirs) is path-relative. If a project also has a `stdlib/` folder with user code, it will be included automatically. This is by design but worth noting.

## Relevant Files Modified This Session
- `internal/emitter/escape_analysis.go` — new (tracked)
- `internal/emitter/llvm_emitter.go` — added `declare ptr @wolf_strings_slug`, `declare ptr @wolf_strings_truncate`
- `internal/compiler/autodiscovery.go` — added `stdlib` to `dirsToScan`
- `runtime/wolf_runtime.c` — added `wolf_strings_slug`, `wolf_strings_truncate`
- `stdlib/strings.go` — added `Slug()`, `Truncate()`
- `stdlib/higher_order.wolf` — new (tracked)
- `e2e/45_closures_mutation.wolf` — new E2E test
- `e2e/46_stdlib_polish.wolf` — new E2E test
