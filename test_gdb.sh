gdb -batch -ex "run" -ex "bt full" -ex "info threads" -ex "thread apply all bt" ./wolf_out/61_route_ping > gdb.log 2>&1
