# 🧭 Sprint Brief — 2026-06-28

**Session Context:** Session 35 — Post Sprint-12 Triage + Test Architecture
**Requested by:** User (test suite design session)

---

## Pre-Sprint Checklist

- [x] All P0/P1 bugs from `bugs_fixed.md` status `✅`? → **YES** (Open: None)
- [x] Is the next task in the dependency graph actually unblocked? → **YES**
- [x] Are `./internal/...` unit tests green? → **YES** (18 packages, 0 failures)
- [x] Is `wolf test` wired up? → **YES** (Session 30 — `internal/tester/runner.go` complete)

---

## Candidate Tasks (Decision Matrix Applied)

### Candidate A — Concrete Test Suite (Tier 1–3 as proposed)

**Proposed:** Write T1–T3 test files for Wolf using `wolf test` harness
**Priority Score:** 8/10
**Conflicts:** None — `wolf test` is already wired (Session 30). Test runner concatenates `.wolf` test files, compiles, executes.
**Verdict:** ✅ GO

**Rationale:**
The user has written a clean, well-structured three-tier test suite. The infrastructure to run it already exists (`internal/tester/runner.go`). The test runner discovers `_test.wolf` files, scans for `test_*` functions, generates a runner script, compiles it, and executes it. Sprint 10 (Session 30) proved this works with the native Parser test. The proposed T1 tests map exactly to what the runner can handle: they are Wolf functions with assertions. T2 HTTP tests require a running Wolf binary (`wolf dev` or spawned server), which means they need Go-level E2E harness rather than `wolf test` — but that's already the pattern used in `e2e/` since Session 1.

**Critical gap identified by Compass:** T2 and T3 tests are **not `wolf test` territory** — they require:
- A live Wolf HTTP server (needs `wolf build` + `exec`)  
- External tooling (`curl`, `wrk`, `k6`)
- These belong in the **Go E2E harness** (`e2e/`) not in `wolf test`

This is not a blocker. It clarifies the mapping: T1 → `wolf test`, T2 → `go test ./e2e/...`, T3 → shell scripts in `bench/`.

---

### Candidate B — `wolf test` Extension: Assertions + HTTP Mock

**Proposed:** Extend `wolf test` to support `assert_eq`, `assert_type`, `assert_error` built-ins and a built-in HTTP mock for T2-style tests within Wolf
**Priority Score:** 6/10
**Conflicts:** None, but adds scope
**Verdict:** ⚠️ DEFER

**Rationale:**
The user's test suite as written works with the existing `error($msg)` assertion pattern the runner already generates. Adding a built-in HTTP mock server for T2 tests inside `wolf test` is Phase 3 DX work — it scores 2× (DX) on the Compass matrix, not the 5× (Foundation) or 4× (Stability) that governs sprint priority. The existing Go E2E harness already covers T2. Defer to Sprint 14+.

---

### Candidate C — Commit Uncommitted Changes (git status: 11 modified files)

**Proposed:** Commit `cmd/wolf/main.go`, `internal/lsp/handler.go`, `runtime/wolf_http_engine.c`, `runtime/wolf_runtime.c`, `internal/packager/client.go`, and 6 bench files
**Priority Score:** 9/10
**Conflicts:** None
**Verdict:** ✅ GO (must be done FIRST before any test work)

**Rationale:**
`git status` shows 11 modified files and several untracked files (`vscode-wolf/`, `internal/packager/publish.go`, `bench/plot_results.py`, `bench/real.c`, etc.) that are not committed. The Sprint 12 handoff (2026-06-27) says the session committed `a3460e2`, but these changes are post-commit diffs. Tests should not be written against an uncommitted codebase. Stage, review, commit before writing tests.

---

## Sprint Brief — T1 Test Files (Immediate Action)

The correct shape for the test suite the user designed:

### T1 → `tests/` directory, `_test.wolf` suffix, `test_*` functions

```
tests/
  t1_type_inference_test.wolf       → T1-01
  t1_functions_test.wolf            → T1-02
  t1_classes_test.wolf              → T1-03
  t1_null_safety_test.wolf          → T1-04 (compile-error test — needs Go harness)
  t1_scope_isolation_test.wolf      → T1-05
```

**Important:** T1-04 (undefined variable → compile error) **cannot** be tested inside `wolf test` because the runner compiles the file and an expected compile error would abort the entire test run. T1-04 needs a Go-level test that invokes the compiler and asserts `len(errors) > 0`.

### T2 → `e2e/testdata/` Wolf files + Go test driver

These are already the pattern for the 60 existing E2E tests. Map T2-01 through T2-06 to new test files: `61_route_ping.wolf`, `62_route_params.wolf`, etc.

### T3 → `bench/` shell scripts

`run_real_benchmarks.sh` already exists. Add `run_tier3_suite.sh`.

---

## Answer to the User's Direct Question

> "Is `wolf test` wired up yet or are you running Go tests directly?"

**Both.** `wolf test` is wired up via `internal/tester/runner.go` (Session 30). It:
1. Walks the project for `_test.wolf` files
2. Scans for `test_*` functions via the lexer/parser
3. Concatenates all source + generates a runner scaffold
4. Compiles with the Wolf compiler
5. Executes the binary

Go tests (`go test ./e2e/...`) run separately via the existing E2E harness.

**Use `wolf test` for T1-01, T1-02, T1-03, T1-05.**
**Use `go test ./e2e/...` for T1-04, T2-01 through T2-06.**
**Use `bench/` scripts for T3-01 through T3-05.**

---

## SRE Lens — Reliability Notes on the Test Suite

**T2-06 (Concurrent writes, final count = 50)** is the correct canary for the arena allocator. However: Wolf's arena is request-scoped. If `POST /counter` shares a global counter, the race condition is in the DB layer, not the arena. The test is valid but is testing DB atomicity (needs `SELECT ... FOR UPDATE` or `UPDATE counter SET val = val + 1`) — not Wolf's memory model. Scope the test correctly.

**T3-04 (RSS flat after warmup)** — `ps aux | grep wolf` samples RSS. Better: use `pmap -x <pid>` for heap-only tracking, or `/proc/<pid>/status` (`VmRSS`, `VmHeap`). The bench directory already has `results_investigation.txt` with a load test that hit 950 RPS / 47ms avg — well below the T3-03 target of 8,000 RPS. This delta (8× gap) needs investigation before T3 targets are treated as pass/fail criteria.

**The `results_investigation.txt` result (950 RPS, 47ms avg)** is the existing real benchmark. T3-01 target of 40,000 RPS for `/ping` is plausible (no DB), but T3-03 at 8,000 RPS for a DB route requires verification that the MySQL/SQLite pool is configured correctly in the bench environment.

---

## Next Immediate Actions (Priority Order)

1. **Commit uncommitted changes** — 11 modified files, before any test work
2. **Create `tests/` directory** with T1 test files (`wolf test` compatible)
3. **Create `e2e/testdata/61_route_ping.wolf`** through `66_concurrent_writes.wolf` for T2
4. **Add T1-04 compile-error test** to Go unit tests in `internal/compiler/` or `internal/resolver/`
5. **Calibrate T3 RPS targets** against `bench/results_investigation.txt` reality (current: 950 RPS on DB route)
