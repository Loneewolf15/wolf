## Sprint Brief — 2026-06-21

**Candidate 1: Database ORM UX Foundation (Eager Loading, Pagination, N+1 Detection)**
**Priority Score:** 8/10
**Conflicts:** None
**Verdict:** ✅ GO

**Rationale:** With the Package Manager MVP (`wolf install`) shipped and `Closures`/`Pattern Matching` natively implemented, the language completeness foundation is largely solid. The next P1 roadmap tier (DB-02, DB-03, DB-04) tackles the universal ORM debate highlighted in the `manifesto.md`. Implementing eager loading (`$this->db->with()`) and pagination directly into the query builder brings Wolf's database DX to the next level without sacrificing raw SQL honesty. This fulfills a major core promise of the vision.

---

**Candidate 2: Standard Library Expansions (Type Casting & Money Module)**
**Priority Score:** 6/10
**Conflicts:** None
**Verdict:** ⚠️ DEFER

**Rationale:** `STDLIB-09` (Type casting) and `STDLIB-10` (Money module) are P1 priorities but they represent incremental DX improvements rather than architectural keystones. They should be queued behind the database query builder expansions (which have a higher risk of architectural regression and require Sentinel compliance).

---

**Candidate 3: Built-in Pub/Sub & WebSockets Native Bindings**
**Priority Score:** 5/10
**Conflicts:** Requires DB stability first
**Verdict:** ⚠️ DEFER (for this sprint)

**Rationale:** The `manifesto.md` outlines Phoenix-style channels and WebSockets as a core language feature, but the `roadmap.md` categorizes Brokerless native pub/sub as P3. You explicitly asked "what's our next sprint?" after my WebSocket suggestion. The Compass rules dictate that we do not skip the dependency graph. We must greenlight the P1 Database and STDLIB layer before building the high-level Pub/Sub event streaming layer.
