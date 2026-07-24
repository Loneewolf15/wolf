#!/bin/bash
rm -rf /tmp/wolf_rt_cache
go install ./cmd/wolf
./wolf run e2e/testdata/31_websocket.wolf > server.log 2>&1 &
SERVER_PID=$!
sleep 5
cat << 'GO' > /tmp/test_ws.go
package main
import ("net"; "fmt")
func main() {
    conn, err := net.Dial("tcp", "127.0.0.1:8080")
    if err != nil { panic(err) }
    req := "GET / HTTP/1.1\r\nHost: 127.0.0.1:8080\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\nSec-WebSocket-Version: 13\r\n\r\n"
    conn.Write([]byte(req))
    resp := make([]byte, 1024)
    n, _ := conn.Read(resp)
    fmt.Println(string(resp[:n]))
    msg := "Hello Wolf!"
    mask := []byte{0x11, 0x22, 0x33, 0x44}
    maskedPayload := make([]byte, len(msg))
    for i := 0; i < len(msg); i++ { maskedPayload[i] = msg[i] ^ mask[i%4] }
    frame := append([]byte{0x81, 0x8B}, mask...)
    frame = append(frame, maskedPayload...)
    conn.Write(frame)
    echoBuf := make([]byte, 1024)
    n, _ = conn.Read(echoBuf)
    fmt.Printf("Received %d bytes: %s\n", n, string(echoBuf[:n]))
}
GO
go run /tmp/test_ws.go
kill $SERVER_PID
cat server.log
