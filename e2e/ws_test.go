package e2e_test

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWebSocketHandshake(t *testing.T) {
	// 1. Compile and start the Wolf server
	wolfFile := filepath.Join("testdata", "31_websocket.wolf")
	// Use the pre-built wolf binary
	cmd := exec.Command("../wolf", "run", wolfFile)
	cmd.Env = append(os.Environ(), "WOLF_PORT=8082")
	logFile, _ := os.Create("server_ws.log")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	defer logFile.Close()

	err := cmd.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	// 2. Wait for server to boot with retries
	var conn net.Conn
	for i := 0; i < 45; i++ {
		conn, err = net.Dial("tcp", "127.0.0.1:8082")
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		t.Fatalf("Failed to connect after 15s: %v", err)
	}
	defer conn.Close()

	// Sample key from RFC 6455
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	expectedAccept := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="

	req := "GET / HTTP/1.1\r\n" +
		"Host: 127.0.0.1:8080\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"

	conn.SetDeadline(time.Now().Add(5 * time.Second))
	_, err = conn.Write([]byte(req))
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	buf := make([]byte, 2048)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	output := string(buf[:n])
	t.Logf("Server response:\n%s", output)

	if !strings.Contains(output, "101 Switching Protocols") {
		t.Errorf("Expected '101 Switching Protocols' in response")
	}
	if !strings.Contains(output, "Sec-WebSocket-Accept: "+expectedAccept) {
		t.Errorf("Expected Accept key '%s', got something else", expectedAccept)
	}
}

func TestWebSocketEcho(t *testing.T) {
	wolfFile := filepath.Join("testdata", "31_websocket.wolf")
	cmd := exec.Command("../wolf", "run", wolfFile)
	cmd.Env = append(os.Environ(), "WOLF_PORT=8083")
	logFile, _ := os.Create("server_echo.log")
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	defer logFile.Close()

	err := cmd.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	// Wait for server to boot with retries
	var conn net.Conn
	for i := 0; i < 45; i++ {
		conn, err = net.Dial("tcp", "127.0.0.1:8083")
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Handshake
	req := "GET / HTTP/1.1\r\n" +
		"Host: 127.0.0.1:8080\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
		"Sec-WebSocket-Version: 13\r\n\r\n"
	conn.Write([]byte(req))
	resp := make([]byte, 1024)
	n, _ := conn.Read(resp)
	if !strings.Contains(string(resp[:n]), "101 Switching Protocols") {
		t.Fatal("Handshake failed")
	}

	// Send "Hello Wolf!" (Masked Text Frame)
	// Header: 0x81 (FIN+Text), 0x8b (Masked, Len 11)
	// Mask: 0x11 0x22 0x33 0x44
	msg := "Hello Wolf!"
	mask := []byte{0x11, 0x22, 0x33, 0x44}
	maskedPayload := make([]byte, len(msg))
	for i := 0; i < len(msg); i++ {
		maskedPayload[i] = msg[i] ^ mask[i%4]
	}

	frame := append([]byte{0x81, 0x8B}, mask...)
	frame = append(frame, maskedPayload...)
	conn.Write(frame)

	// Read Echo Response (Unmasked Text Frame)
	echoBuf := make([]byte, 1024)
	totalRead := 0
	for totalRead < len("Echo: Hello Wolf!")+2 {
		n, err = conn.Read(echoBuf[totalRead:])
		if err != nil {
			t.Fatalf("Failed to read echo: %v (read %d bytes)", err, totalRead)
		}
		totalRead += n
	}

	echoOutput := string(echoBuf[:totalRead])
	t.Logf("Echo output: %s", echoOutput)
	if !strings.Contains(echoOutput, "Echo: Hello Wolf!") {
		t.Errorf("Expected 'Echo: Hello Wolf!', got: %s", echoOutput)
	}
}
