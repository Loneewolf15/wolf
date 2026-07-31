# third_party Static Library Bundle — Version Manifest

> **IMPORTANT:** These are vendored static archives (.a files) for musl/Alpine Linux x86_64.
> They are **frozen snapshots**, not linked to any OS package manager.
> Security patches in these libraries require manual refresh of this bundle.
> See the refresh procedure below.

## Source

| Field | Value |
|---|---|
| Alpine Linux Version | 3.20 |
| Repository | `https://dl-cdn.alpinelinux.org/alpine/v3.20/main/x86_64/` |
| Target Architecture | `x86_64-linux-musl` |
| Bundled Date | 2026-07-31 |
| Bundled By | Session 39 — Scope 1 Static Binary Compilation |

## Bundled Libraries

| Library | Alpine Package | Package Version | Library Version | Security Critical |
|---|---|---|---|---|
| OpenSSL (crypto) | `openssl-libs-static` | `3.3.7-r0` | `3.3.7 (7 Apr 2026)` | ⚠️ YES — CVE watch required |
| OpenSSL (ssl) | `openssl-libs-static` | `3.3.7-r0` | `3.3.7` | ⚠️ YES — CVE watch required |
| libsodium | `libsodium-static` | (from dev pkg) | `1.0.19` | ⚠️ YES — crypto primitives |
| libmariadb | `mariadb-static` | `10.11.18-r0` | `10.8.8-MariaDB` | ⚠️ YES — SQL auth/TLS |
| libcurl | `curl-static` | `8.14.1-r2` | `8.14.1` | ⚠️ YES — HTTP/TLS |
| libz | `zlib-static` | `1.3.2-r0` | `1.3.2` | LOW — compression only |
| liburing | `liburing-static` | (from dev pkg) | `2.6 (major=2, minor=6)` | LOW — io syscall wrapper |
| libucontext | `libucontext-dev` | `1.2-r3` | `1.2` | LOW — coroutine context API |

## What Needs Watching

**High priority (CVE-sensitive):**
- `libssl.a` / `libcrypto.a` — OpenSSL 3.3.7: subscribe to https://www.openssl.org/news/secadv/ or https://nvd.nist.gov (search "openssl")
- `libsodium.a` — libsodium 1.0.19: subscribe to https://github.com/jedisct1/libsodium/releases
- `libcurl.a` — curl 8.14.1: subscribe to https://curl.se/docs/security.html
- `libmariadb.a` — MariaDB 10.8.8: subscribe to https://mariadb.com/kb/en/security/

**Low priority (non-security, refresh on major version bumps only):**
- `libz.a` (zlib), `liburing.a`, `libucontext.a`

## Refresh Procedure

Run `scripts/vendor-static-libs.sh` to re-download and replace all archives.

**Trigger refresh when:**
1. A CVE is published for any security-critical library above
2. Quarterly check even if no CVE (quarterly rotation)
3. Upgrading Alpine base for container images (stay in sync with target OS version)

### Manual refresh (if script unavailable):

```bash
ALPINE_VER="v3.21"  # bump to latest Alpine
ALPINE_REPO="https://dl-cdn.alpinelinux.org/alpine/${ALPINE_VER}/main/x86_64"
cd third_party

# Check current index for package versions
wget -qO- ${ALPINE_REPO}/APKINDEX.tar.gz | tar -xzf - -O APKINDEX | grep -A3 "^P:openssl-libs-static"

# Download and extract each .apk (see scripts/vendor-static-libs.sh for full list)
# After replacing .a files and headers, run verification gate:
cd ..
./wolf build --static bench/real.wolf
ldd wolf_out/real  # must say "not a dynamic executable"
python3 bench/test_smuggling_live.py  # smuggling probes must all return 400
```

## Verification Gate (run after every refresh)

```bash
# 1. Build static server binary
./wolf build --static bench/real.wolf

# 2. Confirm static (zero deps)
ldd wolf_out/real         # → "not a dynamic executable"
readelf -d wolf_out/real | grep NEEDED  # → empty

# 3. Start server and run behavioral tests
./wolf_out/real &
sleep 2
curl -s -o /dev/null -w "%{http_code}" http://localhost:8084/   # → 200
python3 bench/test_smuggling_live.py                             # → 6/6 PASS
kill %1

# 4. Re-run internal test suite
go test -count=1 ./internal/... -timeout 120s
```
