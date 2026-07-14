package lsp

import (
	"testing"
)

func TestLSPInit(t *testing.T) {
	h := NewHandler()

	params := InitializeParams{
		ProcessID: 1234,
		RootURI:   "file:///test",
	}

	res := h.Initialize(params)
	if res.Capabilities.TextDocumentSync != TextDocumentSyncKindFull {
		t.Errorf("Expected TextDocumentSyncKindFull, got %v", res.Capabilities.TextDocumentSync)
	}
}

func TestLSPDidChange(t *testing.T) {
	h := NewHandler()

	params := DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			URI: "file:///test.wolf",
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{Text: "class Test {}"},
		},
	}

	h.DidChange(params)

	if h.vfs["/test.wolf"] != "class Test {}" {
		t.Errorf("Expected VFS to have content 'class Test {}', got '%v'", h.vfs["/test.wolf"])
	}
}

func TestSplitFunc(t *testing.T) {
	data := []byte("Content-Length: 46\r\n\r\n{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}")

	adv, token, err := splitFunc(data, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if adv != len(data) {
		t.Errorf("expected advance %d, got %d", len(data), adv)
	}

	expectedToken := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"initialize\"}"
	if string(token) != expectedToken {
		t.Errorf("expected token '%s', got '%s'", expectedToken, string(token))
	}
}
