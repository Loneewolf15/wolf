# Wolf Vision & Roadmap

The language ships when the standard library is complete and the runtime handles 10,000 concurrent websockets without dropping a single DB connection.

## Implementation Priority Order (Compass Ranked)

| Priority | Feature | Score | Definition |
|----------|---------|-------|--------|
| **P0** | **Package System** | 9/10 | `import "wolf/std/http"`, `wolf.mod`, multi-file compilation |
| **P0** | **`wolf_dns_lookup` timeout** | 8/10 | 2s deadline preventing worker stalls |
| **P1** | **Eager loading (DB-02)** | 7/10 | `$this->db->with("relation")` support |
| **P1** | **N+1 detection (DB-03)** | 7/10 | Compiler warning on queries in loops |
| **P1** | **Pagination (DB-04)** | 7/10 | `$this->db->paginate($req)` built-in pagination |
| **P1** | **Closures** | 7/10 | First-class functions |
| **P1** | **Type casting (STDLIB-09)** | 6/10 | `settype()`, improved `gettype()` return names |
| **P1** | **Money module (STDLIB-10)** | 6/10 | `money_format`, `money_add`, `money_percentage` etc. |
| **P2** | **Pattern matching** | 5/10 | `match` statement full exhaustiveness check |
| **P2** | **`wolf dev`** | 5/10 | Hot reload dev server |
| **P2** | **`wolf test`** | 5/10 | Built-in test runner with mocking |
| **P2** | **`wolf migrate`** | 5/10 | DB migration management |
| **P2** | **`wolf generate feature X`** | 5/10 | Native app scaffolding generator |
| **P2** | **Higher-order utilities** | 5/10 | `pipeline()`, `retry()`, `memoize()` (STDLIB-10) |
| **P2** | **Phone module** | 4/10 | `phone_format()`, `pluralise()` (STDLIB-10) |
| **P2** | **Concurrency constructs** | 4/10 | `@safe`, `@queue`, `@cache`, `@guard` blocks |
| **P2** | **`wolf cron`** | 4/10 | Native scheduler |
| **P2** | **Crypto OS guards** | 4/10 | `#ifdef WOLF_CRYPTO` for bare-metal crypto swaps |
| **P2** | **Windows M:N API** | 4/10 | `ucontext_t` fallback using Fiber API for Windows |
| **P3** | **Observability** | 4/10 | Telemetry dash, health endpoint, OTel export |
| **P3** | **Security defaults** | 4/10 | Security headers, SQL inject trace, audit trail |
| **P3** | **MSSQL integration** | 4/10 | Replace mock with FreeTDS |
| **P3** | **`wolf docker init`** | 3/10 | Generator from wolf.config |
| **P3** | **GraphQL generation** | 3/10 | `@graphql` annotations |
| **P3** | **gRPC support** | 3/10 | `@grpc` service definitions |
| **P3** | **Built-in pub/sub** | 3/10 | Brokerless native pub-sub |
| **P4** | **IDE Tooling** | 3/10 | Wolf LSP + VS Code extension |
| **P4** | **Desktop target** | 3/10 | Built-in WebView via v0.3.0 |
| **P4** | **WASM target** | 3/10 | Wolf browser support |
| **P5** | **Self-hosting compiler** | —/10 | Bootstrapping Wolf (18-24 months) |

## Standard Library — Core Coverage Required Before v0.1.0

- **Missing String/Array**: `array_fill`, `array_chunk`, `array_column`
- **Missing Math**: `array_mean`, `array_std_dev`
- **Missing Date**: date object methods and timezone constants
- **Missing File System**: `copy()`, `move()`, `scan_dir()`