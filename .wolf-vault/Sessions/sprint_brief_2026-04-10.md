## Sprint Brief — 2026-04-10 (Compass Roadmap Audit)

### State Assessment
- **Open bugs:** None (BUG-041 fixed this session)
- **Current sprint:** Sprint 7 — Database Parity & Benchmarking
- **Phase 2 stdlib:** 3/10 items complete (STDLIB-01, 02, 05, 08 done)
- **Phase 3 DB:** 1/4 items complete (DB-01 done)
- **Phase 4+:** Blocked behind Phase 2/3

---

### Candidate 1: STDLIB-06 — HTTP Client
**Priority Score:** 8/10 *(Foundation 5× — every API-consuming Wolf app needs this)*
**Conflicts:** None — unblocked
**Verdict:** ✅ GO

**Rationale:** The HTTP client (`wolf_http_client.c` via libcurl) is the top missing stdlib item. It unblocks real-world Wolf API integrations and is referenced in plan.md as "Next Unblocked". libcurl is already linked (`#include <curl/curl.h>` + `wolf_ensure_curl`). The wolf_http_client_write_cb scaffolding exists — this is largely a runtime wiring + emitter task.

---

### Candidate 2: STDLIB-03 + STDLIB-04 — Math Stats & Date Object Methods
**Priority Score:** 6/10 *(Stability 4× — partial stdlib causes E2E gaps)*
**Conflicts:** None — unblocked
**Verdict:** ✅ GO

**Rationale:** `array_mean`, `array_std_dev` (STDLIB-03) and date object methods + timezone constants (STDLIB-04) are both partially done. They are low-risk, high-completeness-score items. Finishing them closes known partial-stdlib warnings in the roadmap table.

---

### Candidate 3: STDLIB-07 — File System Completion
**Priority Score:** 5/10 *(DX 2× + Scale 3×)*
**Conflicts:** None — unblocked
**Verdict:** ✅ GO

**Rationale:** `copy`, `move`, `scan_dir`, `path_join` close the file system module. These are pure C runtime additions with no emitter complexity. High value for Wolf web apps that handle file management.

---

### DEFERRED
- **DB-02 (Eager loading):** DEFER until Phase 2 stdlib ≥80% complete.
- **Phase 4 (Enums, Closures, Interfaces):** DEFER until Phase 3 DB layer is green.
- **Real MSSQL:** System-blocked on `freetds-dev` install — resolve separately.

