#!/bin/bash
./wolf_out/server_drain &
SERVER_PID=$!
sleep 2
echo "--- Sending request..."
curl -v http://localhost:9998/ &
CURL_PID=$!
sleep 1
echo "--- Sending SIGINT to server ($SERVER_PID)..."
kill -SIGINT $SERVER_PID
echo "--- Waiting for curl ($CURL_PID)..."
wait $CURL_PID
echo "--- Curl finished."
wait $SERVER_PID
echo "--- Server finished."
