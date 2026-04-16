# Wolf — Product Roadmap

## Vision Statement
Wolf is a compiled, natively fast language with PHP-like syntax.
**Target users:** Backend developers who want PHP's ergonomics with C's speed.
**Adoption goal:** Production-grade for millions of developers and AIs by 2028.

---

## Phase Roadmap

### Phase 0 — Foundation (DONE ✅)

- [x] Lexer (support #, //, /* ... */ comments)
- [x] Parser, Resolver, TypeChecker
- [x] LLVM IR emitter + native binary output
- [x] C runtime library (wolf_runtime.c)
- [x] HTTP server (multi-threaded, worker pool)
- [x] MySQL, Redis, JWT stdlib
- [x] JSON encode/decode (full RFC 7159 including \uXXXX)
- [x] wolf.config (TOML format, handles target device/memory logic)
- [x] MySQL connection pool (mutex+cond_var, health-check)
- [x] String interpolation: {$var}, {$obj->prop}, {$obj->method()}, {func()}

### Phase 1 — Production Runtime (DONE ✅)
- [x] Memory arena allocator (per-request, zero leaks in HTTP server)
- [x] Real Redis client (hiredis integration, thread-local context)
- [x] Graceful shutdown (SIGTERM/SIGINT → drain → pool destroy → exit)
- [x] Request timeout enforcement (SO_RCVTIMEO, HTTP 408)
- [x] SIGPIPE guard (broken socket no longer kills process)
- [x] wolf_db_pool_destroy() (clean teardown, unblocks waiters)
- [x] Double-free fix in free_http_context() (arena owns HTTP strings)
- [x] Static bundling (libsodium + OpenSSL linked statically, zero-dep binary)
- [x] Native CLI support (argc() / argv() functions)
- [x] Sequential SQL parameter binding (fixed wolf_db_bind limit)
- [x] Compiler Function Prefixing (prevent @main collisions)
- [x] File upload handling (multipart/form-data)
- [🔄] WebSocket support (wolf_ws_*) — Framing complete, Scaling pending
- [ ] MSSQL real implementation (freetds-dev, replace mock)

### Phase 2 — Stdlib Completion (In Progress 🔄)

**Goal: a Wolf developer never installs a package for common backend work.**

#### STDLIB-01 — String Functions
- [x] str_replace, explode, implode, ucfirst, ucwords, lcfirst, etc.
- [x] preg_match, preg_match_all, preg_replace, preg_split (PCRE2 integration)

#### STDLIB-02 — Array Functions

- [x] array_fill, array_combine, array_chunk, array_product
- [x] array_column, compact, extract, array_diff_key

#### STDLIB-03 — Math Functions

- [ ] clamp, rand_float, rand_secure, array_rand
- [ ] array_mean, array_median, array_mode, array_variance, array_std_dev, array_percentile

#### [x] STDLIB-04 — Date & Time Functions
- [x] time_ms, time_ns (done in runtime, wired in emitter)
- [x] date_create, date_format, date_diff (object-style)
- [x] Date object methods: addDays, addMonths, isPast, isFuture, toISO, timezone()
- [ ] Timezone constants: TZ_UTC, TZ_LAGOS, TZ_BERLIN, TZ_LONDON, TZ_NEW_YORK, TZ_DUBAI

#### [x] STDLIB-05 — Security & Crypto
- [x] Real password_hash (Argon2id via libsodium)
- [x] Real sha256/sha512 (OpenSSL EVP)
- [x] hash_hmac, hash_equals (timing-safe via sodium_memcmp)
- [x] Real JWT (HMAC-SHA256, expiration checking)
- [x] encrypt/decrypt (XSalsa20-Poly1305)
- [x] Curve25519 ECDH (keypairs, shared secrets)
- [x] rand_bytes, rand_token (libsodium randombytes)
- [x] uuid_v7 (time-ordered), nanoid, custom_id
- [x] sign/verify (RSA via OpenSSL)
- [x] base64_url_encode/decode, hex_encode/decode

#### [🔄] STDLIB-06 — HTTP Client (Native libcurl)
- [x] http_get, http_post
- [ ] http_put, http_delete, http_patch, http_request
- [ ] Response object: status, body(), json(), header(), ok(), failed()
- [ ] parse_url, parse_query, build_query, build_url, urljoin
- [ ] get_client_ip, is_valid_ip, is_valid_ipv4, is_valid_ipv6
- [ ] dns_lookup, reverse_dns, geoip_lookup

#### STDLIB-07 — File System
- [ ] file_read_lines, file_write_lines
- [ ] file_copy, file_move, file_chmod
- [ ] make_dir (recursive), remove_dir, scan_dir, scan_dir_recursive
- [ ] app_root, temp_dir, temp_file
- [ ] path_join, path_resolve, path_relative
- [ ] file_mime, file_modified_at

#### [x] STDLIB-08 — Validation Rules Engine
- [x] validate($data, $rules) — main validator returning validator object
- [x] $v->passes(), $v->errors(), $v->validated()
- [x] Rules: required, nullable, string, integer, float, bool, array
- [x] Rules: email, url, phone, uuid, date, date_format
- [x] Rules: min, max, between, in, not_in, regex, confirmed
- [x] Rules: unique:table,col, exists:table,col (DB-integrated)
- [x] Rules: file, image, max_size, mime
- [x] Rules: digits, digits_between, alpha, alpha_num, alpha_dash

#### STDLIB-09 — Type Casting
- [ ] settype(&$val, $type) — mutate variable type in place
- [ ] gettype() returning proper type name (currently returns "string"/"null" only)

#### STDLIB-10 — Wolf-Specific Functions
- [ ] env_required($key) — get or throw if missing
- [ ] is_production(), is_development(), is_testing()
- [ ] pipeline($value)->through($fn)->result()
- [ ] retry($n, $fn), retry($n, $fn, $delayMs)
- [ ] memoize($fn)
- [ ] rate_limit($key, $max, $window) — in-memory limiter
- [ ] slug($str) — done, verify wired in emitter
- [ ] truncate($str, $len, $suffix) — done, verify wired
- [ ] pluralise($word, $count) — irregular plurals
- [ ] money_format, money_add, money_subtract, money_multiply, money_divide, money_percentage
- [ ] phone_format($number, $region) — African-first phone normalisation
- [ ] log_debug($msg) — only output with --debug flag
- [ ] inspect($val) — returns formatted string (not void)
- [ ] json_pretty($data) — encode with indentation

### Phase 3 — Database Layer
- [x] DB-01: Query builder — $this->db->builder("table")->where()->orderBy()->get()
- [ ] DB-02: Eager loading — $this->db->with("relation")->query()
- [ ] DB-03: N+1 detection — compiler warning on queries inside loops
- [ ] DB-04: $this->db->paginate($req) — built-in pagination

### Phase 4 — Language Completeness
- [ ] Interfaces / traits
- [ ] Generics (type parameters)
- [ ] Error handling (try/catch complete, propagation)
- [ ] Closures / first-class functions
- [ ] Package system (wolf.mod + wolfpkg)
- [ ] Goroutines + Channels (wolf_async, wolf_await, channel(type))
- [ ] Pattern matching (match — full exhaustiveness check)
- [ ] Enums (native enum types)
- [ ] @supervise blocks (fault tolerance, let it crash)
- [ ] @safe blocks (error-safe execution)
- [ ] @queue and @cache blocks (native background jobs and caching)
- [ ] @guard blocks (authentication as language construct)
- [ ] @contract blocks (API contract testing in CI)
- [ ] Built-in pub/sub (publish/subscribe without external broker)

### Phase 5 — Real-time & Protocol
- [ ] WebSocket support — $this->res->websocket(), broadcast(), presence_track()
- [ ] GraphQL auto-generation — @graphql annotations on models
- [ ] gRPC support — @grpc service definitions

### Phase 6 — Tooling
- [ ] wolf dev — start entire stack in one command, zero config
- [x] wolf new (Smart Mode) — supports script and API templates with Docker
- [ ] TOOL-01: Auto-discovery via `wolf.config` — array of directories (controllers, models, services) that the compiler automatically crawls and includes, eliminating manual file imports.
- [ ] TOOL-02: Dynamic Test Generation — `wolf test --generate` automatically writes test implementations based on `@contract` API definitions.
- [ ] wolf generate feature X — add features to existing project
- [ ] wolf test — built-in test runner with mocking
- [ ] wolf migrate — database migration management
- [ ] wolf docker init — generate docker-compose.yml from wolf.config
- [ ] wolf explain — AI-powered error explanation
- [ ] wolf profile — production profiler without APM
- [ ] wolf deploy --hot — zero-downtime hot code reload
- [ ] wolf fmt — code formatter

### Phase 7 — Observability & Security Defaults
- [ ] OBS-01: metrics_increment/gauge/histogram built-in functions
- [ ] OBS-02: Automatic GET /health endpoint on every Wolf app
- [ ] OBS-03: wolf dev dashboard at localhost:8081
- [ ] OBS-04: OpenTelemetry export
- [ ] SEC-01: Automatic security headers on every response
- [ ] SEC-02: SQL injection compiler warning (string interpolation in query())
- [ ] SEC-03: Secret redaction in logs (@encrypted values, JWT truncation)
- [ ] SEC-04: $this->req->audit() — object request audit trail
- [ ] SEC-05: $this->req->idempotent() — idempotency tracking
- [ ] SEC-06: $this->req->diff() — object diff request feature

### Phase 8 — Package Manager
- [ ] PKG-01: wolf.mod module file (identifies module, no downloading)
- [ ] PKG-02: wolfpkg CLI — install, publish, search (3–6 months post launch)
- [ ] PKG-03: wolfpkg.dev registry — public, permanent mirroring (6–12 months post launch)

### Phase 9 — Self-Hosting (Year 1)
- [ ] Re-implement lexer in Wolf
- [ ] Re-implement parser in Wolf
- [ ] Re-implement emitter in Wolf
- [ ] Wolf compiler compiles Wolf compiler (bootstrap)
- [ ] Go compiler kept as parity/fallback

### Phase 10 — Go Independence (Year 1.5)
- [ ] Go compiler retired
- [ ] Wolf is the only Wolf compiler
- [ ] All stdlib written in Wolf + C runtime only

### Phase 11 — Ecosystem
- [ ] Wolf LSP (language server protocol)
- [ ] VS Code extension (syntax highlighting + LSP)
- [ ] Benchmarking suite vs Rust/Go/C (public results)
- [ ] wolfpkg.dev fully live
- [ ] WASM target (Wolf in the browser)

### Phase 12 — Embedded & Distribution
- [ ] DIST-01: Package Manager Distribution (brew, winget, install.sh) — Auto-fetches Clang dependency and handles PATH setup.
- [ ] EMB-01: Zero-Flag Builds via `wolf.config` — Declare `device = "esp32"` once during `wolf new`, compiler calculates memory budgets.
- [ ] EMB-02: Zero-Runtime Compilation — For ultra-constrained targets (Arduino Uno), emit pure C with inlined stdlib macros and zero C runtime linker overhead.

---

## Standard Library — Current Status

| Module | Functions | Status |
|--------|-----------|--------|
| Strings | 30+ (upper, lower, trim, pad, split, join, replace, regex...) | ✅ Complete |
| Arrays | 25+ (push, pop, slice, merge, sort, diff, intersect...) | ✅ Partial — missing array_fill, array_chunk, array_column |
| Maps | get, set, keys, values, flip | ✅ |
| JSON | encode, decode, pretty | ✅ |
| Math | abs, ceil, floor, sqrt, pow, trig, clamp... | ✅ Partial — missing array_mean, array_std_dev |
| Date/Time | now, format, diff, strtotime, time_ms, time_ns | ✅ Partial — missing date object methods |
| Security | Argon2id, SHA-256/512, JWT, XSalsa20, RSA, Curve25519, UUID v7, Nanoid | ✅ Complete (libsodium + OpenSSL) |
| Database | connect, prepare, bind, execute, fetch, pool, destroy | ✅ |
| Redis | connect, get, set, del, hget, hset (real hiredis) | ✅ |
| HTTP Server | serve, req, res, graceful shutdown | ✅ |
| HTTP Client | get, post, put, delete | ❌ Not started |
| File System | read, write, append, delete, stat, slug, truncate | ✅ Partial — missing copy, move, scan_dir |
| Validation | email, url, phone, uuid, json, ip, alpha | ✅ Partial — no rules engine yet |
| Encoding | base64, url encode/decode | ✅ |
| ML Bridge | @ml blocks → CPython | ✅ |
| Money | — | ❌ Not started |
| Wolf-specific | slug, truncate, uuid_v4, rand_hex | ✅ Partial |

---

## Implementation Priority Order (Current)

```
NOW — Phase 2: Stdlib Completion
- [x] STDLIB-01: String Functions & Regex
- [x] STDLIB-02  Array functions (fill, chunk, column, product)
  STDLIB-03  Math functions (clamp done, add statistics)
  STDLIB-04  Date/time (object methods, timezone constants)
  - [x] STDLIB-05: Security (Argon2id, OpenSSL, JWT, XSalsa20, Curve25519, RSA)
  STDLIB-06  HTTP client (new wolf_http_client.c)
  STDLIB-07  File system (copy, move, scan_dir, path_join)
- [x] STDLIB-08  Validation rules engine
  STDLIB-09  Type casting (settype, improved gettype)
  STDLIB-10  Wolf-specific (money, pipeline, retry, memoize, pluralise)

THEN — Phase 3: Database Layer
- [x] DB-01  Query builder
  DB-02  Eager loading
  DB-03  N+1 detection
  DB-04  Pagination

THEN — Phase 4: Language Completeness
  Enums → @supervise → @safe → @contract → pub/sub → WebSocket

THEN — Phase 6: Tooling
  wolf dev → [x] wolf new → wolf generate → wolf test → wolf migrate
```