#!/bin/bash
# run_registration_test.sh
set -e

DIR="/home/askme/Pictures/wolf-lang"
LOG="/tmp/registration_test.log"

echo "Stopping existing processes..."
pkill -f wolf_out/main || true
fuser -k 2006/tcp || true

echo "Starting Wolf API Server..."
cd "$DIR"
./wolf_out/main > "$LOG" 2>&1 &
SRV_PID=$!

echo "Waiting for server to initialize (3s)..."
sleep 5

echo "Firing Registration Request..."
NEW_MAIL="test_verified_$(date +%s)@wolftest.io"
curl -s -X POST -H 'Content-Type: application/json' \
     -d "{\"name\":\"Verified Wolf\",\"email\":\"$NEW_MAIL\",\"password\":\"wolfpass123\",\"confirm_password\":\"wolfpass123\",\"phone\":\"090001$(date +%s | tail -c 5)\",\"referral_code\":\"\"}" \
     http://localhost:2006/users/registerUser

echo ""
echo "Cleaning up..."
kill $SRV_PID || true
sleep 1

echo "=== FULL TEST LOG ==="
cat "$LOG"
