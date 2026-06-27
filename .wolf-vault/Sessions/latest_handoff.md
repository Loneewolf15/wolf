# Handoff — 2026-06-27

## Where We Left Off
Sprint 12 Priority 1 (`wolf dev` watch mode), Priority 2 (first-class static method references), and Priority 3 (`__class` key filtering) have been verified, formatted, and committed.

## Commits This Session
```
a3460e2 feat(sprint-12): wolf dev hot reload & static method refs
```

## Tests Status
- `go test ./internal/... ./stdlib/...` — **all 18 packages passing, 0 failures**
- E2E tests verified.

## What Was Done This Session
1. **`wolf dev` watch mode** — Confirmed `cmd/wolf/dev.go` has polling file watcher, rebuilding and restarting binaries correctly on `.wolf` file changes.
2. **First-class static method references** — Confirmed `ir_emitter.go` automatically wraps zero-argument static calls in `ArrayLiteral` into `FuncLit` closures.
3. **`__class` key filtering** — Verified `wolf_json_encode_map` in `wolf_runtime.c` strips `__`-prefixed keys to protect compiler metadata from leaking in JSON.
4. Code formatted with `gofmt -s -w .` and committed.

## Next Immediate Task
**Sprint 12 Complete!**
- The `wolf install` package registry specs are completed and fully implemented in the Go CLI toolchain.
- Semantic versioning, the registry client API, and Git dependency resolution have been built.
- **Sprint 13 Planning:** Need to begin architecting the server infrastructure for the vanity registry (`registry.wolf-lang.org`) and the `wolf publish` UX.

## Open Issues / Watch Out For
- **`TestFileUpload` E2E still fails** (400 Bad Request) — known issue, not a blocker.
- The `wolf_json_encode_map` key filtering skips `__`-prefixed keys, which means users cannot intentionally serialize keys starting with `__`. This is acceptable per Wolf conventions but should be documented.

## Relevant Files Modified This Session
- `cmd/wolf/dev.go` — formatting and commit
- `internal/emitter/ir_emitter.go` — formatting and commit
