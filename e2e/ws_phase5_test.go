//go:build ws_e2e
// +build ws_e2e

package e2e_test

import (
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketPhase5_Broadcast(t *testing.T) {
	if os.Getenv("WOLF_HTTP_TEST") == "" {
		t.Skip("set WOLF_HTTP_TEST=1 to run")
	}
	conn1, _, err := websocket.DefaultDialer.Dial("ws://localhost:8084/ws", nil)
	if err != nil {
		t.Fatal("conn1 dial:", err)
	}
	conn2, _, err := websocket.DefaultDialer.Dial("ws://localhost:8084/ws", nil)
	if err != nil {
		t.Fatal("conn2 dial:", err)
	}
	defer conn1.Close()
	defer conn2.Close()
	err = conn1.WriteMessage(websocket.TextMessage, []byte("broadcast:hello"))
	if err != nil {
		t.Fatal("write:", err)
	}
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn2.ReadMessage()
	if err != nil {
		t.Fatal("read:", err)
	}
	if string(msg) != "hello" {
		t.Fatalf("expected 'hello', got '%s'", string(msg))
	}
}

func TestWebSocketPhase5_PingPong(t *testing.T) {
	if os.Getenv("WOLF_HTTP_TEST") == "" {
		t.Skip("set WOLF_HTTP_TEST=1 to run")
	}
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8084/ws", nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	err = conn.WriteMessage(websocket.PingMessage, []byte{})
	if err != nil {
		t.Fatal("ping write:", err)
	}
}

func TestWebSocketPhase5_CloseWithDrain(t *testing.T) {
	if os.Getenv("WOLF_HTTP_TEST") == "" {
		t.Skip("set WOLF_HTTP_TEST=1 to run")
	}
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8084/ws", nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	err = conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		t.Fatal("close write:", err)
	}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatal("expected connection closed, got nil error")
	}
}

func TestWebSocketPhase5_OnClose(t *testing.T) {
	if os.Getenv("WOLF_HTTP_TEST") == "" {
		t.Skip("set WOLF_HTTP_TEST=1 to run")
	}
	conn, _, err := websocket.DefaultDialer.Dial("ws://localhost:8084/ws", nil)
	if err != nil {
		t.Fatal("dial:", err)
	}
	conn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	conn.ReadMessage()
}
