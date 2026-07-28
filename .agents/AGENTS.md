# The Gate Discipline

**A plausible fix is never a confirmed fix.**

1. **Integration Trumps Isolation:** Never accept isolated, pure-function unit tests (or stub tests) as proof of correctness for features that touch network boundaries, I/O loops, or parser integration. You must run live probes against the actual running server.
2. **Mandatory Fault Injection:** Never assume infrastructure changes (shutdown sequences, rate limiters, arena memory) are safe just because they look logically correct. You must rerun full concurrency stress tests (e.g., `bombardier` + `SIGTERM` + `WOLF_TSAN=1`) to prove stability under real failure conditions.
3. **Prove the Negative:** A silent log is not proof of safety. Before claiming a bug is fixed, you must prove that your fault injection successfully triggers failures when the fix is absent or incomplete. 
4. **Honest Handoffs:** Never log an item as unqualified "Verified" unless you have raw logs, fault injection, and live probing to back it up. If a test has constraints (e.g., "Tested single-IP contention only"), explicitly document those limitations in the handoff state.

# Architectural North Star: Zero-Dependency Self-Hosting

**Always remember and evaluate work against `/home/askme/Pictures/wolf-lang/long_term_vision.md`.**

1. **The Holy Grail:** Moving away from Clang, LLVM, and external `.c` runtime dependencies toward a zero-dependency standalone compiler is Wolf's ultimate destination.
2. **Minimize C/LLVM Entrenchment:** When architecting features or runtime additions, avoid introducing new external C-library dependencies or complex LLVM flags where pure Wolf syntax, direct syscalls (`io_uring`, `kqueue`), or self-hosted abstractions can be used.
3. **Design for Phase 5 Transition:** Build runtime structures and stdlib APIs so that existing C-wrappers can be cleanly swapped out for self-hosted Wolf implementations in Phase 5 without breaking developer contracts.
4. **Target Standalone Portability:** Remember that the end goal of every compiler and runtime feature is instant cross-compilation and `FROM scratch` container deployments with zero external dependencies.

# I/O Context-Bleeding in Asynchronous C Runtimes

**Output redirection is determined by thread-local context flags.**

1. **Check the Context Matrix First:** When `print()`, `printf()`, or other standard I/O fails to appear in stdout/stderr during background events (like WebSocket pollers or timers), do not assume a timing or flushing race condition. First, trace the active context ID (e.g., `res_id`).
2. **Beware HTTP Inheritance:** Event dispatchers often lazily reuse HTTP context structs or IDs. If a non-HTTP event sets an HTTP `res_id`, the runtime's I/O wrappers will silently buffer the output into an HTTP response body that is never sent.
3. **Explicit Context Termination:** Always explicitly set response context IDs to `-1` (or equivalent "none" state) when dispatching tasks that are not generating an HTTP response, ensuring standard I/O correctly defaults to the console.

# Go Tooling: Native OS Efficiency over Naive Fallbacks

**Zero external dependencies is not an excuse for inefficient pure-Go polling.**

1. **Leverage Standard Syscalls:** When building foundational tooling (like file watchers, network dispatchers, or IPC) that typically requires third-party dependencies (e.g., `fsnotify`), do not settle for naive pure-Go polling loops (`time.Sleep` + `filepath.Walk`). Instead, write direct, OS-specific bindings using Go's standard `syscall` package (e.g., `syscall.InotifyInit` on Linux, `syscall.Kqueue` on macOS) to achieve maximum performance while maintaining zero external dependencies.
2. **Graceful Degradation:** Always implement a highly optimized Linux-native fast path (`_linux.go`), as Linux is the primary target for Wolf's architecture. Fall back to naive polling (`_fallback.go`) only on unoptimized platforms (like Windows) if native bindings are too complex for the immediate task.
