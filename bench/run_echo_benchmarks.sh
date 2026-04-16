#!/bin/bash
set -e

echo "=== Benchmarking Echo Servers ==="
fuser -k 8081/tcp 8082/tcp 8083/tcp 8084/tcp 2>/dev/null || true
sleep 1

echo "--- 1. Go ---"
/home/askme/go-local/go/bin/go build -o bench/echo_go bench/echo.go
./bench/echo_go &
GO_PID=$!
sleep 2
./bench/load_tester -c 150 -n 100000 -url http://127.0.0.1:8081/ > bench/results_go.txt
kill $GO_PID

echo "--- 2. Node.js ---"
node bench/echo.js &
NODE_PID=$!
sleep 2
./bench/load_tester -c 150 -n 100000 -url http://127.0.0.1:8082/ > bench/results_node.txt
kill $NODE_PID

echo "--- 3. Python (asyncio) ---"
python3 bench/echo.py &
PY_PID=$!
sleep 2
./bench/load_tester -c 150 -n 100000 -url http://127.0.0.1:8083/ > bench/results_python.txt
kill $PY_PID

echo "--- 4. Wolf ---"
./wolf_out/echo &
WOLF_PID=$!
sleep 2
./bench/load_tester -c 150 -n 100000 -url http://127.0.0.1:8084/ > bench/results_wolf.txt
kill $WOLF_PID

echo "=== Done ==="
