## Sprint Brief — 2026-06-09

**Proposed:** The "Pure Wolf" Systems Rewrite (Path 1)
**Priority Score:** 10/10 (Foundation 5x + DX 2x + Scale 3x)
**Conflicts:** None. This naturally bridges Phase 3 (Ecosystem) into Phase 4 (Self-Hosting).

**Verdict:** ✅ GO

**Rationale:** 
The Compass agrees with the commander. Relying on massive C libraries like `libcurl` and `libmysqlclient` creates unacceptable installation friction across operating systems (requiring `apt-get` or MSYS2). By dropping these dependencies and rewriting the HTTP Client and MySQL driver in **Pure Wolf** using bare OS socket bindings (`socket()`, `connect()`, `send()`, `recv()`), we guarantee a zero-dependency installation for Wolf developers. This fundamentally fulfills the DX goals of Phase 3, sets the absolute foundation for Phase 4 (Self-Hosting), and can absolutely be executed within the 30-day timeline. 
