## Sprint Brief — 2026-06-10 (Sprint 9)

**Proposed:** `wolf install` — Package Manager MVP
**Priority Score:** 10/10 (Ecosystem Foundation 5x + DX 3x + Phase 3 Gate 2x)
**Conflicts:** None. Directly unblocks all other Phase 3 tooling.

**Verdict:** ✅ GO — Queen's Ruling

**Rationale:**
The Queen and forge-orchestrator are aligned. The package manager is the single highest-leverage task in the ecosystem layer. Without `wolf.mod` and `wolf install`, Wolf is a single-file scripting language — impressive but isolated. With it, Wolf becomes a shareable, composable ecosystem where every stdlib module, every controller library, every auth package can be distributed and versioned.

The LSP was evaluated and explicitly deferred. An LSP requires: JSON-RPC transport, initialize handshake, textDocument routing, diagnostic publishing, and a VS Code extension. That is a 3-4 week build that produces zero user-facing value until there is a package ecosystem to navigate through it. LSP belongs in Phase 11. Not Sprint 9.

**Sprint 9 Execution Order:**
1. Commit BUG-080/081/082 fixes (first action, unblocks clean state)
2. `__class` JSON filter in `wolf_json_encode_map` (15 min warm-up)
3. Design `wolf.mod` spec — `module`, `version`, `require` directives
4. `internal/packager/registry.go` — `wolf.mod` parser + dep graph resolution
5. `internal/packager/fetcher.go` — Git-based fetch into `.wolf_modules/`
6. `internal/cli/install.go` — `wolf install` Cobra command
7. E2E fixture test — full round-trip: create module → install → compile → run

**ADHD Challenge Accepted:**
`wolf.mod` will NOT be invented from scratch. Design is Go-inspired (`module`, `require`) but intentionally simpler. A Wolf v1 module has: a name, a semantic version, and a list of git-based requirements. No checksums, no proxy, no replace directives for v1. Complexity is earned, not front-loaded.
