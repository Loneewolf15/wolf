## Sprint Brief — 2026-03-26

**Proposed Task 1: Stdlib HTTP Client (STDLIB-06)**
**Priority Score:** 9/10
**Conflicts:** None
**Verdict:** ✅ GO

**Rationale:** This is a critical infrastructure component (Foundation 5x). Most real-world Wolf applications will require the ability to make outbound HTTP requests. It is the highest-priority remaining unblocked item in the Phase 2 Stdlib roadmap.

---

**Proposed Task 2: Production Load Test**
**Priority Score:** 8/10
**Conflicts:** None
**Verdict:** ✅ GO

**Rationale:** Directly aligns with the current Sprint 7 goal. It validates the "Metal-Ready" and "Graceful Shutdown" infrastructure implemented in recent sessions. Crucial for establishing a performance baseline (Scale 3x) before further feature expansion.

---

**Proposed Task 3: WebSocket Support**
**Priority Score:** 7/10
**Conflicts:** None
**Verdict:** ⚠️ DEFER

**Rationale:** While unblocked and high impact (Scale 4x), it introduces significant runtime complexity and stability risk (Stability 4x). Per the "Stability" directive, it should follow the completion of the more foundational HTTP Client to ensure the core network stack is battle-tested first.

---

**Proposed Task 4: Real MSSQL Implementation**
**Priority Score:** 4/10
**Conflicts:** Blocked by `freetds-dev` system dependency.
**Verdict:** 🚫 BLOCK

**Rationale:** Violates Resolution Rule 1: "Never unlock a Feature layer before the Infrastructure layer below it is green." The system lacks the necessary C headers/libraries for real implementation.
