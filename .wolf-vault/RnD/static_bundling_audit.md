# Technical Spec: Static Bundling & Production Readiness

## 1. Zero-Dependency Static Bundling
Wolf binaries are now fully portable across Linux environments by statically linking all cryptographic and security libraries.

### Key Libraries (runtime/static/):
- **OpenSSL (libcrypto.a, libssl.a)**: Provides high-performance SHA-256/512 and RSA.
- **Libsodium (libsodium.a)**: Provides Argon2id, XSalsa20-Poly1305, and Curve25519.
- **PCRE2 (libpcre2-8.a)**: Provides high-speed regular expressions.
- **Hiredis (libhiredis.a)**: Provides native Redis connectivity.

### Results:
- **Binary Size**: 4.8MB (URL Shortener POC).
- **Portability**: Verified via `ldd` — no dependencies on `libssl.so`, `libsodium.so`, or `libcrypto.so`.

---

## 2. Production Hardening Audit (March 2026)
We addressed several prototype-level weaknesses to ensure Wolf can handle enterprise-scale traffic.

### Thread Safety:
- **Per-Request Arena Allocation**: Every HTTP worker thread now has a `__thread` local memory arena. This prevents cross-request data pollution and memory corruption.
- **DB Pool Isolation**: Added a `pthread_mutex_t` and `pthread_cond_t` to the MySQL connection pool. It now gracefully handles wait-queues and prevents race conditions during acquisition.

### Graceful Shutdown:
- **SIGTERM/SIGINT**: The runtime captures shutdown signals, stops the HTTP accept loop, and **drains** all in-flight requests before exiting.
- **Clean Teardown**: `wolf_db_pool_destroy()` correctly closes all connections and wakes up blocked threads.

### Reliability:
- **Request Timeouts**: Integrated `SO_RCVTIMEO` into HTTP workers. This prevents "slowloris" style attacks or hanging on dead connections.
- **Broken Pipes**: Added `SIGPIPE` ignore to prevent crashes when a client abruptly closes a socket while the server is writing.

**Status**: Wolf is now recommended for high-performance, single-binary production deployments. 🐺🔥
