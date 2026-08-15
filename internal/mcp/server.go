package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
	"sync"
)

const protocolVersion = "2025-11-25"
const defaultConcurrentRequests = 8
const maxStdioFrameBytes = 16 << 20
const serverBusyCode = -32098

type Server struct {
	Services      Services
	MaxConcurrent int
	lifecycle     *lifecycle
}
type lifecycle struct {
	mutex              sync.Mutex
	initialized, ready bool
}
type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}
type task struct {
	request request
	context context.Context
	cancel  context.CancelFunc
}
type activeRequest struct {
	cancel     context.CancelFunc
	initialize bool
}

func (server Server) Serve(ctx context.Context, input io.Reader, output io.Writer) error {
	if server.Services == nil {
		return &Error{Code: internalError, Message: "SERVICES_REQUIRED"}
	}
	limit := server.MaxConcurrent
	if limit <= 0 {
		limit = defaultConcurrentRequests
	}
	root, cancel := context.WithCancel(ctx)
	defer cancel()
	server.lifecycle = &lifecycle{}
	if closer, ok := input.(io.ReadCloser); ok {
		go func() { <-root.Done(); _ = closer.Close() }()
	}
	writer := &frameWriter{writer: output, onFailure: cancel}
	queue := make(chan task, limit)
	var workers sync.WaitGroup
	active := map[string]activeRequest{}
	var activeMu sync.Mutex
	for range limit {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for item := range queue {
				if item.context.Err() == nil {
					server.handle(item.context, writer, item.request)
				}
				item.cancel()
				activeMu.Lock()
				key, _ := canonicalRequestID(item.request.ID)
				delete(active, key)
				activeMu.Unlock()
			}
		}()
	}
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), maxStdioFrameBytes)
	for scanner.Scan() {
		var generic any
		if err := json.Unmarshal(scanner.Bytes(), &generic); err != nil {
			if err := writer.write(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &Error{Code: parseError, Message: "PARSE_ERROR"}}); err != nil {
				break
			}
			continue
		}
		if _, object := generic.(map[string]any); !object {
			if err := writer.write(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &Error{Code: invalidRequest, Message: "INVALID_REQUEST"}}); err != nil {
				break
			}
			continue
		}
		var incoming request
		if err := json.Unmarshal(scanner.Bytes(), &incoming); err != nil {
			if err := writer.write(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &Error{Code: invalidRequest, Message: "INVALID_REQUEST"}}); err != nil {
				break
			}
			continue
		}
		if incoming.JSONRPC != "2.0" || incoming.Method == "" {
			_ = writer.write(response{JSONRPC: "2.0", ID: incoming.ID, Error: &Error{Code: invalidRequest, Message: "INVALID_REQUEST"}})
			continue
		}
		if incoming.Method == "notifications/cancelled" || incoming.Method == "notifications/initialized" {
			if len(incoming.ID) != 0 {
				id := incoming.ID
				if _, valid := canonicalRequestID(id); !valid {
					id = json.RawMessage("null")
				}
				_ = writer.write(response{JSONRPC: "2.0", ID: id, Error: &Error{Code: invalidRequest, Message: "INVALID_NOTIFICATION_REQUEST"}})
				continue
			}
		}
		if incoming.Method == "notifications/cancelled" {
			value, failure := decodeObject(incoming.Params, "requestId", "_meta")
			if failure == nil {
				requestID, exists := value["requestId"]
				key, valid := canonicalRequestID(requestID)
				if !exists || !valid {
					continue
				}
				activeMu.Lock()
				if target, exists := active[key]; exists && !target.initialize {
					target.cancel()
				}
				activeMu.Unlock()
			}
			continue
		}
		if incoming.Method == "notifications/initialized" {
			if _, failure := decodeObject(incoming.Params, "_meta"); failure == nil {
				server.markReady()
			}
			continue
		}
		if len(incoming.ID) == 0 {
			continue
		}
		key, valid := canonicalRequestID(incoming.ID)
		if !valid {
			_ = writer.write(response{JSONRPC: "2.0", ID: json.RawMessage("null"), Error: &Error{Code: invalidRequest, Message: "INVALID_REQUEST_ID"}})
			continue
		}
		activeMu.Lock()
		if _, duplicate := active[key]; duplicate {
			activeMu.Unlock()
			_ = writer.write(response{JSONRPC: "2.0", ID: incoming.ID, Error: &Error{Code: invalidRequest, Message: "DUPLICATE_REQUEST_ID"}})
			continue
		}
		requestCtx, requestCancel := context.WithCancel(root)
		active[key] = activeRequest{cancel: requestCancel, initialize: incoming.Method == "initialize"}
		activeMu.Unlock()
		item := task{request: incoming, context: requestCtx, cancel: requestCancel}
		select {
		case queue <- item:
		case <-root.Done():
			requestCancel()
		default:
			requestCancel()
			activeMu.Lock()
			delete(active, key)
			activeMu.Unlock()
			_ = writer.write(response{JSONRPC: "2.0", ID: incoming.ID, Error: &Error{Code: serverBusyCode, Message: "SERVER_BUSY"}})
		}
	}
	cancel()
	close(queue)
	workers.Wait()
	if err := writer.err(); err != nil {
		return err
	}
	if ctx.Err() != nil {
		return nil
	}
	return scanner.Err()
}
func (server Server) handle(ctx context.Context, writer *frameWriter, request request) {
	result, failure := server.dispatch(ctx, request)
	if ctx.Err() == nil {
		_ = writer.write(response{JSONRPC: "2.0", ID: request.ID, Result: result, Error: failure})
	}
}
func (server Server) dispatch(ctx context.Context, request request) (any, *Error) {
	if request.Method != "initialize" && (request.Method == "tools/list" || request.Method == "tools/call") && !server.ready() {
		return nil, &Error{Code: invalidRequest, Message: "SERVER_NOT_INITIALIZED"}
	}
	switch request.Method {
	case "initialize":
		value, err := decodeObject(request.Params, "protocolVersion", "capabilities", "clientInfo", "_meta")
		if err != nil {
			return nil, err
		}
		var version string
		if input, ok := value["protocolVersion"]; !ok || json.Unmarshal(input, &version) != nil || version == "" {
			return nil, &Error{Code: invalidParams, Message: "INVALID_INITIALIZE"}
		}
		var capabilities map[string]json.RawMessage
		var clientInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if input, ok := value["capabilities"]; !ok || json.Unmarshal(input, &capabilities) != nil || capabilities == nil {
			return nil, &Error{Code: invalidParams, Message: "INVALID_INITIALIZE_CAPABILITIES"}
		}
		if input, ok := value["clientInfo"]; !ok || json.Unmarshal(input, &clientInfo) != nil || clientInfo.Name == "" || clientInfo.Version == "" {
			return nil, &Error{Code: invalidParams, Message: "INVALID_INITIALIZE_CLIENT_INFO"}
		}
		if !server.claimInitialized() {
			return nil, &Error{Code: invalidRequest, Message: "ALREADY_INITIALIZED"}
		}
		return map[string]any{"protocolVersion": protocolVersion, "capabilities": map[string]any{"tools": map[string]any{}}, "serverInfo": map[string]string{"name": "cidx", "version": "v1"}}, nil
	case "tools/list":
		value, err := decodeObject(request.Params, "cursor", "_meta")
		if err != nil {
			return nil, err
		}
		if cursor, exists := value["cursor"]; exists {
			var value string
			if json.Unmarshal(cursor, &value) != nil {
				return nil, &Error{Code: invalidParams, Message: "INVALID_CURSOR"}
			}
		}
		return map[string]any{"tools": toolRegistry()}, nil
	case "tools/call":
		return callTool(ctx, server.Services, request.Params)
	case "ping":
		return map[string]any{}, nil
	default:
		return nil, &Error{Code: methodNotFound, Message: "METHOD_NOT_FOUND"}
	}
}

func canonicalRequestID(raw json.RawMessage) (string, bool) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return "", false
	}
	switch value := value.(type) {
	case string:
		return "string:" + value, true
	case json.Number:
		text := value.String()
		if strings.ContainsAny(text, ".eE") {
			return "", false
		}
		integer, ok := new(big.Int).SetString(text, 10)
		if !ok {
			return "", false
		}
		return "integer:" + integer.String(), true
	default:
		return "", false
	}
}
func (server Server) claimInitialized() bool {
	if server.lifecycle == nil {
		return true
	}
	server.lifecycle.mutex.Lock()
	defer server.lifecycle.mutex.Unlock()
	if server.lifecycle.initialized {
		return false
	}
	server.lifecycle.initialized = true
	return true
}
func (server Server) markReady() {
	if server.lifecycle == nil {
		return
	}
	server.lifecycle.mutex.Lock()
	if server.lifecycle.initialized {
		server.lifecycle.ready = true
	}
	server.lifecycle.mutex.Unlock()
}
func (server Server) ready() bool {
	if server.lifecycle == nil {
		return true
	}
	server.lifecycle.mutex.Lock()
	defer server.lifecycle.mutex.Unlock()
	return server.lifecycle.ready
}

type frameWriter struct {
	mutex     sync.Mutex
	writer    io.Writer
	onFailure func()
	failure   error
}

func (writer *frameWriter) write(value response) error {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	if writer.failure != nil {
		return writer.failure
	}
	data, err := json.Marshal(value)
	if err == nil {
		frame := append(data, '\n')
		written, writeErr := writer.writer.Write(frame)
		if writeErr != nil {
			err = writeErr
		} else if written != len(frame) {
			err = io.ErrShortWrite
		}
	}
	if err != nil {
		writer.failure = fmt.Errorf("write MCP frame: %w", err)
		if writer.onFailure != nil {
			writer.onFailure()
		}
	}
	return err
}
func (writer *frameWriter) err() error {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	return writer.failure
}
