#!/bin/bash
./wolf build bench/real.wolf
./wolf_out/real 2>/dev/null &
WOLF_PID=$!
sleep 2
./bench/load_tester -c 150 -n 100000 -url http://127.0.0.1:8084/
kill $WOLF_PID
