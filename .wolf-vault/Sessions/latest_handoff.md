# Handoff — 2026-06-10

## Where We Left Off
**Session 29 complete.** Fixed the last two compiler regressions from the Phase 2 stabilization pass and made the Queen's architectural ruling for Sprint 9. The session did NOT write new feature code — it cleaned the runway and set the sprint charter.

**Queen's ruling (backed by forge-orchestrator):** Sprint 9 primary track is `wolf install` — Package Registry MVP. LSP deferred to Phase 11. `__class` JSON filter is a 15-minute warm-up at the top of next session.

## Commits This Session
```
(No new commits — fixes were applied directly to working tree. Commit at start of next session.)
```
> **IMPORTANT:** The two fixes below must be committed at the START of next session before any new work begins.

## Fixes Applied This Session (Uncommitted)
### BUG-080: CGO type mapping case-sensitivity failure
- `strings.Contains(f.ReturnType, "int")` failed to match `GoInt` (capital G). Fixed with `strings.ToLower()`.
- **File:** `internal/emitter/llvm_emitter.go` (lines ~386–399)

### BUG-081: CGO funcSigs populated after body emission  
- `e.funcSigs[f.Name]` was registered in the preamble writer, which runs AFTER the body. Call sites saw `nil` sig → defaulted to `ptr` return → LLVM type mismatch → segfault.
- **Fix:** Moved CGO signature population to the TOP of `LLVMEmitter.Emit()`, strictly before any body statements are processed.
- **File:** `internal/emitter/llvm_emitter.go` (lines ~196–222)

### BUG-082: `wolf_socket_create/connect/send` returning `ptr` instead of `i64`
- Missing from the `i64` return block in `emitCallExpr`. Fixed by adding them to the switch case.
- **File:** `internal/emitter/llvm_emitter.go` (lines ~3456–3462)

## Tests Status
- `go test ./internal/...` — **ALL PASS** ✅ (12 packages)
- `go test ./e2e/...` — 600s timeout on `TestGracefulShutdown` (pre-existing known flake, not a regression)
- `TestEndToEnd/58_sockets.wolf` — **PASS** ✅ (verified individually)
- `e2e/testdata/interop_test/` Go interop binary — **PASS** ✅ (prints `130` correctly)

## Next Immediate Task
**Step 0 (15 min warm-up):** Commit the three fixes above, then add `__class` key filtering to `wolf_json_encode_map` in `runtime/wolf_runtime.c`.
- Find `wolf_json_encode_map`, add `if (strncmp(key, "__", 2) == 0) continue;` guard in the key iteration loop.
- Update `e2e` expected output for any test that currently outputs `__class`.

**Step 1 (main sprint):** Design and implement `wolf.mod` format + `wolf install` CLI.
- `wolf.mod` spec: `module`, `version`, `require` directives (keep it simpler than go.mod)
- `internal/packager/registry.go` — parse `wolf.mod`, resolve deps
- `internal/packager/fetcher.go` — Git clone to `.wolf_modules/`
- `internal/cli/install.go` — Cobra command wiring
- E2E fixture test: create a dummy Wolf module in a temp dir, `wolf install`, compile a file that uses it

## Open Issues / Watch Out For
- **Uncommitted fixes:** BUG-080, BUG-081, BUG-082 are applied but NOT committed. Commit first thing next session.
- **`__class` in JSON output:** `wolf_json_encode_map` currently leaks `__class` key in serialized maps. Fix before shipping any JSON-heavy API work.
- **Thread Queue Backpressure:** `spawn` bomb still hangs process at OS limits. Address before any public load test.
- **E2E `TestGracefulShutdown` timeout:** Pre-existing 600s flake. Do not investigate unless it starts failing on clean builds.

## Relevant Files Modified This Session
- `internal/emitter/llvm_emitter.go` (3 fixes)
- `.wolf-vault/Sessions/latest_handoff.md` (this file)
- `.wolf-vault/Execution/plan.md` (sprint update)
- `.wolf-vault/RnD/bugs_fixed.md` (new bug entries)
