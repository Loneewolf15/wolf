package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Server struct {
	handler *Handler
}

func NewServer() *Server {
	return &Server{
		handler: NewHandler(),
	}
}

func (s *Server) Start() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(splitFunc)

	for scanner.Scan() {
		msg := scanner.Bytes()
		s.handleMessage(msg)
	}
}

func (s *Server) handleMessage(msg []byte) {
	// Parse as basic request to check ID
	var req RequestMessage
	if err := json.Unmarshal(msg, &req); err != nil {
		return
	}

	if req.ID != 0 {
		// It's a request
		s.handleRequest(msg, req.ID, req.Method)
	} else {
		// It's a notification
		var notif NotificationMessage
		if err := json.Unmarshal(msg, &notif); err == nil {
			s.handleNotification(msg, notif.Method)
		}
	}
}

func (s *Server) handleRequest(msg []byte, id int, method string) {
	var result interface{}
	var errObj *ErrorObj

	switch method {
	case "initialize":
		var req struct {
			Params InitializeParams `json:"params"`
		}
		json.Unmarshal(msg, &req)
		result = s.handler.Initialize(req.Params)
	case "textDocument/hover":
		var req struct {
			Params HoverParams `json:"params"`
		}
		json.Unmarshal(msg, &req)
		result = s.handler.Hover(req.Params)
	case "textDocument/definition":
		var req struct {
			Params DefinitionParams `json:"params"`
		}
		json.Unmarshal(msg, &req)
		result = s.handler.Definition(req.Params)
	case "textDocument/completion":
		var req struct {
			Params CompletionParams `json:"params"`
		}
		json.Unmarshal(msg, &req)
		result = s.handler.Completion(req.Params)
	case "shutdown":
		result = nil
	default:
		errObj = &ErrorObj{Code: -32601, Message: "Method not found"}
	}

	s.sendResponse(id, result, errObj)
}

func (s *Server) handleNotification(msg []byte, method string) {
	switch method {
	case "textDocument/didOpen":
		var req struct {
			Params DidOpenTextDocumentParams `json:"params"`
		}
		json.Unmarshal(msg, &req)
		s.handler.DidOpen(req.Params)
	case "textDocument/didChange":
		var req struct {
			Params DidChangeTextDocumentParams `json:"params"`
		}
		json.Unmarshal(msg, &req)
		s.handler.DidChange(req.Params)
	case "exit":
		os.Exit(0)
	}
}

func (s *Server) sendResponse(id int, result interface{}, errObj *ErrorObj) {
	resp := ResponseMessage{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   errObj,
	}
	s.writeMessage(resp)
}

func (s *Server) writeMessage(msg interface{}) {
	b, _ := json.Marshal(msg)
	fmt.Fprintf(os.Stdout, "Content-Length: %d\r\n\r\n%s", len(b), string(b))
}

func SendNotification(method string, params interface{}) {
	notif := NotificationMessage{
		JSONRPC: "2.0",
		Method:  method,
	}
	bParams, _ := json.Marshal(params)
	notif.Params = bParams
	
	// manually build so params is not double encoded if we don't want, wait params is []byte in struct
	// let's do a map
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	b, _ := json.Marshal(msg)
	fmt.Fprintf(os.Stdout, "Content-Length: %d\r\n\r\n%s", len(b), string(b))
}

func splitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		return 0, nil, nil
	}

	header := string(data[:headerEnd])
	contentLength := 0
	for _, line := range strings.Split(header, "\r\n") {
		if strings.HasPrefix(line, "Content-Length: ") {
			clStr := strings.TrimPrefix(line, "Content-Length: ")
			contentLength, _ = strconv.Atoi(clStr)
		}
	}

	if contentLength == 0 {
		return 0, nil, fmt.Errorf("missing Content-Length header")
	}

	totalLen := headerEnd + 4 + contentLength
	if len(data) >= totalLen {
		return totalLen, data[headerEnd+4 : totalLen], nil
	}

	return 0, nil, nil
}
