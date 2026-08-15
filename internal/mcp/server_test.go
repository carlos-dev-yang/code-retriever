package mcp

import (
	"bytes"
	"cidx/internal/app"
	"cidx/internal/index"
	"context"
	"encoding/json"
	"io"
	"sync"
	"testing"
	"time"
)

type fakeServices struct{}

func (fakeServices) Status(context.Context) (app.StatusResponse, error) {
	return app.StatusResponse{}, nil
}
func (fakeServices) Search(context.Context, app.SearchRequest) (any, error) { return nil, nil }
func (fakeServices) ReadSpan(context.Context, app.ReadSpanRequest) (app.ReadSpanResponse, error) {
	return app.ReadSpanResponse{}, nil
}
func (fakeServices) Reindex(context.Context, bool) (any, error) { return nil, nil }

type shortWriter struct{}

func (shortWriter) Write(data []byte) (int, error) { return len(data) - 1, nil }

type frameCapture struct{ frames chan []byte }

func (capture frameCapture) Write(data []byte) (int, error) {
	copy := append([]byte(nil), data...)
	capture.frames <- copy
	return len(data), nil
}

type blockingServices struct {
	started chan struct{}
	release <-chan struct{}
	once    sync.Once
}

func (service *blockingServices) Status(context.Context) (app.StatusResponse, error) {
	return app.StatusResponse{}, nil
}
func (service *blockingServices) Search(ctx context.Context, request app.SearchRequest) (any, error) {
	if request.Query != "slow" {
		return map[string]any{"query": request.Query}, nil
	}
	service.once.Do(func() { close(service.started) })
	select {
	case <-service.release:
		return map[string]any{"query": request.Query}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
func (service *blockingServices) ReadSpan(context.Context, app.ReadSpanRequest) (app.ReadSpanResponse, error) {
	return app.ReadSpanResponse{}, nil
}
func (service *blockingServices) Reindex(context.Context, bool) (any, error) {
	return map[string]any{}, nil
}

func writeRequest(t *testing.T, writer io.Writer, value string) {
	t.Helper()
	if _, err := io.WriteString(writer, value+"\n"); err != nil {
		t.Fatal(err)
	}
}
func readResponse(t *testing.T, frames <-chan []byte) response {
	t.Helper()
	select {
	case frame := <-frames:
		var value response
		if err := json.Unmarshal(bytes.TrimSpace(frame), &value); err != nil {
			t.Fatalf("stdout was not a JSON-RPC frame: %q: %v", frame, err)
		}
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for MCP response")
		return response{}
	}
}
func requestID(value response) string { return string(value.ID) }

const initializeFrame = `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`

func initializeSession(t *testing.T, writer io.Writer, frames <-chan []byte) {
	t.Helper()
	writeRequest(t, writer, initializeFrame)
	if result := readResponse(t, frames); requestID(result) != "0" || result.Error != nil {
		t.Fatalf("initialize=%+v", result)
	}
	writeRequest(t, writer, `{"jsonrpc":"2.0","method":"notifications/initialized"}`)
}

func TestFrameWriterRejectsShortWrite(t *testing.T) {
	writer := &frameWriter{writer: shortWriter{}}
	if err := writer.write(response{JSONRPC: "2.0", ID: json.RawMessage("1"), Result: map[string]any{}}); err == nil {
		t.Fatal("short write accepted")
	}
}

func TestServeCancellationClosesBlockingRead(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- (Server{Services: fakeServices{}}).Serve(ctx, reader, io.Discard) }()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("cancel result=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve remained blocked after cancellation")
	}
}

func TestInitializeAndToolRegistry(t *testing.T) {
	server := Server{}
	params := json.RawMessage(`{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}`)
	result, failure := server.dispatch(context.Background(), request{Method: "initialize", Params: params})
	if failure != nil {
		t.Fatalf("initialize: %v", failure)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" {
		t.Fatal("missing initialize response")
	}
	result, failure = server.dispatch(context.Background(), request{Method: "tools/list"})
	if failure != nil {
		t.Fatalf("list: %v", failure)
	}
	value := result.(map[string]any)
	tools := value["tools"].([]ToolDefinition)
	if len(tools) != 4 || tools[0].Name != "status" || tools[1].Name != "search" || tools[2].Name != "read_span" || tools[3].Name != "reindex" {
		t.Fatalf("registry=%#v", tools)
	}
}

func TestInitializeNegotiatesCurrentVersion(t *testing.T) {
	result, failure := (Server{}).dispatch(context.Background(), request{Method: "initialize", Params: json.RawMessage(`{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1"}}`)})
	if failure != nil {
		t.Fatalf("failure=%v", failure)
	}
	encoded, _ := json.Marshal(result)
	if !bytes.Contains(encoded, []byte(`"2025-11-25"`)) {
		t.Fatalf("result=%s", encoded)
	}
}

func TestInitializeRejectsRepeatForOneSession(t *testing.T) {
	server := Server{lifecycle: &lifecycle{}}
	params := json.RawMessage(`{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}`)
	if _, failure := server.dispatch(context.Background(), request{Method: "initialize", Params: params}); failure != nil {
		t.Fatal(failure)
	}
	if _, failure := server.dispatch(context.Background(), request{Method: "initialize", Params: params}); failure == nil || failure.Message != "ALREADY_INITIALIZED" {
		t.Fatalf("failure=%v", failure)
	}
}

func TestInitializeClaimsLifecycleAtomically(t *testing.T) {
	server := Server{lifecycle: &lifecycle{}}
	params := json.RawMessage(`{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}`)
	results := make(chan *Error, 2)
	for range 2 {
		go func() {
			_, failure := server.dispatch(context.Background(), request{Method: "initialize", Params: params})
			results <- failure
		}()
	}
	successes, repeats := 0, 0
	for range 2 {
		failure := <-results
		if failure == nil {
			successes++
		} else if failure.Message == "ALREADY_INITIALIZED" {
			repeats++
		}
	}
	if successes != 1 || repeats != 1 {
		t.Fatalf("successes=%d repeats=%d", successes, repeats)
	}
}

func TestMCPParamsAndUnknownToolFollow20251125(t *testing.T) {
	server := Server{Services: fakeServices{}, lifecycle: &lifecycle{initialized: true, ready: true}}
	if _, failure := server.dispatch(context.Background(), request{Method: "tools/list", Params: json.RawMessage(`{"cursor":"next","_meta":{}}`)}); failure != nil {
		t.Fatalf("tools/list params=%v", failure)
	}
	if _, failure := server.dispatch(context.Background(), request{Method: "tools/call", Params: json.RawMessage(`{"name":"status","arguments":{},"_meta":{}}`)}); failure != nil {
		t.Fatalf("tools/call params=%v", failure)
	}
	if _, failure := server.dispatch(context.Background(), request{Method: "tools/call", Params: json.RawMessage(`{"name":"not_a_tool","arguments":{}}`)}); failure == nil || failure.Code != invalidParams {
		t.Fatalf("unknown tool failure=%v", failure)
	}
	if _, failure := (Server{Services: fakeServices{}}).dispatch(context.Background(), request{Method: "initialize", Params: json.RawMessage(`{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"},"_meta":{}}`)}); failure != nil {
		t.Fatalf("initialize _meta=%v", failure)
	}
	for _, request := range []request{
		{Method: "tools/list", Params: json.RawMessage(`{"_meta":[]}`)},
		{Method: "tools/call", Params: json.RawMessage(`{"name":"status","arguments":{},"_meta":null}`)},
	} {
		if _, failure := server.dispatch(context.Background(), request); failure == nil || failure.Message != "INVALID_META" {
			t.Fatalf("invalid meta failure=%v", failure)
		}
	}
}

func TestServeCanonicalizesActiveRequestIDsAndPermitsCompletedReuse(t *testing.T) {
	input, writer := io.Pipe()
	frames := make(chan []byte, 12)
	started, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- (Server{Services: &blockingServices{started: started, release: release}}).Serve(context.Background(), input, frameCapture{frames: frames})
	}()
	initializeSession(t, writer, frames)
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":"\u0061","method":"tools/call","params":{"name":"search","arguments":{"query":"slow","max_inline_bytes":0}}}`)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("escaped-ID slow request did not start")
	}
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":"a","method":"tools/list"}`)
	if result := readResponse(t, frames); result.Error == nil || result.Error.Message != "DUPLICATE_REQUEST_ID" {
		t.Fatalf("escaped duplicate=%+v", result)
	}
	writeRequest(t, writer, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":"a"}}`)
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":-0,"method":"tools/call","params":{"name":"search","arguments":{"query":"slow","max_inline_bytes":0}}}`)
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":0,"method":"tools/list"}`)
	if result := readResponse(t, frames); result.Error == nil || result.Error.Message != "DUPLICATE_REQUEST_ID" {
		t.Fatalf("integer canonical duplicate=%+v", result)
	}
	writeRequest(t, writer, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":0}}`)
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if result := readResponse(t, frames); result.Error != nil {
		t.Fatalf("completed-ID first use=%+v", result)
	}
	deadline := time.After(time.Second)
	for {
		writeRequest(t, writer, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
		result := readResponse(t, frames)
		if result.Error == nil {
			break
		}
		if result.Error.Message != "DUPLICATE_REQUEST_ID" {
			t.Fatalf("completed-ID reuse=%+v", result)
		}
		select {
		case <-deadline:
			t.Fatal("completed ID remained active after response")
		default:
		}
	}
	for _, invalid := range []string{"null", "true", "{}", "1.5"} {
		writeRequest(t, writer, `{"jsonrpc":"2.0","id":`+invalid+`,"method":"tools/list"}`)
		if result := readResponse(t, frames); result.Error == nil || result.Error.Message != "INVALID_REQUEST_ID" || string(result.ID) != "null" {
			t.Fatalf("invalid id=%s result=%+v", invalid, result)
		}
	}
	close(release)
	_ = writer.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRequiredIntegerRejectsLossyAndNonIntegralJSON(t *testing.T) {
	for _, input := range []string{"1.0", "1e2", "9223372036854775808", "true"} {
		value := map[string]json.RawMessage{"value": json.RawMessage(input)}
		var target int
		if requiredInteger(value, "value", &target) {
			t.Fatalf("accepted %s as integer", input)
		}
	}
	value := map[string]json.RawMessage{"value": json.RawMessage(`-12`)}
	var target int
	if !requiredInteger(value, "value", &target) || target != -12 {
		t.Fatalf("exact integer=%d", target)
	}
}

type spanFailureServices struct{ fakeServices }

func (spanFailureServices) ReadSpan(context.Context, app.ReadSpanRequest) (app.ReadSpanResponse, error) {
	return app.ReadSpanResponse{}, app.ReadSpanError{Code: app.ReadSpanTooLarge, MaxBytes: 64}
}

func TestToolErrorPreservesReadSpanTypedData(t *testing.T) {
	result, failure := callTool(context.Background(), spanFailureServices{}, json.RawMessage(`{"name":"read_span","arguments":{"path":"a.go","start_line":1,"end_line":1,"expected_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`))
	if failure != nil {
		t.Fatalf("protocol failure=%v", failure)
	}
	tool := result.(callToolResult)
	payload, ok := tool.StructuredContent.(map[string]any)
	if !ok || !tool.IsError || payload["code"] != app.ReadSpanTooLarge || payload["max_bytes"] != 64 || !bytes.Contains([]byte(tool.Content[0].Text), []byte(`"max_bytes":64`)) {
		t.Fatalf("tool error=%#v", tool)
	}
}

func TestReadSpanRejectsInvalidWireRangesAndDigest(t *testing.T) {
	for _, arguments := range []string{
		`{"path":"a.go","start_line":0,"end_line":1,"expected_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
		`{"path":"a.go","start_line":2,"end_line":1,"expected_sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}`,
		`{"path":"a.go","start_line":1,"end_line":1,"expected_sha256":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}`,
	} {
		_, failure := callTool(context.Background(), fakeServices{}, json.RawMessage(`{"name":"read_span","arguments":`+arguments+`}`))
		if failure == nil || failure.Code != invalidParams || failure.Message != "INVALID_READ_SPAN_REQUEST" {
			t.Fatalf("arguments=%s failure=%v", arguments, failure)
		}
	}
}

func TestReindexRejectsNonBooleanDryRun(t *testing.T) {
	for _, dryRun := range []string{`null`, `0`, `"false"`} {
		_, failure := callTool(context.Background(), fakeServices{}, json.RawMessage(`{"name":"reindex","arguments":{"dry_run":`+dryRun+`}}`))
		if failure == nil || failure.Code != invalidParams || failure.Message != "INVALID_DRY_RUN" {
			t.Fatalf("dry_run=%s failure=%v", dryRun, failure)
		}
	}
}

func TestServeTreatsValidNonObjectFrameAsInvalidRequest(t *testing.T) {
	input, writer := io.Pipe()
	frames := make(chan []byte, 2)
	done := make(chan error, 1)
	go func() {
		done <- (Server{Services: fakeServices{}}).Serve(context.Background(), input, frameCapture{frames: frames})
	}()
	writeRequest(t, writer, `[]`)
	if result := readResponse(t, frames); result.Error == nil || result.Error.Code != invalidRequest || result.Error.Message != "INVALID_REQUEST" {
		t.Fatalf("array frame=%+v", result)
	}
	_ = writer.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestServeRejectsRequestsForNotifications(t *testing.T) {
	input, writer := io.Pipe()
	frames := make(chan []byte, 3)
	done := make(chan error, 1)
	go func() {
		done <- (Server{Services: fakeServices{}}).Serve(context.Background(), input, frameCapture{frames: frames})
	}()
	for _, frame := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"notifications/cancelled","params":{"requestId":1}}`,
	} {
		writeRequest(t, writer, frame)
		if result := readResponse(t, frames); result.Error == nil || result.Error.Code != invalidRequest || result.Error.Message != "INVALID_NOTIFICATION_REQUEST" {
			t.Fatalf("notification request=%+v", result)
		}
	}
	_ = writer.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestWireReindexNoopUsesCurrentManifestWithoutGeneration(t *testing.T) {
	value := wireReindex(reindexOutcome{Result: index.Result{Reused: 2}, Status: app.StatusResponse{ManifestSHA256: "current"}}).(map[string]any)
	if value["manifest_sha256"] != "current" {
		t.Fatalf("manifest=%#v", value)
	}
	if _, exists := value["activated_generation"]; exists {
		t.Fatalf("no-op advertised activation=%#v", value)
	}
}

func TestServeSaturationCancellationDuplicateAndStdoutPurity(t *testing.T) {
	input, writer := io.Pipe()
	frames := make(chan []byte, 16)
	started, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- (Server{Services: &blockingServices{started: started, release: release}, MaxConcurrent: 1}).Serve(context.Background(), input, frameCapture{frames: frames})
	}()
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":90,"method":"tools/list"}`)
	if result := readResponse(t, frames); requestID(result) != "90" || result.Error == nil || result.Error.Message != "SERVER_NOT_INITIALIZED" {
		t.Fatalf("pre-initialize response=%+v", result)
	}
	initializeSession(t, writer, frames)
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"slow","max_inline_bytes":0}}}`)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("slow request did not start")
	}
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"status","arguments":{}}}`)
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"status","arguments":{}}}`)
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"status","arguments":{}}}`)
	seenBusy, seenDuplicate := false, false
	for range 2 {
		result := readResponse(t, frames)
		if requestID(result) == "3" && result.Error != nil && result.Error.Message == "SERVER_BUSY" && result.Error.Code == serverBusyCode {
			seenBusy = true
		}
		if requestID(result) == "1" && result.Error != nil && result.Error.Message == "DUPLICATE_REQUEST_ID" {
			seenDuplicate = true
		}
	}
	if !seenBusy || !seenDuplicate {
		t.Fatalf("busy=%t duplicate=%t", seenBusy, seenDuplicate)
	}
	writeRequest(t, writer, `{"jsonrpc":"2.0","method":"notifications/cancelled","params":{"requestId":1}}`)
	if result := readResponse(t, frames); requestID(result) != "2" || result.Error != nil {
		t.Fatalf("queued request after cancellation=%+v", result)
	}
	close(release)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not stop at EOF")
	}
}

func TestServeConcurrentResponsesCanArriveOutOfOrder(t *testing.T) {
	input, writer := io.Pipe()
	frames := make(chan []byte, 8)
	started, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- (Server{Services: &blockingServices{started: started, release: release}, MaxConcurrent: 2}).Serve(context.Background(), input, frameCapture{frames: frames})
	}()
	initializeSession(t, writer, frames)
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search","arguments":{"query":"slow","max_inline_bytes":0}}}`)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("slow request did not start")
	}
	writeRequest(t, writer, `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"status","arguments":{}}}`)
	if result := readResponse(t, frames); requestID(result) != "2" || result.Error != nil {
		t.Fatalf("fast request was not returned first: %+v", result)
	}
	close(release)
	if result := readResponse(t, frames); requestID(result) != "1" || result.Error != nil {
		t.Fatalf("slow request=%+v", result)
	}
	_ = writer.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
