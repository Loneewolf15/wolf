#!/usr/bin/env bash
# scripts/vendor-static-libs.sh
#
# Downloads and extracts musl static libraries and headers from Alpine Linux
# for use with `wolf build --static` (zero-dependency server binary compilation).
#
# Usage:
#   ./scripts/vendor-static-libs.sh [ALPINE_VERSION]
#
# Example:
#   ./scripts/vendor-static-libs.sh v3.20     # (default)
#   ./scripts/vendor-static-libs.sh v3.21     # upgrade to Alpine 3.21
#
# After running, rebuild and verify:
#   ./wolf build --static bench/real.wolf
#   ldd wolf_out/real    # must say "not a dynamic executable"
#   go test -count=1 ./internal/... -timeout 120s

set -euo pipefail

ALPINE_VER="${1:-v3.20}"
ARCH="x86_64"
REPO="https://dl-cdn.alpinelinux.org/alpine/${ALPINE_VER}/main/${ARCH}"
THIRD_PARTY_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/third_party"
LIB_DIR="${THIRD_PARTY_DIR}/lib/linux_x64_musl"
INC_DIR="${THIRD_PARTY_DIR}/include"
TMP_DIR="$(mktemp -d)"

cleanup() { rm -rf "${TMP_DIR}"; }
trap cleanup EXIT

echo "==> Vendoring musl static libs from Alpine ${ALPINE_VER}"
echo "    Repo:    ${REPO}"
echo "    Lib out: ${LIB_DIR}"
echo "    Inc out: ${INC_DIR}"
echo ""

mkdir -p "${LIB_DIR}" "${INC_DIR}"

# ---------------------------------------------------------------------------
# Helper: download and extract specific paths from an .apk
# Usage: extract_apk <package-name> <path-in-apk...>
# ---------------------------------------------------------------------------
extract_apk() {
    local pkg="$1"; shift
    local paths=("$@")
    local apk_file="${TMP_DIR}/${pkg}.apk"

    echo "  [fetch] ${pkg}"
    wget -q "${REPO}/${pkg}" -O "${apk_file}"

    echo "  [extract] ${pkg}: ${paths[*]}"
    for path in "${paths[@]}"; do
        tar -xzf "${apk_file}" -C "${TMP_DIR}" "${path}" 2>/dev/null || true
    done
}

# ---------------------------------------------------------------------------
# Query current package versions from APKINDEX
# ---------------------------------------------------------------------------
echo "==> Querying APKINDEX for current package versions..."
APKINDEX_FILE="${TMP_DIR}/APKINDEX"
wget -q "${REPO}/APKINDEX.tar.gz" -O "${TMP_DIR}/APKINDEX.tar.gz"
tar -xzf "${TMP_DIR}/APKINDEX.tar.gz" -C "${TMP_DIR}" APKINDEX

get_pkg_version() {
    grep -A1 "^P:$1$" "${APKINDEX_FILE}" | grep "^V:" | cut -d: -f2
}

OPENSSL_VER=$(get_pkg_version "openssl-libs-static")
MARIADB_VER=$(get_pkg_version "mariadb-static")
SODIUM_VER=$(get_pkg_version "libsodium-static")
CURL_STATIC_VER=$(get_pkg_version "curl-static")
CURL_DEV_VER=$(get_pkg_version "curl-dev")
ZLIB_VER=$(get_pkg_version "zlib-static")
LIBURING_VER=$(get_pkg_version "liburing-static")
LIBUCONTEXT_VER=$(get_pkg_version "libucontext-dev")

echo ""
echo "  Versions found:"
echo "    openssl-libs-static:  ${OPENSSL_VER}"
echo "    mariadb-static:       ${MARIADB_VER}"
echo "    libsodium-static:     ${SODIUM_VER}"
echo "    curl-static:          ${CURL_STATIC_VER}"
echo "    curl-dev:             ${CURL_DEV_VER}"
echo "    zlib-static:          ${ZLIB_VER}"
echo "    liburing-static:      ${LIBURING_VER}"
echo "    libucontext-dev:      ${LIBUCONTEXT_VER}"
echo ""

# ---------------------------------------------------------------------------
# 1. OpenSSL (libssl.a + libcrypto.a + headers)
# ---------------------------------------------------------------------------
echo "==> OpenSSL ${OPENSSL_VER}"
extract_apk "openssl-libs-static-${OPENSSL_VER}.apk" \
    "usr/lib/libssl.a" "usr/lib/libcrypto.a"
cp "${TMP_DIR}/usr/lib/libssl.a"   "${LIB_DIR}/"
cp "${TMP_DIR}/usr/lib/libcrypto.a" "${LIB_DIR}/"
rm -rf "${TMP_DIR}/usr"

OPENSSL_DEV_VER=$(get_pkg_version "openssl-dev")
extract_apk "openssl-dev-${OPENSSL_DEV_VER}.apk" "usr/include/openssl"
rm -rf "${INC_DIR}/openssl"
cp -r "${TMP_DIR}/usr/include/openssl" "${INC_DIR}/"
rm -rf "${TMP_DIR}/usr"

# ---------------------------------------------------------------------------
# 2. libsodium (libsodium.a + headers)
# ---------------------------------------------------------------------------
echo "==> libsodium ${SODIUM_VER}"
extract_apk "libsodium-static-${SODIUM_VER}.apk" "usr/lib/libsodium.a"
cp "${TMP_DIR}/usr/lib/libsodium.a" "${LIB_DIR}/"
rm -rf "${TMP_DIR}/usr"

SODIUM_DEV_VER=$(get_pkg_version "libsodium-dev")
extract_apk "libsodium-dev-${SODIUM_DEV_VER}.apk" "usr/include/sodium.h" "usr/include/sodium"
rm -rf "${INC_DIR}/sodium" "${INC_DIR}/sodium.h"
cp -r "${TMP_DIR}/usr/include/sodium" "${INC_DIR}/"
cp "${TMP_DIR}/usr/include/sodium.h" "${INC_DIR}/"
rm -rf "${TMP_DIR}/usr"

# ---------------------------------------------------------------------------
# 3. MariaDB (libmariadb.a + headers)
# ---------------------------------------------------------------------------
echo "==> MariaDB ${MARIADB_VER}"
extract_apk "mariadb-static-${MARIADB_VER}.apk" "usr/lib/libmariadb.a"
cp "${TMP_DIR}/usr/lib/libmariadb.a" "${LIB_DIR}/"
rm -rf "${TMP_DIR}/usr"

MARIADB_DEV_VER=$(get_pkg_version "mariadb-dev")
extract_apk "mariadb-dev-${MARIADB_DEV_VER}.apk" "usr/include/mysql"
rm -rf "${INC_DIR}/mysql"
cp -r "${TMP_DIR}/usr/include/mysql" "${INC_DIR}/"
rm -rf "${TMP_DIR}/usr"

# ---------------------------------------------------------------------------
# 4. curl (libcurl.a + headers)
# ---------------------------------------------------------------------------
echo "==> curl ${CURL_STATIC_VER}"
extract_apk "curl-static-${CURL_STATIC_VER}.apk" "usr/lib/libcurl.a"
cp "${TMP_DIR}/usr/lib/libcurl.a" "${LIB_DIR}/"
rm -rf "${TMP_DIR}/usr"

extract_apk "curl-dev-${CURL_DEV_VER}.apk" "usr/include/curl"
rm -rf "${INC_DIR}/curl"
cp -r "${TMP_DIR}/usr/include/curl" "${INC_DIR}/"
rm -rf "${TMP_DIR}/usr"

# ---------------------------------------------------------------------------
# 5. zlib (libz.a)
# ---------------------------------------------------------------------------
echo "==> zlib ${ZLIB_VER}"
extract_apk "zlib-static-${ZLIB_VER}.apk" "lib/libz.a"
cp "${TMP_DIR}/lib/libz.a" "${LIB_DIR}/"
rm -rf "${TMP_DIR}/lib"

# ---------------------------------------------------------------------------
# 6. liburing (liburing.a + headers)
# ---------------------------------------------------------------------------
echo "==> liburing ${LIBURING_VER}"
extract_apk "liburing-static-${LIBURING_VER}.apk" "usr/lib/liburing.a"
cp "${TMP_DIR}/usr/lib/liburing.a" "${LIB_DIR}/"
rm -rf "${TMP_DIR}/usr"

LIBURING_DEV_VER=$(get_pkg_version "liburing-dev")
extract_apk "liburing-dev-${LIBURING_DEV_VER}.apk" "usr/include/liburing.h" "usr/include/liburing"
rm -rf "${INC_DIR}/liburing" "${INC_DIR}/liburing.h"
cp -r "${TMP_DIR}/usr/include/liburing" "${INC_DIR}/" 2>/dev/null || true
cp "${TMP_DIR}/usr/include/liburing.h" "${INC_DIR}/" 2>/dev/null || true
rm -rf "${TMP_DIR}/usr"

# ---------------------------------------------------------------------------
# 7. libucontext (libucontext.a + libucontext_posix.a + headers + shim)
# ---------------------------------------------------------------------------
echo "==> libucontext ${LIBUCONTEXT_VER}"
extract_apk "libucontext-dev-${LIBUCONTEXT_VER}.apk" \
    "lib/libucontext.a" "lib/libucontext_posix.a" "usr/include/libucontext"
cp "${TMP_DIR}/lib/libucontext.a" "${LIB_DIR}/"
cp "${TMP_DIR}/lib/libucontext_posix.a" "${LIB_DIR}/"
rm -rf "${INC_DIR}/libucontext"
cp -r "${TMP_DIR}/usr/include/libucontext" "${INC_DIR}/"
rm -rf "${TMP_DIR}/lib" "${TMP_DIR}/usr"

# Regenerate the ucontext.h shim (maps POSIX API to libucontext equivalents)
cat > "${INC_DIR}/ucontext.h" << 'EOF'
#ifndef UCONTEXT_H
#define UCONTEXT_H

#include <libucontext/libucontext.h>

#define ucontext_t libucontext_ucontext_t
#define mcontext_t libucontext_mcontext_t
#define getcontext libucontext_getcontext
#define setcontext libucontext_setcontext
#define swapcontext libucontext_swapcontext
#define makecontext libucontext_makecontext

#endif
EOF
echo "  [shim] ucontext.h regenerated"

# ---------------------------------------------------------------------------
# Update VERSIONS.md
# ---------------------------------------------------------------------------
echo ""
echo "==> Updating third_party/VERSIONS.md"
VERSIONS_FILE="${THIRD_PARTY_DIR}/VERSIONS.md"
TODAY=$(date +%Y-%m-%d)

# Update the bundled date and Alpine version in the existing file
sed -i "s/| Alpine Linux Version .*/| Alpine Linux Version | ${ALPINE_VER} |/" "${VERSIONS_FILE}"
sed -i "s/| Bundled Date .*/| Bundled Date | ${TODAY} |/" "${VERSIONS_FILE}"

echo ""
echo "=== DONE ==="
echo ""
echo "Libraries updated in ${LIB_DIR}:"
ls -lh "${LIB_DIR}"/*.a
echo ""
echo "NEXT STEPS — run the verification gate:"
echo "  1. Rebuild compiler:  go build -o wolf ./cmd/wolf"
echo "  2. Static build:      ./wolf build --static bench/real.wolf"
echo "  3. Check static:      ldd wolf_out/real"
echo "  4. Smuggling probes:  python3 bench/test_smuggling_live.py (with server running)"
echo "  5. Internal tests:    go test -count=1 ./internal/... -timeout 120s"
echo ""
echo "If all pass: commit third_party/ with 'chore(deps): refresh musl static bundle to Alpine ${ALPINE_VER}'"
