# 🐺 Wolf — Vision Document

> *"We keep everything simple."* — José Valim, Creator of Elixir
>
> That sentence is Wolf's soul. Every feature either serves that principle — or it does not ship.

---

## The Problem

Starting a backend project today is an act of archaeology.

Before you write a single line of business logic you have chosen a language, installed a framework,
picked an ORM (and debated it for a week), configured a logger, wired up a metrics exporter,
decided on a queue driver, set up a test runner, scaffolded your folder structure, wrestled with
Docker, and stared at a blank APM dashboard.

Every language punts this cost onto the developer.

- **Go** gives you a fast binary and then leaves you to assemble the rest from scratch.
- **Node.js** gives you npm and 200 choices for every problem.
- **Python** gives you three async models and a virtual environment to manage.
- **PHP** gives you familiar syntax wrapped around performance that was never designed for the
  modern backend.

None of them ship security, observability, real-time, database safety, or fault tolerance as
first-class language features. All of them make you reach for an external package for things that
every production backend needs anyway.

Wolf is the answer to that problem.

---

## The Mission

> **Wolf is a compiled programming language for the backend that ships production-grade
> infrastructure as part of the language itself.**

A Wolf developer should never need to install a package for common backend work. Everything they
reach for is already there.

Write PHP-familiar syntax. Compile to native machine code. Ship fast.

---

## The Founding Insight

The most valuable backends in the world are not the fastest ones. They are the ones that:

1. **Do not crash silently** when something goes wrong.
2. **Do not leak data** through unsanitised inputs, headers, or logs.
3. **Do not degrade invisibly** under load — developers see what is happening.
4. **Do not take weeks to set up** before the first endpoint is live.

Every language makes these four things the developer's responsibility. Wolf makes them the
language's responsibility.

---

## Who Wolf Is For

Wolf is for backend engineers who know what they are building and are tired of assembling the
same infrastructure from scratch on every project.

**Primary audience:** PHP and Go developers building production APIs — especially in markets where
developer time is expensive, infrastructure budgets are limited, and a single binary with no
external dependencies is a genuine competitive advantage.

**Secondary audience:** Full-stack developers who want a backend that is fast enough for production,
safe enough to sleep through on-call, and simple enough to onboard a junior in a day.

Wolf is not trying to replace Go for systems programming. It is not competing with Rust for
embedded firmware. It is laser-focused on the backend API layer — the layer where most production
software actually runs, and where the cost of bad defaults is highest.

---

## The Core Guarantees

These are not aspirations. They are constraints. If Wolf ships a version that violates any of
these, that version is not ready.

### 1. Familiar, Not Clever

Wolf syntax is PHP-familiar: `$variables`, `$this->method()`, curly braces, `foreach as`,
`match`. Developers who have written PHP or who have read PHP can read Wolf immediately. There are
no new paradigms. There is no monad tutorial. There is no borrow checker lecture.

The learning cost is the toolchain install.

### 2. One Binary, No Baggage

`wolf build` produces a single native binary. No runtime. No GC. No interpreter. No dependency
manifest to ship alongside it. The binary runs anywhere the target architecture runs. Deploy it
with `scp`. Run it with `./app`. Done.

Wolf binaries are compiled via LLVM IR — the same optimisation engine behind Rust and Clang.
Performance is not a Wolf feature. It is an LLVM guarantee.

### 3. Security Is Default, Not Optional

SQL injection is structurally prevented — the compiler warns on raw string interpolation inside
`query()` calls. Security headers are emitted on every response without configuration. Mass
assignment is impossible because `@validate` returns only declared fields. JWT tokens are
truncated in logs. `@encrypted` fields are never logged.

The developer does not need to know about any of this. It is just what Wolf does.

### 4. Zero Setup for Common Backend Work

Authentication, database queries, Redis caching, file uploads, WebSockets, outbound HTTP,
validation, email, real-time broadcasting, job queues, rate limiting — Wolf ships stdlib
implementations of all of these. No `npm install`. No `pip install`. No `go get`. They are
already there. They already work.

### 5. Failures Are Supervised, Not Ignored

Wolf implements the Elixir "let it crash" philosophy without requiring functional programming.
`@supervise` blocks restart automatically on failure. Supervision strategies — `one_for_one`,
`one_for_all`, exponential backoff — are language constructs, not library patterns. A Wolf
developer writes happy-path code. The runtime handles infrastructure failures.

### 6. Observability Is Free

Every Wolf application exposes `GET /health` automatically. Every `wolf dev` session includes a
real-time dashboard at `localhost:8081` — request rates, latency percentiles, DB query times,
error rates — with zero configuration. OpenTelemetry export is one config line. `wolf profile`
samples any running server without agents, subscriptions, or code changes.

---

## What Wolf Takes From Each Language

| Language | What Wolf Inherits | What Wolf Fixes |
|---|---|---|
| **PHP** | `$variables`, `$this->`, familiar OOP, routing convention | Compiled native, no GC, no interpreter overhead |
| **Go** | Single binary, fast compile, simple concurrency model | Verbose error handling → `try/catch`; no built-in DB pattern → `$this->db->`; no enums |
| **Elixir** | OTP supervision trees, "let it crash", Phoenix Channels | Without requiring functional programming or BEAM knowledge |
| **Rust** | LLVM optimisation pipeline, quality error messages with carets | Without borrow checker complexity or steep learning curve |
| **Python** | ML ecosystem | Via `@ml {}` blocks backed by CPython C API — same ecosystem, no separate service |
| **Node.js** | JSON-first, event-driven async | Without the npm ecosystem fragmentation, without single-threaded CPU starvation |

---

## The Competitive Position

Wolf is not a general-purpose language. It is the best language for one specific job:
**building production HTTP backends that need to stay online.**

The claim:

> A Wolf backend is faster than PHP by 100×, faster than Python by 10-50×, and competitive
> with Go — while shipping in a third of the setup time, with security defaults no other
> language provides out of the box.

Benchmark proof of concept (Session 8, April 2026):

| Language | RPS (100k req / 150 concurrent) | P50 | P95 |
|---|---|---|---|
| **Wolf** | **~2,700** | **49.8ms** | **175ms** |
| Go | ~1,900 | 78ms | 310ms |
| Node.js | ~1,200 | 120ms | 490ms |
| Python | ~400 | 380ms | 1100ms |

These numbers will improve. The architecture is designed for it.

---

## The Roadmap — Five Phases to Self-Hosting

Wolf is built in phases. Each phase gate is a testable, shippable milestone.

### Phase 1 — Production Hardening ✅ (Complete)
*What it proved: Wolf can run in production.*

- LLVM IR compilation pipeline end-to-end
- Native HTTP server with M:N POSIX scheduler and SIGURG preemption
- MySQL / PostgreSQL / Redis drivers with connection pooling
- Graceful shutdown, request timeouts, SIGPIPE guard
- 54 E2E tests across Linux, macOS, and Windows
- Single binary cross-compilation (Linux / macOS / Windows)
- Security: SQL injection prevention, path traversal hardening, arena slab proofs,
  Argon2id at OWASP standard, AES-256-GCM, HKDF key derivation

### Phase 2 — Language Completeness ✅ (Complete)
*What it proves: Wolf is a complete programming language.*

- Interfaces, generics (monomorphised), closures, first-class functions
- try/catch exception handling
- Package system with namespace-aware compilation
- Enums, protected visibility, structured concurrency
- Outbound HTTP client (`libcurl`), URL/network utilities
- `wolf dev` HMR dev server with live reload
- `WOLF_HTTP_CLIENT_ENABLED` compile flag, `io_uring` streaming multipart, `wolf_req_arena.active` guard

### Phase 3 — Ecosystem 🔄 (Current)
*What it proves: Wolf can serve a team, not just a solo developer.*

- `wolf install` and package registry
- Wolf LSP + VS Code extension with diagnostics
- `wolf test` built-in test runner with HTTP client, DB mocking, coverage
- `wolf migrate` database migration management
- N+1 query detection at compile time
- Eager loading (`$this->db->with()`)
- Built-in pub/sub (Redis → Kafka → RabbitMQ via config)
- `wolf docker init` from `wolf.config`
- Observability dashboard, `GET /health`, OTel export
- `wolf explain` AI error diagnostic tool
- Rust-quality error messages (file, line, column, caret, suggestion, docs link)
- Benchmarks vs Rust, Go, C — published and reproducible

### Phase 4 — Platform Expansion
*What it proves: Wolf runs everywhere.*

- Wolf LSP with full semantic analysis
- Desktop build target (native WebView)
- WebAssembly compilation target
- Windows M:N scheduler via Fiber API
- GraphQL schema auto-generation from `@graphql` annotations
- gRPC stubs from `@grpc` service definitions
- `wolf profile` production profiler
- `wolf deploy --hot` zero-downtime hot code reload

### Phase 5 — Self-Hosting
*What it proves: Wolf is a real programming language.*

The Wolf compiler is rewritten in Wolf. The Go bootstrap is retired. Wolf compiles Wolf.

This is the terminal milestone. Everything before it is preparation.

Timeline: 18–24 months from Phase 3 completion.

---

## Design Constraints — What Wolf Will Never Do

These are as important as the roadmap.

**1. Wolf will never require a package for something every backend needs.**
If authentication, validation, file uploads, WebSockets, or database queries require an
`install` command, the stdlib has failed.

**2. Wolf will never produce a cryptic error message.**
Every compiler error includes file, line, column, the exact code, a caret, a suggestion,
and a documentation link. No exceptions.

**3. Wolf will never allow security to be opt-in.**
SQL injection prevention, security headers, mass assignment protection, and secret redaction
are on by default. Turning them off requires explicit override. They are never silently absent.

**4. Wolf will never force a new paradigm on the developer.**
Monads, functors, borrow checkers, and ownership models are powerful — and they are not Wolf.
Wolf solves hard problems with constructs that look familiar to anyone who has written backend
code before.

**5. Wolf will never sacrifice observability.**
A Wolf application in production is never a black box. `GET /health`, the dev dashboard,
and telemetry exports exist in every Wolf app by default. A developer must actively remove
them to lose visibility.

---

## The Long Game

Wolf is building toward a world where a solo developer or a small team can ship a production
backend that handles real load, survives real failures, and is maintained by people who read
it on day one — without a week of framework orientation and a month of infrastructure setup.

The benchmark for success is not academic. It is:

> A junior developer joins a Wolf project on a Monday morning. By Tuesday afternoon they have
> shipped their first endpoint to production, they understand the error messages they see,
> they can read the observability dashboard, and they have not introduced a SQL injection
> vulnerability — because Wolf made that structurally impossible.

That is the goal. Everything in this document serves it.

---

## Current Status

| Area | Status |
|---|---|
| Compiler (Lexer → Parser → WIR → LLVM → Binary) | ✅ Production |
| C Runtime (HTTP engine, DB drivers, stdlib 110+) | ✅ Production |
| Security hardening | ✅ Complete (AXIOM audited) |
| Cross-platform builds (Linux / macOS / Windows) | ✅ Complete |
| Language features (interfaces, generics, closures, enums) | ✅ Complete |
| HMR dev server (`wolf dev`) | ✅ Functional |
| E2E test suite | ✅ 54 tests |
| Package system | ✅ Complete |
| `wolf install` / package registry | ⬜ Planned (Phase 3) |
| Wolf LSP + VS Code | ⬜ Planned (Phase 3-4) |
| Built-in test runner (`wolf test`) | ⬜ Planned (Phase 3) |
| Self-hosting compiler | ⬜ Planned (Phase 5) |

---

*This document is the north star. The execution plan lives in `.wolf-vault/Execution/plan.md`.
The feature backlog lives in `features.md`. The research manifesto lives in
`.wolf-vault/Product/manifesto.md`. When there is a conflict between any of those documents and
this one, this one wins.*

---

**Wolf** — Write PHP. Compile native. Ship fast. 🐺
