#!/usr/bin/env bash
# =============================================================================
# test_local.sh
#
# Runs the full Wolf test suite locally, exactly mirroring what CI does.
# Run this before pushing to catch failures early.
#
# Usage:
#   chmod +x test_local.sh
#   ./test_local.sh
#
# Run from the ROOT of your wolf repo:
#   cd /path/to/wolf
#   ./test_local.sh
#
# Optional flags:
#   --no-fmt          Skip gofmt check
#   --no-build        Skip binary build
#   --no-e2e          Skip e2e tests (run unit tests only)
#   --no-cross        Skip cross-platform build check
#   --fix-fmt         Auto-fix formatting issues (runs gofmt -w -s .)
# =============================================================================

set -euo pipefail

# Workaround for missing go in basic PATH
export PATH=$HOME/go-local/go/bin:$PATH

# ─── Flags ────────────────────────────────────────────────────────────────────
RUN_FMT=true
RUN_BUILD=true
RUN_E2E=true
RUN_CROSS=true
FIX_FMT=false
SKIP_SLOW=false   # --no-slow skips pythonenv tests (each does a real pip install, ~5-8 min total)

for arg in "$@"; do
  case $arg in
    --no-fmt)   RUN_FMT=false   ;;
    --no-build) RUN_BUILD=false ;;
    --no-e2e)   RUN_E2E=false   ;;
    --no-cross) RUN_CROSS=false ;;
    --fix-fmt)  FIX_FMT=true    ;;
    --no-slow)  SKIP_SLOW=true  ;;
  esac
done

# ─── Colours ──────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[1;34m'; BOLD='\033[1m'; RESET='\033[0m'

info()  { echo -e "${BLUE}[INFO]${RESET}  $*"; }
ok()    { echo -e "${GREEN}[ OK ]${RESET}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${RESET}  $*"; }
fail()  { echo -e "${RED}[FAIL]${RESET}  $*"; }
header(){ echo -e "\n${BOLD}━━━  $*  ━━━${RESET}"; }

# ─── Tracking ─────────────────────────────────────────────────────────────────
PASS=0
FAIL=0
SKIP=0
FAILED_STEPS=()

pass() { PASS=$((PASS+1)); ok "$1"; }
skip() { SKIP=$((SKIP+1)); warn "SKIPPED: $1"; }
die()  {
  FAIL=$((FAIL+1))
  FAILED_STEPS+=("$1")
  fail "$1"
  if [[ "${2:-}" == "fatal" ]]; then
    summary
    exit 1
  fi
}

# ─── Summary ──────────────────────────────────────────────────────────────────
summary() {
  echo ""
  echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  echo -e "${BOLD}  Results: ${GREEN}${PASS} passed${RESET}  ${RED}${FAIL} failed${RESET}  ${YELLOW}${SKIP} skipped${RESET}"
  echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${RESET}"
  if [[ ${#FAILED_STEPS[@]} -gt 0 ]]; then
    echo -e "${RED}Failed steps:${RESET}"
    for s in "${FAILED_STEPS[@]}"; do
      echo -e "  ${RED}✗${RESET} $s"
    done
  fi
  echo ""
}

# ─── Preflight ────────────────────────────────────────────────────────────────
header "Preflight Checks"

# Must be run from repo root
if [[ ! -f "go.mod" ]] || ! grep -q "wolflang/wolf" go.mod 2>/dev/null; then
  fail "Run this script from the root of the wolf repository"
  fail "Current directory: $(pwd)"
  exit 1
fi
ok "Running from repo root: $(pwd)"

# Required tools
for tool in go gofmt clang llc; do
  if command -v "$tool" >/dev/null 2>&1; then
    ok "$tool found: $(command -v $tool)  ($(${tool} --version 2>&1 | head -1))"
  else
    if [[ "$tool" == "clang" || "$tool" == "llc" ]]; then
      warn "$tool not found — e2e tests will fail (install LLVM: sudo apt install clang llvm)"
    else
      fail "Required tool missing: $tool" fatal
    fi
  fi
done

# Check system libs needed for wolf_runtime.c compilation
header "System Library Checks"

check_lib() {
  local name="$1" header_path="$2" pkg_hint="$3"
  if [[ -f "$header_path" ]]; then
    ok "$name headers found: $header_path"
  else
    warn "$name headers not found at $header_path"
    warn "  → Install with: $pkg_hint"
    warn "  → e2e tests may fail without this"
  fi
}

check_lib "hiredis"      "/usr/include/hiredis/hiredis.h"  "sudo apt install libhiredis-dev"
check_lib "MySQL client" "/usr/include/mysql/mysql.h"       "sudo apt install libmysqlclient-dev"

# ─── Step 1: Format ───────────────────────────────────────────────────────────
header "Step 1: Go Format Check"

if [[ "$RUN_FMT" == "false" ]]; then
  skip "gofmt (--no-fmt)"
else
  UNFORMATTED=$(gofmt -l . 2>/dev/null | grep '\.go$' || true)
  if [[ -n "$UNFORMATTED" ]]; then
    fail "gofmt — these files need formatting:"
    echo "$UNFORMATTED" | sed "s/^/    ${RED}✗${RESET} /"
    if [[ "$FIX_FMT" == "true" ]]; then
      info "Auto-fixing with gofmt -s -w ..."
      gofmt -s -w .
      ok "gofmt — files reformatted"
      pass "gofmt (auto-fixed)"
    else
      die "gofmt" "non-fatal"
      warn "  → Re-run with --fix-fmt to auto-fix, or run: gofmt -s -w ."
    fi
  else
    pass "gofmt — all files correctly formatted"
  fi
fi

# ─── Step 2: Build ────────────────────────────────────────────────────────────
header "Step 2: Build Wolf Binary"

if [[ "$RUN_BUILD" == "false" ]]; then
  skip "build (--no-build)"
else
  info "Running: make build"
  if make build 2>&1; then
    pass "make build"
  else
    die "make build" "fatal"
  fi
fi

# ─── Step 3: Unit + Integration Tests ─────────────────────────────────────────
header "Step 3: Unit & Integration Tests (with coverage)"

# pythonenv tests do REAL pip installs into real venvs — each takes 100-200s.
# Four of them back-to-back = ~10 min, which exceeds the default 10m go test
# timeout. We run them with -timeout 30m. Pass --no-slow to skip them entirely.

if [[ "$SKIP_SLOW" == "true" ]]; then
  info "Running fast tests only (skipping pythonenv — use --no-slow to re-enable)"
  TEST_PKGS=$(go list ./... | grep -v "internal/pythonenv" | tr '\n' ' ')
  TEST_CMD="go test $TEST_PKGS -v -cover -coverprofile=coverage.out -timeout 10m"
else
  info "Running all tests including slow pythonenv pip-install tests (~10 min)"
  info "Tip: use --no-slow to skip pythonenv tests for fast iteration"
  TEST_CMD="go test ./... -v -cover -coverprofile=coverage.out -timeout 30m"
fi

echo ""
info "Running: $TEST_CMD"
echo ""

TEST_LOG=$(mktemp)
if eval "$TEST_CMD" 2>&1 | tee "$TEST_LOG"; then
  echo ""
  pass "go test ./..."
else
  echo ""
  die "go test ./..." "non-fatal"
  info "Showing last 30 lines of test output:"
  tail -30 "$TEST_LOG"
fi
rm -f "$TEST_LOG"

# Coverage summary
if [[ -f coverage.out ]]; then
  echo ""
  info "Coverage summary:"
  go tool cover -func=coverage.out | grep -E "^total|FAIL" | \
    awk '{printf "  %-50s %s\n", $1, $3}'
  echo ""
  info "Per-package coverage:"
  go tool cover -func=coverage.out | grep "^github" | \
    grep -v "_test.go" | \
    awk '{printf "  %-60s %s\n", $1, $3}' | sort
fi

# ─── Step 4: E2E Breakdown ────────────────────────────────────────────────────
header "Step 4: E2E Test Breakdown"

if [[ "$RUN_E2E" == "false" ]]; then
  skip "e2e tests (--no-e2e)"
else
  info "Running e2e tests in isolation for detailed output..."
  echo ""

  E2E_PASS=0
  E2E_FAIL=0
  E2E_FAILED_NAMES=()

  # Run e2e package with verbose output, parse results
  E2E_OUT=$(go test ./e2e/... -v -timeout 1800s 2>&1 || true)

  # Parse individual test results.
  # NOTE: go test -v indents subtest results with 4 spaces, e.g.:
  #   "    --- PASS: TestEndToEnd/01_hello.wolf (6.85s)"
  # The ^ anchor would break the match, so we use unanchored patterns.
  while IFS= read -r line; do
    if [[ "$line" =~ "--- PASS: TestEndToEnd/" ]]; then
      name=$(echo "$line" | sed 's/.*--- PASS: TestEndToEnd\///' | awk '{print $1}')
      echo -e "  ${GREEN}✓${RESET} $name"
      E2E_PASS=$((E2E_PASS+1))
    elif [[ "$line" =~ "--- FAIL: TestEndToEnd/" ]]; then
      name=$(echo "$line" | sed 's/.*--- FAIL: TestEndToEnd\///' | awk '{print $1}')
      echo -e "  ${RED}✗${RESET} $name"
      E2E_FAIL=$((E2E_FAIL+1))
      E2E_FAILED_NAMES+=("$name")
    fi
  done <<< "$E2E_OUT"

  echo ""
  echo -e "  E2E: ${GREEN}${E2E_PASS} passed${RESET}  ${RED}${E2E_FAIL} failed${RESET}"

  if [[ $E2E_FAIL -gt 0 ]]; then
    echo ""
    warn "Failed e2e tests — showing error details:"
    echo ""
    # Extract and print the error messages for failed tests
    current_test=""
    in_fail=false
    while IFS= read -r line; do
      if [[ "$line" =~ "=== RUN   TestEndToEnd/" ]]; then
        current_test=$(echo "$line" | sed 's/=== RUN   TestEndToEnd\///')
        in_fail=false
      fi
      if [[ "$line" =~ "Output mismatch"|"Compilation failed"|"execution failed" ]]; then
        in_fail=true
        echo -e "  ${RED}[$current_test]${RESET}"
      fi
      if [[ "$in_fail" == "true" ]]; then
        echo "    $line"
        if [[ "$line" =~ "^$" || "$line" =~ "=== RUN" ]]; then
          in_fail=false
        fi
      fi
    done <<< "$E2E_OUT"

    die "e2e tests ($E2E_FAIL failures)" "non-fatal"
  else
    pass "e2e tests (all ${E2E_PASS} passed)"
  fi
fi

# ─── Step 5: Cross-platform build check ───────────────────────────────────────
header "Step 5: Cross-Platform Build Check"

if [[ "$RUN_CROSS" == "false" ]]; then
  skip "cross-platform builds (--no-cross)"
else
  PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
  )

  CROSS_FAIL=0
  for platform in "${PLATFORMS[@]}"; do
    OS="${platform%/*}"
    ARCH="${platform#*/}"
    EXT=""
    [[ "$OS" == "windows" ]] && EXT=".exe"
    OUT="/tmp/wolf-${OS}-${ARCH}${EXT}"

    if CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" \
       go build -ldflags="-s -w" -o "$OUT" ./cmd/wolf 2>/dev/null; then
      SIZE=$(du -sh "$OUT" 2>/dev/null | cut -f1)
      ok "${OS}/${ARCH} — ${SIZE}"
      rm -f "$OUT"
    else
      fail "${OS}/${ARCH}"
      CROSS_FAIL=$((CROSS_FAIL+1))
    fi
  done

  if [[ $CROSS_FAIL -gt 0 ]]; then
    die "cross-platform builds ($CROSS_FAIL failures)" "non-fatal"
  else
    pass "cross-platform builds (all 6 platforms)"
  fi
fi

# ─── Final summary ────────────────────────────────────────────────────────────
summary

if [[ $FAIL -gt 0 ]]; then
  echo -e "${RED}${BOLD}Local tests FAILED — fix above issues before pushing.${RESET}"
  exit 1
else
  echo -e "${GREEN}${BOLD}All checks passed — safe to push.${RESET}"
  echo -e "  Run: ${BOLD}./fix_and_push.sh${RESET}"
  exit 0
fi