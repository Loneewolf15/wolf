#!/usr/bin/env bash
# =============================================================================
# fix_and_push.sh
#
# Applies all Wolf runtime fixes, runs gofmt on every Go file, then commits
# and pushes to origin/main.
#
# Usage:
#   chmod +x fix_and_push.sh
#   ./fix_and_push.sh
#
# Requirements:
#   - git configured with push access to github.com/Loneewolf15/wolf
#   - Go installed (for gofmt)
#   - The fixed wolf_runtime.c, wolf_runtime.h, and ci.yml files sitting
#     in the same directory as this script  (or update the paths below)
# =============================================================================

set -euo pipefail

# --------------------------------------------------------------------------- #
# Config — edit these if needed                                                #
# --------------------------------------------------------------------------- #
REPO_URL="https://github.com/Loneewolf15/wolf.git"
BRANCH="main"
COMMIT_MESSAGE="feat: implement graceful shutdown, request draining, and fix BUG-021 double-free"

# Paths to the fixed files produced in this session.
# If they live elsewhere, update these variables.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FIXED_RUNTIME_C="${SCRIPT_DIR}/runtime/wolf_runtime.c"
FIXED_RUNTIME_H="${SCRIPT_DIR}/runtime/wolf_runtime.h"
FIXED_CI_YML="${SCRIPT_DIR}/.github/workflows/ci.yml"
FIXED_MAKEFILE="${SCRIPT_DIR}/Makefile"
FIXED_SHUTDOWN_TEST="${SCRIPT_DIR}/e2e/shutdown_test.go"
FIXED_SHUTDOWN_WOLF="${SCRIPT_DIR}/e2e/testdata/server_shutdown.wolf"
FIXED_DRAIN_WOLF="${SCRIPT_DIR}/e2e/testdata/server_drain.wolf"

# --------------------------------------------------------------------------- #
# Helpers                                                                      #
# --------------------------------------------------------------------------- #
info()  { echo -e "\033[1;34m[INFO]\033[0m  $*"; }
ok()    { echo -e "\033[1;32m[ OK ]\033[0m  $*"; }
warn()  { echo -e "\033[1;33m[WARN]\033[0m  $*"; }
die()   { echo -e "\033[1;31m[FAIL]\033[0m  $*" >&2; exit 1; }

# --------------------------------------------------------------------------- #
# Preflight checks                                                              #
# --------------------------------------------------------------------------- #
info "Checking prerequisites..."

command -v git  >/dev/null 2>&1 || die "git is not installed"
command -v gofmt >/dev/null 2>&1 || die "gofmt is not installed (install Go)"

[[ -f "$FIXED_RUNTIME_C" ]] || die "Fixed wolf_runtime.c not found at: $FIXED_RUNTIME_C"
[[ -f "$FIXED_RUNTIME_H" ]] || die "Fixed wolf_runtime.h not found at: $FIXED_RUNTIME_H"
[[ -f "$FIXED_CI_YML"    ]] || die "Fixed ci.yml not found at: $FIXED_CI_YML"
[[ -f "$FIXED_MAKEFILE"  ]] || die "Fixed Makefile not found at: $FIXED_MAKEFILE"

ok "Prerequisites satisfied"

# --------------------------------------------------------------------------- #
# Clone or reuse existing repo                                                 #
# --------------------------------------------------------------------------- #
WORK_DIR="/tmp/wolf_fix_$$"

if [[ -d "$WORK_DIR" ]]; then
    warn "Work directory already exists, removing: $WORK_DIR"
    rm -rf "$WORK_DIR"
fi

info "Cloning $REPO_URL into $WORK_DIR ..."
git clone --branch "$BRANCH" --single-branch "$REPO_URL" "$WORK_DIR"
ok "Clone complete"

cd "$WORK_DIR"

# --------------------------------------------------------------------------- #
# Apply fixed files                                                            #
# --------------------------------------------------------------------------- #
info "Applying fixed runtime/wolf_runtime.c ..."
cp "$FIXED_RUNTIME_C" runtime/wolf_runtime.c
ok "runtime/wolf_runtime.c updated"

info "Applying fixed runtime/wolf_runtime.h ..."
cp "$FIXED_RUNTIME_H" runtime/wolf_runtime.h
ok "runtime/wolf_runtime.h updated"

info "Applying fixed .github/workflows/ci.yml ..."
mkdir -p .github/workflows
cp "$FIXED_CI_YML" .github/workflows/ci.yml
ok ".github/workflows/ci.yml updated"

info "Applying fixed Makefile ..."
cp "$FIXED_MAKEFILE" Makefile
ok "Makefile updated"

info "Applying new test files ..."
mkdir -p e2e/testdata
cp "$FIXED_SHUTDOWN_TEST" e2e/shutdown_test.go
cp "$FIXED_SHUTDOWN_WOLF" e2e/testdata/server_shutdown.wolf
cp "$FIXED_DRAIN_WOLF" e2e/testdata/server_drain.wolf
ok "Test files applied"

# --------------------------------------------------------------------------- #
# Run gofmt on every .go file in the repo                                     #
# --------------------------------------------------------------------------- #
info "Running gofmt on all Go source files..."

# -w  writes changes back in place
# -s  simplifies code (e.g. []T{x} → []T{x})
gofmt -w -s .

# Report what changed
FORMATTED=$(git diff --name-only | grep '\.go$' || true)
if [[ -n "$FORMATTED" ]]; then
    info "gofmt reformatted the following files:"
    echo "$FORMATTED" | sed 's/^/    /'
else
    ok "All Go files were already correctly formatted"
fi

# --------------------------------------------------------------------------- #
# Stage everything                                                             #
# --------------------------------------------------------------------------- #
info "Staging changes..."

git add runtime/wolf_runtime.c
git add runtime/wolf_runtime.h
git add .github/workflows/ci.yml
git add Makefile
git add e2e/shutdown_test.go
git add e2e/testdata/server_shutdown.wolf
git add e2e/testdata/server_drain.wolf

# Stage any Go files that gofmt touched
if [[ -n "$FORMATTED" ]]; then
    git add $(echo "$FORMATTED")
fi

# Show a clean diff summary before committing
echo ""
info "Staged diff summary:"
git diff --cached --stat
echo ""

# --------------------------------------------------------------------------- #
# Commit                                                                       #
# --------------------------------------------------------------------------- #
# Check there is actually something to commit
if git diff --cached --quiet; then
    warn "Nothing to commit — all files already match what is in the repo."
    warn "If you expected changes, check that the fixed files are different"
    warn "from what is already on disk in the repo."
    exit 0
fi

info "Committing..."
git commit -m "$COMMIT_MESSAGE"
ok "Committed: $COMMIT_MESSAGE"

# --------------------------------------------------------------------------- #
# Push                                                                         #
# --------------------------------------------------------------------------- #
info "Pushing to origin/$BRANCH ..."
git push origin "$BRANCH"
ok "Push complete — https://github.com/Loneewolf15/wolf/actions"

# --------------------------------------------------------------------------- #
# Cleanup                                                                      #
# --------------------------------------------------------------------------- #
info "Cleaning up work directory..."
rm -rf "$WORK_DIR"
ok "Done. CI pipeline should now be running."