#!/bin/bash
rm -rf /tmp/wolf_compiler_assets
./wolf_cli build ./e2e/testdata/61_route_ping.wolf

gdb -batch -ex "run" -ex "thread apply all bt" -ex "quit" --args ./wolf_out/61_route_ping > gdb.log 2>&1 &
PID=$!

# Wait for server to start listening
while ! nc -z 127.0.0.1 19090; do   
  sleep 0.1
done

/tmp/wrk/wrk -t4 -c100 -d4s http://127.0.0.1:19090/ping

# Instead of killing gdb immediately, give it a moment to finish dumping trace after SIGSEGV
sleep 2
kill -9 $PID
cat gdb.log
