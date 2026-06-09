#!/bin/bash
set -e

echo "Starting C Server..."
gcc -O3 http.c -o http_c
./http_c &
PID_C=$!
sleep 2
oha -z 5s -c 1000 http://127.0.0.1:8081 > c_oha.txt
kill $PID_C || true

echo "Starting Go Server..."
go build -o http_go http.go
./http_go &
PID_GO=$!
sleep 2
oha -z 5s -c 1000 http://127.0.0.1:8082 > go_oha.txt
kill $PID_GO || true

echo "Starting Node.js Server..."
node http.js &
PID_NODE=$!
sleep 2
oha -z 5s -c 1000 http://127.0.0.1:8083 > node_oha.txt
kill $PID_NODE || true

echo "Starting Python Server..."
python3 http_server.py &
PID_PY=$!
sleep 2
oha -z 5s -c 1000 http://127.0.0.1:8084 > py_oha.txt
kill $PID_PY || true

echo "Starting Wolf Server..."
go run ../cmd/wolf run http.wolf &
PID_WOLF=$!
sleep 5 # Wolf needs time to compile
oha -z 5s -c 1000 http://127.0.0.1:8085 > wolf_oha.txt
kill $PID_WOLF || true

echo "Benchmarking Complete!"
