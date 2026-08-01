# Handoff — 2026-08-01 (Session 39)

## Where We Left Off

Session 39 was a recovery session following an unverified attempt to implement `WOLF_FREESTANDING` / "Trojan Horse" proxy logic. That logic was reverted and moved to `feature/freestanding`. The `main` branch has now been stabilized, test suite integrity restored, and previous broken state cleaned up.

## Commits This Session
```
d82d63e (HEAD -> main) fix(compiler): restore Static target config without runtime hacks
727d26c (origin/main) fix(runtime): respect PORT env var for cloud deployment and start websocket poller thread
dae451d fix(http): drop connection on 400 Bad Request to defend against HTTP smuggling pipelines; switch to FROM scratch deployment
859a530 chore(agents): add static dep maintenance and TSAN+musl rules to AGENTS.md
222553c chore(deps): add VERSIONS.md manifest and vendor-static-libs.sh refresh script
40cbf50 Merge branch 'wnee-self-hosting': Scope 1 static server binary compilation
c7b0ef9 (wnee-self-hosting) feat(static): bundle musl static libs and headers for zero-dependency server builds
2018ae8 test(watcher): add fault-injection suite and fix IN_NONBLOCK hang
b8ce8a2 fix(tooling): build zero-dependency inotify watcher and patch process leaks
8156ce5 fix(runtime): add missing timewheel and ratelimit header/source files
03b02a7 fix(compiler): implement Horizon 2 WIR optimization passes and fix native build discovery
f95b564 fix(compiler): resolve missing Lexer symbols by fixing internal autodiscovery path
8fcd023 docs: resolve paper-trail contradiction regarding BUG-052 status
ff5e3d0 chore(config): Add workspace AGENTS.md rule for Gate Discipline
```

## What Was Accomplished

### Reversion & Branch Isolation
Reverted `main` to `727d26c` to eliminate unverified `WOLF_FREESTANDING` hacks that were directly modifying `wolf_runtime.c` outside of scope. Offending work was moved to `feature/freestanding`.

### `Target.Static` Build Error (FIXED)
The `wnee-self-hosting` merge (`40cbf50`) left dangling references to `c.Config.Target.Static` in `compiler.go`. This was properly resolved in `d82d63e` by restoring the `Static` target config option without the stripped-runtime hacks.

### `Lexer.wolf:320 EOF` Parse Error (FIXED)
`TestAutoDiscoveryParity` and `TestAutoDiscoverASTCount` were failing due to scanning untracked debris in `test_dir/Lexer.wolf` which contained invalid syntax (`public constructor`). This directory, along with `e2e/testdata/wolf_out_53_telemetry.wolf/` and other untracked debris, was deleted.

## Tests Status
- `go test ./internal/... ./e2e/...` — **PASS** ✅
- Build integrity restored (`go build ./internal/compiler` succeeds).
- Security Gauntlet: `bench/test_smuggling.c` remains at 21/21 PASS.

## Next Immediate Tasks

1. **Scope `wolf deploy`**: Next session must properly scope CGI/Apache integration as a new I/O adapter around the existing HTTP engine, explicitly excluding runtime-stripping modes or "Trojan Horse" proxies.
2. **Phase 5B — HTTP/1.1 Pipelining Defenses**
3. **wolf install package registry**
4. **Wolf LSP + VS Code foundation**

## Open Items
- Proceed with a fresh session using `/resume`. The baseline is now stable and clean.
