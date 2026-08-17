package cli

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cidx/internal/app"
	"cidx/internal/config"
	"cidx/internal/index"
)

type frameSink struct{ frames chan []byte }

func (sink frameSink) Write(data []byte) (int, error) {
	sink.frames <- append([]byte(nil), data...)
	return len(data), nil
}

func TestStableCommandsRejectPositionalArguments(t *testing.T) {
	for _, args := range [][]string{
		{"init", "extra"},
		{"serve", "--root", ".", "extra"},
		{"status", "extra"},
		{"index", "extra"},
		{"embed", "extra"},
	} {
		err := Run(context.Background(), args, Dependencies{Stdout: io.Discard, Stderr: io.Discard})
		if err == nil || !strings.Contains(err.Error(), "positional") {
			t.Fatalf("args=%v err=%v", args, err)
		}
	}
}

func TestInitUsesServingDimensionThroughInitializer(t *testing.T) {
	var calls []struct {
		root string
		dim  int
	}
	deps := Dependencies{
		Stdout: io.Discard,
		Stderr: io.Discard,
		Initialize: func(_ context.Context, root string, dimension int) error {
			calls = append(calls, struct {
				root string
				dim  int
			}{root, dimension})
			return nil
		},
	}
	if err := Run(context.Background(), []string{"init", "--serving-dim", "512"}, deps); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].root != "." || calls[0].dim != 512 {
		t.Fatalf("initializer calls=%#v", calls)
	}
	if err := Run(context.Background(), []string{"init", "--serving-dim", "1024"}, deps); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[1].dim != 1024 {
		t.Fatalf("initializer calls=%#v", calls)
	}
}

func TestInitRejectsInvalidAndLegacyDimensionFlags(t *testing.T) {
	called := false
	deps := Dependencies{Stdout: io.Discard, Stderr: io.Discard, Initialize: func(context.Context, string, int) error {
		called = true
		return nil
	}}
	for _, args := range [][]string{
		{"init", "--serving-dim", "255"},
		{"init", "--serving-dim", "768"},
		{"init", "--codec", "int8"},
		{"init", "--target-dim", "512"},
	} {
		if err := Run(context.Background(), args, deps); err == nil {
			t.Fatalf("args=%v unexpectedly succeeded", args)
		}
	}
	if called {
		t.Fatal("initializer called for invalid init flags")
	}
}

func TestInitCreatesStateAtGitRootFromNestedDirectory(t *testing.T) {
	repository := t.TempDir()
	git(t, repository, "init")
	nested := filepath.Join(repository, "nested")
	if err := os.MkdirAll(nested, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir(nested)
	if err := Run(context.Background(), []string{"init", "--serving-dim", "512"}, Dependencies{Stdout: io.Discard, Stderr: io.Discard}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repository, ".cidx", "config.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repository, ".cidx", "db", "index.db")); err != nil {
		t.Fatal(err)
	}
}

func TestUsageAdvertisesServingDimensionAndLocalInit(t *testing.T) {
	var output strings.Builder
	if err := Run(context.Background(), []string{"help"}, Dependencies{Stdout: &output, Stderr: io.Discard}); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if !strings.Contains(text, "--serving-dim") || strings.Contains(text, "--target-dim") || !strings.Contains(text, "init creates local config") || !strings.Contains(text, "cidx version [--json]") {
		t.Fatalf("help=%q", text)
	}
}

func TestVersionJSONIsBuildInfoOnly(t *testing.T) {
	var output strings.Builder
	if err := Run(context.Background(), []string{"version", "--json"}, Dependencies{Stdout: &output, Stderr: io.Discard}); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Version  string `json:"version"`
		Commit   string `json:"commit"`
		TargetOS string `json:"target_os"`
		Runtime  struct {
			FTS5Available       bool `json:"fts5_available"`
			WALAvailable        bool `json:"wal_available"`
			ProductionSchemaMax int  `json:"production_schema_maximum"`
		} `json:"runtime"`
	}
	if err := json.Unmarshal([]byte(output.String()), &got); err != nil {
		t.Fatal(err)
	}
	if got.Version == "" || got.Commit == "" || got.TargetOS == "" || !got.Runtime.FTS5Available || !got.Runtime.WALAvailable || got.Runtime.ProductionSchemaMax < 1 {
		t.Fatalf("incomplete version JSON: %s", output.String())
	}
}

func TestStableCLIJSONUsesSnakeCaseFields(t *testing.T) {
	for _, fixture := range []struct {
		value     any
		required  string
		forbidden string
	}{
		{app.StatusResponse{Desired: config.AppliedProfiles{ActiveGeneration: 1}, ObservedGeneration: 1, Files: 2}, `"desired"`, `"Desired"`},
		{index.Result{ManifestSHA256: "hash", PlannedEmbeddingsPending: 1}, `"planned_embeddings_pending"`, `"PlannedEmbeddingsPending"`},
		{app.PublicEmbeddingPlan{ActiveDistinct: 1, ManifestSHA256: "hash"}, `"voyage_input_count"`, `"paid_input_count"`},
		{app.PublicEmbeddingResult{ActualTokens: 1}, `"actual_tokens"`, `"ActualTokens"`},
	} {
		encoded, err := json.Marshal(fixture.value)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(encoded), fixture.required) || strings.Contains(string(encoded), fixture.forbidden) {
			t.Fatalf("json=%s required=%s forbidden=%s", encoded, fixture.required, fixture.forbidden)
		}
	}
}

func TestServeHandshakeEOFDoesNotCreateLabState(t *testing.T) {
	root := t.TempDir()
	git(t, root, "init")
	dim := 512
	raw := config.RawConfig{Version: 1, Index: config.RawIndex{Languages: []string{"go"}, MaxSourceFileBytes: 4096, TargetSegmentBytes: 2048}, Embedding: config.RawEmbedding{ServingDimensions: &dim, Request: config.RawRequest{MaxInputs: 1, MaxTotalInputBytes: 8192, TimeoutSeconds: 1}}, MCP: config.RawMCP{HardMaxInlineBytes: 1024}}
	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".cidx"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".cidx", "config.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	input, writer := io.Pipe()
	frames := make(chan []byte, 2)
	done := make(chan error, 1)
	go func() {
		done <- Run(context.Background(), []string{"serve", "--root", root}, Dependencies{Stdin: input, Stdout: frameSink{frames: frames}, Stderr: io.Discard})
	}()
	_, err = io.WriteString(writer, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"smoke","version":"1"}}}`+"\n")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case frame := <-frames:
		if !strings.Contains(string(frame), `"protocolVersion":"2025-11-25"`) {
			t.Fatalf("initialize frame=%s", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not complete initialize handshake")
	}
	_, err = io.WriteString(writer, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n"+`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`+"\n")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case frame := <-frames:
		var result struct {
			ID     int `json:"id"`
			Result struct {
				Tools []json.RawMessage `json:"tools"`
			} `json:"result"`
		}
		if err := json.Unmarshal(frame, &result); err != nil {
			t.Fatalf("tools/list frame=%q: %v", frame, err)
		}
		if result.ID != 2 || len(result.Result.Tools) != 4 {
			t.Fatalf("tools/list=%s", frame)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not return tool registry")
	}
	_ = writer.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("serve did not exit at EOF")
	}
	if _, err := os.Stat(filepath.Join(root, ".cidx", "lab")); !os.IsNotExist(err) {
		t.Fatalf("serve created lab state: %v", err)
	}
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, output)
	}
}
