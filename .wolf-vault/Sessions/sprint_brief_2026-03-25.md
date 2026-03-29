## Sprint Brief — 2026-03-25

**Proposed:** Sprint 6 — Native HTTP Client (`libcurl` integration)
**Priority Score:** 42/50
- **Foundation:** 5/5 (Core infra for API-driven logic)
- **Stability:** 4/5 (Standard libcurl is robust)
- **Scale:** 2/5 (External calls; blocking vs non-blocking trade-offs)
- **DX:** 5/5 (Enables calling external APIs from Wolf)

**Conflicts:** 
- **MSSQL:** 🚫 BLOCKED (Missing `freetds-dev` or `unixodbc-dev` on host).
- **WebSocket:** ⚠️ DEFER (Higher complexity than client; client is shorter path to DX).

**Verdict:** ✅ **GO**

**Rationale:** 
With File Uploads and the Metal-Ready audit complete (Sprint 5), Wolf is a great *server*. However, it currently has zero ability to act as a *client* (it cannot call external APIs). Since `libcurl` is already available on the system, implementing a native bridge is the most impactful infrastructure task unblocked. This completes the "Phase 2: Stdlib Completion" goal from the roadmap.

**Proposed MVP:**
- `wolf_http_get(url, headers)` → JSON `{status, body, headers}`
- `wolf_http_post(url, body, headers)`
- Integrated into `wolf_runtime.c` + new `wolf_http_client.c`.

**Done Definition:**
- E2E test calling a Go mock server.
- 0 memory leaks on curl handle cleanup.
- Supported on all targets (non-freestanding).
