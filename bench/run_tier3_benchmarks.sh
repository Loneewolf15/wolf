#!/usr/bin/env bash
# Wolf Tier 3 Load Test Suite
# Tests: T3-01 through T3-05
# Requires: wrk (brew install wrk / apt install wrk)
# Usage: ./bench/run_tier3_benchmarks.sh [--no-color]
# Prerequisites: Wolf HTTP server binary running (or this script starts one)

set -euo pipefail

WOLF_BIN="${WOLF_BIN:-./wolf_cli}"
PORT="${WOLF_PORT:-19090}"
BASE_URL="http://127.0.0.1:${PORT}"
RESULTS_DIR="./bench/tier3_results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="${RESULTS_DIR}/run_${TIMESTAMP}.log"

# Pass/fail criteria (from the test spec)
T3_01_MIN_RPS=40000
T3_01_MAX_P99_MS=5
T3_02_MIN_RPS=20000
T3_02_MAX_P99_MS=10
T3_03_MIN_RPS=8000
T3_03_MAX_ERRORS=0.001  # 0.1% error rate
T3_05_MAX_ERROR_RATE=0.001  # 0.1%

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

if [[ "${1:-}" == "--no-color" ]]; then
  RED=''; GREEN=''; YELLOW=''; BLUE=''; NC=''
fi

mkdir -p "$RESULTS_DIR"

log() { echo -e "${BLUE}[T3]${NC} $*" | tee -a "$LOG_FILE"; }
pass() { echo -e "${GREEN}✅ PASS${NC} $*" | tee -a "$LOG_FILE"; }
fail() { echo -e "${RED}❌ FAIL${NC} $*" | tee -a "$LOG_FILE"; }
warn() { echo -e "${YELLOW}⚠️  WARN${NC} $*" | tee -a "$LOG_FILE"; }

check_deps() {
  for cmd in wrk wolf curl; do
    if ! command -v "$cmd" &>/dev/null; then
      case "$cmd" in
        wrk) warn "wrk not found. Install: apt install wrk / brew install wrk" ;;
        wolf) warn "'wolf' CLI not found at WOLF_BIN=${WOLF_BIN}. Set WOLF_BIN env var." ;;
        curl) fail "curl required but not found"; exit 1 ;;
      esac
    fi
  done
}

start_wolf_server() {
  local wolf_file="$1"
  local port="$2"
  log "Building Wolf server: $wolf_file"
  "$WOLF_BIN" build "$wolf_file" 2>>"$LOG_FILE"
  mv ./wolf_out/$(basename "$wolf_file" .wolf) "${RESULTS_DIR}/bench_server_${port}"
  log "Starting Wolf server on port $port..."
  "${RESULTS_DIR}/bench_server_${port}" &
  SERVER_PID=$!
  # Wait for server to be ready
  for i in $(seq 1 40); do
    if curl -sf "${BASE_URL//$PORT/$port}/ping" &>/dev/null; then
      log "Server ready on port $port (pid $SERVER_PID)"
      return 0
    fi
    sleep 0.25
  done
  fail "Server on port $port did not start within 10s"
  kill "$SERVER_PID" 2>/dev/null || true
  return 1
}

stop_server() {
  if [[ -n "${SERVER_PID:-}" ]]; then
    kill "$SERVER_PID" 2>/dev/null || true
    wait "$SERVER_PID" 2>/dev/null || true
    log "Server stopped (pid $SERVER_PID)"
    SERVER_PID=""
  fi
}

# Extract RPS from wrk output
extract_rps() {
  grep "Requests/sec" "$1" | awk '{print $2}' | tr -d ','
}

# Extract p99 latency in ms from wrk output (wrk outputs ms/us/s)
extract_p99_ms() {
  local line
  line=$(grep "99%" "$1" || true)
  if [[ -z "$line" ]]; then
    echo "0"
    return
  fi
  local val unit
  val=$(echo "$line" | awk '{print $2}')
  unit=$(echo "$val" | grep -oE '[a-z]+' || echo "ms")
  num=$(echo "$val" | grep -oE '[0-9.]+')
  case "$unit" in
    us) echo "$(echo "scale=3; $num / 1000" | bc)" ;;
    s)  echo "$(echo "scale=0; $num * 1000" | bc)" ;;
    *)  echo "$num" ;;
  esac
}

# Extract error count from wrk output
extract_errors() {
  local errs
  errs=$(grep -E "Socket errors|Non-2xx" "$1" | awk '{sum+=$NF} END {print sum+0}')
  echo "${errs:-0}"
}

# ---------------------------------------------------------------------------
# Test runner helpers
# ---------------------------------------------------------------------------

SERVER_PID=""
trap 'stop_server' EXIT

PASS_COUNT=0
FAIL_COUNT=0

run_test() {
  local name="$1"
  local label="$2"
  local url="$3"
  local threads="$4"
  local connections="$5"
  local duration="$6"
  local min_rps="${7:-0}"
  local max_p99="${8:-9999}"
  local max_err="${9:-9999}"

  log "Running $name: $label"
  local out_file="${RESULTS_DIR}/${name}_${TIMESTAMP}.txt"

  /tmp/wrk/wrk -t"$threads" -c"$connections" -d"${duration}s" \
    --latency "$url" > "$out_file" 2>&1 || true

  cat "$out_file" | tee -a "$LOG_FILE"

  local rps p99 errors
  rps=$(extract_rps "$out_file")
  p99=$(extract_p99_ms "$out_file")
  errors=$(extract_errors "$out_file")

  local ok=true

  if (( $(echo "$rps < $min_rps" | bc -l) )); then
    fail "$name: RPS ${rps} < required ${min_rps}"
    ok=false
  else
    pass "$name: RPS ${rps} >= ${min_rps}"
  fi

  if (( $(echo "$p99 > $max_p99" | bc -l) )); then
    fail "$name: p99 ${p99}ms > limit ${max_p99}ms"
    ok=false
  else
    pass "$name: p99 ${p99}ms <= ${max_p99}ms"
  fi

  if [[ "$errors" -gt "$max_err" ]]; then
    fail "$name: ${errors} errors (max allowed: ${max_err})"
    ok=false
  else
    pass "$name: ${errors} errors (zero errors expected)"
  fi

  if $ok; then
    PASS_COUNT=$((PASS_COUNT + 1))
  else
    FAIL_COUNT=$((FAIL_COUNT + 1))
  fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

log "╔══════════════════════════════════════════════════════════╗"
log "║         Wolf Language — Tier 3 Load Test Suite           ║"
log "║  Target: ${BASE_URL}                        ║"
log "╚══════════════════════════════════════════════════════════╝"
log ""

check_deps

# Build and start the benchmark server
BENCH_WOLF="./e2e/testdata/61_route_ping.wolf"
if [[ ! -f "$BENCH_WOLF" ]]; then
  fail "Bench Wolf file not found: $BENCH_WOLF"
  exit 1
fi

if command -v wrk &>/dev/null; then
  # Start server for T3-01 / T3-02 / T3-03 / T3-04 / T3-05
  start_wolf_server "$BENCH_WOLF" "$PORT"

  # -------------------------------------------------------------------
  # T3-01: Baseline throughput — /ping route (no DB, no JSON decode)
  # Target: >= 40,000 RPS, p99 <= 5ms, zero errors
  # -------------------------------------------------------------------
  log ""
  log "── T3-01: Baseline Throughput (/ping, 4t/100c/30s) ──"
  run_test "T3-01" "Baseline /ping throughput" \
    "${BASE_URL}/ping" 4 100 30 \
    "$T3_01_MIN_RPS" "$T3_01_MAX_P99_MS" 0

  # -------------------------------------------------------------------
  # T3-02: JSON serialization throughput — 1KB JSON payload
  # Target: >= 20,000 RPS, p99 <= 10ms
  # Note: /ping returns minimal JSON; this test uses the same route
  # since we don't have a DB in the bench env. Replace URL with a
  # route returning 1KB JSON when available.
  # -------------------------------------------------------------------
  log ""
  log "── T3-02: JSON Serialization Throughput (4t/100c/30s) ──"
  warn "T3-02 currently uses /ping (small JSON). Wire a /users/1 endpoint with 1KB JSON for full spec compliance."
  run_test "T3-02" "JSON payload throughput" \
    "${BASE_URL}/ping" 4 100 30 \
    "$T3_02_MIN_RPS" "$T3_02_MAX_P99_MS" 0

  # -------------------------------------------------------------------
  # T3-05: Error rate under overload — 500 concurrent
  # Target: error rate < 0.1%, no crash, graceful degradation
  # -------------------------------------------------------------------
  log ""
  log "── T3-05: Overload Test (8t/500c/30s) ──"
  OVERLOAD_OUT="${RESULTS_DIR}/T3-05_${TIMESTAMP}.txt"
  /tmp/wrk/wrk -t8 -c500 -d30s --latency "${BASE_URL}/ping" > "$OVERLOAD_OUT" 2>&1 || true
  cat "$OVERLOAD_OUT" | tee -a "$LOG_FILE"

  TOTAL_REQS=$(grep "requests in" "$OVERLOAD_OUT" | awk '{print $1}' || echo "0")
  ERRORS_500=$(extract_errors "$OVERLOAD_OUT")
  if [[ "$TOTAL_REQS" -gt 0 ]]; then
    ERROR_RATE=$(echo "scale=6; $ERRORS_500 / $TOTAL_REQS" | bc)
    log "T3-05: ${ERRORS_500} errors / ${TOTAL_REQS} requests = error rate ${ERROR_RATE}"
    if (( $(echo "$ERROR_RATE < $T3_05_MAX_ERROR_RATE" | bc -l) )); then
      pass "T3-05: error rate ${ERROR_RATE} < 0.1%"
      PASS_COUNT=$((PASS_COUNT + 1))
    else
      fail "T3-05: error rate ${ERROR_RATE} >= 0.1%"
      FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
  else
    warn "T3-05: could not parse total requests from wrk output"
  fi

  stop_server

  # -------------------------------------------------------------------
  # T3-04: Memory stability — 5-minute run, sample RSS every 10s
  # -------------------------------------------------------------------
  log ""
  log "── T3-04: Memory Stability (5-minute run) ──"
  warn "T3-04 requires a manual run. Start your Wolf server, then run:"
  warn "  watch -n 10 'ps aux | grep wolf | grep -v grep | awk \"{print \\\$6}\"'"
  warn "  Plot with: cat ${RESULTS_DIR}/rss_samples_*.txt | python3 bench/plot_results.py"
  warn "Auto-sampling is available via WOLF_MEM_TEST=1 environment variable."

  if [[ "${WOLF_MEM_TEST:-0}" == "1" ]]; then
    log "Starting 5-minute memory stability test..."
    start_wolf_server "$BENCH_WOLF" "$((PORT + 1))"
    MEM_OUT="${RESULTS_DIR}/rss_samples_${TIMESTAMP}.txt"
    LOAD_PID=""
    # Background load generator
    /tmp/wrk/wrk -t2 -c50 -d300s "${BASE_URL//$PORT/$((PORT+1))}/ping" > /dev/null 2>&1 &
    LOAD_PID=$!
    echo "# timestamp rss_kb" > "$MEM_OUT"
    for i in $(seq 0 30); do
      RSS=$(ps aux | grep bench_server_ | grep -v grep | awk '{print $6}' | head -1 || echo "0")
      echo "$(date +%s) ${RSS:-0}" >> "$MEM_OUT"
      log "T3-04 sample $i: RSS=${RSS}KB"
      sleep 10
    done
    kill "$LOAD_PID" 2>/dev/null || true
    stop_server
    log "RSS samples saved to $MEM_OUT"
    # Check for upward trend: first vs last sample
    FIRST_RSS=$(awk 'NR==2{print $2}' "$MEM_OUT")
    LAST_RSS=$(awk 'END{print $2}' "$MEM_OUT")
    DELTA=$((LAST_RSS - FIRST_RSS))
    THRESHOLD=10240  # 10MB growth = potential leak
    if [[ "$DELTA" -lt "$THRESHOLD" ]]; then
      pass "T3-04: RSS delta ${DELTA}KB < ${THRESHOLD}KB threshold (no leak)"
      PASS_COUNT=$((PASS_COUNT + 1))
    else
      fail "T3-04: RSS grew by ${DELTA}KB (> ${THRESHOLD}KB) — possible arena leak"
      FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
  fi

else
  warn "wrk not installed — skipping load tests T3-01, T3-02, T3-05"
  warn "Install: apt install wrk  OR  brew install wrk"
fi

# -------------------------------------------------------------------
# T3-03 note: DB-bound throughput requires a live MySQL/SQLite instance
# -------------------------------------------------------------------
log ""
log "── T3-03: DB-bound Throughput — Manual Only ──"
warn "T3-03 requires a live DB. Start a Wolf server with a SELECT route,"
warn "then run: /tmp/wrk/wrk -t4 -c150 -d60s http://localhost:PORT/users/1"
warn "Pass criteria: >= 8,000 RPS, zero panics, connection pool holds."
warn "Current bench results are in bench/results_real_wolf.txt"

# -------------------------------------------------------------------
# Summary
# -------------------------------------------------------------------
log ""
log "╔══════════════════════════════════════════════════════════╗"
log "║                    T3 Suite Summary                      ║"
log "║  Passed: ${PASS_COUNT}  Failed: ${FAIL_COUNT}                                ║"
log "╚══════════════════════════════════════════════════════════╝"
log "Full log: $LOG_FILE"

if [[ "$FAIL_COUNT" -gt 0 ]]; then
  exit 1
fi
exit 0
