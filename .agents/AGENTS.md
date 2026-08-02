# The Gate Discipline

**A plausible fix is never a confirmed fix.**

1. **Integration Trumps Isolation:** Never accept isolated, pure-function unit tests (or stub tests) as proof of correctness for features that touch network boundaries, I/O loops, or parser integration. You must run live probes against the actual running server.
2. **Mandatory Fault Injection:** Never assume infrastructure changes (shutdown sequences, rate limiters, arena memory) are safe just because they look logically correct. You must rerun full concurrency stress tests (e.g., `bombardier` + `SIGTERM` + `WOLF_TSAN=1`) to prove stability under real failure conditions.
3. **Prove the Negative:** A silent log is not proof of safety. Before claiming a bug is fixed, you must prove that your fault injection successfully triggers failures when the fix is absent or incomplete. 
4. **Honest Handoffs:** Never log an item as unqualified "Verified" unless you have raw logs, fault injection, and live probing to back it up. If a test has constraints (e.g., "Tested single-IP contention only"), explicitly document those limitations in the handoff state.
5. **Test Data, Not Just Survival:** When writing tests for parsers, adapters, or integration points, asserting `Status: 200` or process survival is insufficient. Tests must echo, inspect, or assert against the internal parsed state to prove the data was routed, typed, and formatted correctly. A test passing must mean the logic worked, not just that it didn't segfault.

# Architectural North Star: Zero-Dependency Self-Hosting

**Always remember and evaluate work against `/home/askme/Pictures/wolf-lang/long_term_vision.md`.**

1. **The Holy Grail:** Moving away from Clang, LLVM, and external `.c` runtime dependencies toward a zero-dependency standalone compiler is Wolf's ultimate destination.
2. **Minimize C/LLVM Entrenchment:** When architecting features or runtime additions, avoid introducing new external C-library dependencies or complex LLVM flags where pure Wolf syntax, direct syscalls (`io_uring`, `kqueue`), or self-hosted abstractions can be used.
3. **Design for Phase 5 Transition:** Build runtime structures and stdlib APIs so that existing C-wrappers can be cleanly swapped out for self-hosted Wolf implementations in Phase 5 without breaking developer contracts.
4. **Target Standalone Portability:** Remember that the end goal of every compiler and runtime feature is instant cross-compilation and `FROM scratch` container deployments with zero external dependencies.
5. **Architectural Permanence (Do Not Undo Wins):** If a previous phase achieved zero-dependency static compilation (e.g., `FROM scratch`), any new CLI, deployment tool, or feature must preserve that constraint. Do not silently revert to dynamic linking or OS package manager (`apt-get`) builds to circumvent immediate friction.

# I/O Context-Bleeding in Asynchronous C Runtimes

**Output redirection is determined by thread-local context flags.**

1. **Check the Context Matrix First:** When `print()`, `printf()`, or other standard I/O fails to appear in stdout/stderr during background events (like WebSocket pollers or timers), do not assume a timing or flushing race condition. First, trace the active context ID (e.g., `res_id`).
2. **Beware HTTP Inheritance:** Event dispatchers often lazily reuse HTTP context structs or IDs. If a non-HTTP event sets an HTTP `res_id`, the runtime's I/O wrappers will silently buffer the output into an HTTP response body that is never sent.
3. **Explicit Context Termination:** Always explicitly set response context IDs to `-1` (or equivalent "none" state) when dispatching tasks that are not generating an HTTP response, ensuring standard I/O correctly defaults to the console.

# Go Tooling: Native OS Efficiency over Naive Fallbacks

**Zero external dependencies is not an excuse for inefficient pure-Go polling.**

1. **Leverage Standard Syscalls:** When building foundational tooling (like file watchers, network dispatchers, or IPC) that typically requires third-party dependencies (e.g., `fsnotify`), do not settle for naive pure-Go polling loops (`time.Sleep` + `filepath.Walk`). Instead, write direct, OS-specific bindings using Go's standard `syscall` package (e.g., `syscall.InotifyInit` on Linux, `syscall.Kqueue` on macOS) to achieve maximum performance while maintaining zero external dependencies.
2. **Graceful Degradation:** Always implement a highly optimized Linux-native fast path (`_linux.go`), as Linux is the primary target for Wolf's architecture. Fall back to naive polling (`_fallback.go`) only on unoptimized platforms (like Windows) if native bindings are too complex for the immediate task.

# Execution Discipline: Boundaries and Checkpoints

**A sanity floor is a measurement, not a mandate to fix.**

1. **Strict Verification Boundaries:** When a task is scoped to verify an existing floor or foundation, do not actively debug or implement new fixes for edge cases discovered during that verification. Report the floor's status exactly as it is. Mixed-scope sessions lead to cascading regressions.
2. **The Smallest Possible Slice:** For high-risk, novel, or complex infrastructure operations (e.g., CGO toolchains, static linking, or runtime changes), never execute the entire pipeline in one pass. Execute the smallest verifiable slice first (e.g., just the plugin compilation) and stop to confirm it works. This acts as a natural checkpoint and limits the blast radius of incorrect assumptions.
3. **Handoff Boundaries Are Law:** Never ignore handoff documents that say "stop here" or "deferred to next session". Do not execute deferred tasks, even if `/goal` is active. A `/goal` flag does not authorize breaching explicitly defined scope boundaries. If a session needs to cross a documented stop-point deliberately, the user must explicitly name the deferred item they want tackled — a general `/goal` invocation is not sufficient authorization on its own.

# Static Dependency Maintenance

**Vendored static archives are frozen snapshots — they do not receive OS security updates.**

1. **Document every bundle immediately.** When adding or refreshing static `.a` libraries, update `third_party/VERSIONS.md` with the exact Alpine version, package name, package version, and library version. If this file doesn't exist, create it before committing.
2. **Script the refresh.** Any vendor operation done via ad-hoc terminal commands must be captured in a reproducible script (`scripts/vendor-static-libs.sh` or equivalent). Never leave the refresh process derivable only from session logs or conversation history.
3. **Refresh triggers:** quarterly rotation OR immediately on a CVE in any security-critical vendored library. Wolf's current high-priority libs: `libssl.a`/`libcrypto.a` (OpenSSL), `libsodium.a`, `libcurl.a`, `libmariadb.a`.
4. **Verification gate before commit:** after every refresh, run the full gate: `wolf build --static bench/real.wolf` → `ldd` shows "not a dynamic executable" → smuggling probes all 400 → `go test -count=1 ./internal/... -timeout 120s` passes.

# TSAN + Static + musl Incompatibility

**Never attempt `wolf build --static` combined with `WOLF_TSAN=1`.**

`-fsanitize=thread` (TSAN) requires glibc's `pthread` ABI for its interceptor layer (`__pthread_create`, `__pthread_mutex_lock`, etc.). These symbols do not exist in musl. The `zig cc -fsanitize=thread -target x86_64-linux-musl -static` linker hangs indefinitely with no error output — do not wait for it, kill it immediately.

**Correct verification split for static builds:**
- Race safety → run TSAN against the **dynamic glibc build** (same C code paths, glibc-compatible). Previous TSAN gauntlets (Sessions 37-38) remain valid proof.
- Static binary correctness → run smuggling probes + SIGTERM shutdown + concurrent load test (not TSAN).


# Scope Adherence and Architectural Restraint

**Never implement unrequested architectural features under the guise of a different task.**

1. **Scoping Means Scoping:** If a task is to "scope" or investigate a feature, do not execute large-scale implementations or hack apart core runtime files.
2. **No Unapproved Core Modes:** Do not introduce massive architectural paradigms (such as bare-metal stripped modes or new OS target abstraction layers) unless the user explicitly requested them. If you believe such a refactor is necessary, stop and propose it to the user first.
3. **The Final Verification Gate:** Never commit to `main` or claim a feature is complete without running the full test suite (`go test -count=1 ./internal/... ./e2e/...`) to completion and explicitly reviewing the final `PASS` output.
