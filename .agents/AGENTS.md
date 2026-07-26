# The Gate Discipline

**A plausible fix is never a confirmed fix.**

1. **Integration Trumps Isolation:** Never accept isolated, pure-function unit tests (or stub tests) as proof of correctness for features that touch network boundaries, I/O loops, or parser integration. You must run live probes against the actual running server.
2. **Mandatory Fault Injection:** Never assume infrastructure changes (shutdown sequences, rate limiters, arena memory) are safe just because they look logically correct. You must rerun full concurrency stress tests (e.g., `bombardier` + `SIGTERM` + `WOLF_TSAN=1`) to prove stability under real failure conditions.
3. **Prove the Negative:** A silent log is not proof of safety. Before claiming a bug is fixed, you must prove that your fault injection successfully triggers failures when the fix is absent or incomplete. 
4. **Honest Handoffs:** Never log an item as unqualified "Verified" unless you have raw logs, fault injection, and live probing to back it up. If a test has constraints (e.g., "Tested single-IP contention only"), explicitly document those limitations in the handoff state.
