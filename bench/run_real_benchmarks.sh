#!/bin/bash
set -e

echo "=== Benchmarking Real World DB/JSON Load ==="
fuser -k 8081/tcp 8082/tcp 8083/tcp 8084/tcp 2>/dev/null || true
sleep 1

echo "--- 1. Go ---"
/home/askme/go-local/go/bin/go build -o bench/real_go bench/real.go
./bench/real_go &
GO_PID=$!
sleep 2
./bench/load_tester -c 150 -n 100000 -url http://127.0.0.1:8081/ > bench/results_real_go.txt
kill $GO_PID

echo "--- 2. Node.js ---"
node bench/real.js &
NODE_PID=$!
sleep 2
./bench/load_tester -c 150 -n 100000 -url http://127.0.0.1:8082/ > bench/results_real_node.txt
kill $NODE_PID

echo "--- 3. Python (asyncio) ---"
python3 bench/real.py &
PY_PID=$!
sleep 2
./bench/load_tester -c 150 -n 100000 -url http://127.0.0.1:8083/ > bench/results_real_python.txt
kill $PY_PID

echo "--- 4. Wolf ---"
./wolf build bench/real.wolf
./wolf_out/real &
WOLF_PID=$!
sleep 2
./bench/load_tester -c 150 -n 100000 -url http://127.0.0.1:8084/ > bench/results_real_wolf.txt
kill $WOLF_PID

echo "=== Done ==="
