# Wolf — Product Roadmap

## Vision Statement
Wolf is a compiled, natively fast language with PHP-like syntax.
**Target users:** Backend developers who want PHP's ergonomics with C's speed.
**Adoption goal:** Production-grade for millions of developers and AIs by 2028.

## Phase Roadmap

### Phase 0 — Foundation (DONE ✅)
- [x] Lexer, Parser, Resolver, TypeChecker
- [x] LLVM IR emitter + native binary output
- [x] C runtime library (wolf_runtime.c)
- [x] HTTP server (multi-threaded, worker pool)
- [x] MySQL, Redis, JWT stdlib
- [x] JSON encode/decode (full RFC 7159 including \uXXXX)
- [x] wolf.config (INI format, php.ini equivalent)
- [x] MySQL connection pool (mutex+cond_var, health-check)
- [x] String interpolation: {$var}, {$obj->prop}, {$obj->method()}, {func()}

### Phase 1 — Production Runtime (In Progress 🔄)
- [ ] Memory arena allocator (per-request, zero leaks in HTTP server)
- [ ] Real Redis client (hiredis integration, replace mock)
- [ ] Graceful shutdown (SIGTERM → pool destroy → close connections)
- [ ] Request timeout enforcement
- [ ] File upload handling
- [ ] WebSocket support (wolf_ws_*)

### Phase 2 — Language Completeness
- [ ] Interfaces / traits
- [ ] Generics (type parameters)
- [ ] Error handling (try/catch complete, propagation)
- [ ] Closures / first-class functions
- [ ] Package system (import "github.com/..." style)
- [ ] Goroutine-equivalent (wolf_async, wolf_await)
- [ ] Pattern matching (match statement — full exhaustiveness check)

### Phase 3 — Ecosystem
- [ ] Wolf package registry (wolf install)
- [ ] Standard library expansion (HTTP client, crypto, UUID, mail)
- [ ] Wolf LSP (language server protocol)
- [ ] VS Code extension
- [ ] wolf fmt (code formatter)
- [ ] wolf test (test runner with assertions)
- [ ] Benchmarking suite vs Rust/Go/C (public results)

### Phase 4 — Self-Hosting (Year 1)
- [ ] Re-implement lexer in Wolf
- [ ] Re-implement parser in Wolf
- [ ] Re-implement emitter in Wolf
- [ ] Wolf compiler compiles Wolf compiler (bootstrap)
- [ ] Go compiler kept as parity/fallback

### Phase 5 — Go Independence (Year 1.5)
- [ ] Go compiler retired
- [ ] Wolf is the only Wolf compiler
- [ ] All stdlib written in Wolf + C runtime only

## Standard Library — Current Status

| Module | Functions | Status |
|--------|-----------|--------|
| Strings | 30+ (upper, lower, trim, pad, split, join...) | ✅ |
| Arrays | 20+ (push, pop, slice, merge, sort...) | ✅ |
| Maps | get, set, keys, values, flip | ✅ |
| JSON | encode, decode, pretty | ✅ |
| Math | abs, ceil, floor, sqrt, pow, trig... | ✅ |
| Date/Time | now, format, diff, strtotime | ✅ |
| Security | md5, sha256, bcrypt, uuid_v4, jwt | ✅ |
| Database | connect, prepare, bind, execute, fetch | ✅ |
| Redis | connect, get, set, del, hget, hset | ⚠️ Mock |
| HTTP Server | serve, req, res | ✅ |
| HTTP Client | get, post, put, delete | ❌ Not started |
| File System | read, write, append, delete, stat | ✅ |
| Validation | email, url, phone, uuid, json, ip | ✅ |
| Encoding | base64, url encode/decode | ✅ |
| ML Bridge | @ml blocks → CPython | ✅ |
