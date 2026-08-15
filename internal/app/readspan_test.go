package app

import (
	"cidx/internal/chunk"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinesReturnsCompleteInclusiveSourceLines(t *testing.T) {
	data := []byte("one\ntwo\nthree\n")
	got, ok := lines(data, 2, 3)
	if !ok || string(got) != "two\nthree\n" {
		t.Fatalf("got=%q ok=%t", got, ok)
	}
	if _, ok := lines(data, 4, 4); ok {
		t.Fatal("phantom final line accepted")
	}
}

func TestReadSpanHashesAndReturnsWholeRange(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	source := []byte("one\ntwo\nthree\n")
	mustWriteFile(t, filepath.Join(root, "a.go"), string(source))
	runGit(t, root, "add", "a.go")
	service := ReadSpanService{Root: root, Resolved: materializeResolved(t)}
	hash := fmt.Sprintf("%x", sha256.Sum256(source))
	result, err := service.Read(ctx, ReadSpanRequest{Path: "a.go", StartLine: 2, EndLine: 3, ExpectedSHA256: hash})
	if err != nil || string(result.Body) != "two\nthree\n" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	_, err = service.Read(ctx, ReadSpanRequest{Path: "a.go", StartLine: 1, EndLine: 1, ExpectedSHA256: strings.Repeat("a", 64)})
	if value, ok := err.(ReadSpanError); !ok || value.Code != ReadSpanFileStale {
		t.Fatalf("stale=%v", err)
	}
}

func TestReadSpanHasNoHistoricalLineCap(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	var source strings.Builder
	for line := 0; line < 500; line++ {
		source.WriteString("x\n")
	}
	body := []byte(source.String())
	mustWriteFile(t, filepath.Join(root, "a.go"), string(body))
	runGit(t, root, "add", "a.go")
	service := ReadSpanService{Root: root, Resolved: materializeResolved(t)}
	result, err := service.Read(ctx, ReadSpanRequest{Path: "a.go", StartLine: 1, EndLine: 500, ExpectedSHA256: fmt.Sprintf("%x", sha256.Sum256(body))})
	if err != nil || string(result.Body) != string(body) {
		t.Fatalf("result bytes=%d err=%v", len(result.Body), err)
	}
}

func TestReadSpanOversizeReturnsTypedErrorWithoutPartialBody(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	body := []byte(strings.Repeat("x", 1025) + "\n")
	mustWriteFile(t, filepath.Join(root, "a.go"), string(body))
	runGit(t, root, "add", "a.go")
	service := ReadSpanService{Root: root, Resolved: materializeResolved(t)}
	response, err := service.Read(ctx, ReadSpanRequest{Path: "a.go", StartLine: 1, EndLine: 1, ExpectedSHA256: fmt.Sprintf("%x", sha256.Sum256(body))})
	value, ok := err.(ReadSpanError)
	if !ok || value.Code != ReadSpanTooLarge || value.MaxBytes != service.Resolved.MCP.HardMaxInlineBytes {
		t.Fatalf("error=%#v", err)
	}
	if len(response.Body) != 0 {
		t.Fatalf("oversize response included partial body: %q", response.Body)
	}
}

func TestReadSpanRestrictsEligibleUTF8Source(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "config.json"), "{}")
	text := []byte("tracked but not indexed\n")
	mustWriteFile(t, filepath.Join(root, "note.txt"), string(text))
	invalid := []byte("package p\n// \xff\n")
	if err := os.WriteFile(filepath.Join(root, "bad.go"), invalid, 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, root, "add", "note.txt", "bad.go")
	service := ReadSpanService{Root: root, Resolved: materializeResolved(t)}
	for _, fixture := range []struct {
		path string
		body []byte
		want string
	}{{"note.txt", text, ReadSpanFileNotFound}, {"bad.go", invalid, ReadSpanInvalidPath}} {
		_, err := service.Read(ctx, ReadSpanRequest{Path: fixture.path, StartLine: 1, EndLine: 1, ExpectedSHA256: fmt.Sprintf("%x", sha256.Sum256(fixture.body))})
		if value, ok := err.(ReadSpanError); !ok || value.Code != fixture.want {
			t.Fatalf("path=%s err=%v", fixture.path, err)
		}
	}
}

func TestReadRegularNoSymlinkRejectsSymlinkAndEnforcesCap(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularNoSymlink(root, "a.go", 2); err == nil {
		t.Fatal("oversize accepted")
	}
	if err := os.Symlink("a.go", filepath.Join(root, "link.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := readRegularNoSymlink(root, "link.go", 10); err == nil {
		t.Fatal("symlink accepted")
	}
}

func TestReadPathRejectsTraversalAndAbsolute(t *testing.T) {
	for _, path := range []string{"../a.go", "/a.go", "a/../b.go", "a\n.go"} {
		if validReadPath(path) {
			t.Fatalf("accepted %q", path)
		}
	}
	if !validReadPath("internal/a.go") {
		t.Fatal("valid path rejected")
	}
}

func TestReadSpanDigestRequiresLowercaseHex(t *testing.T) {
	if !isDigest(strings.Repeat("a", 64)) {
		t.Fatal("lowercase digest rejected")
	}
	if isDigest(strings.Repeat("A", 64)) {
		t.Fatal("uppercase digest accepted")
	}
}

func TestStatusLanguageEligibilityFollowsResolvedConfig(t *testing.T) {
	if !enabledStatusLanguage("main.go", []chunk.Language{chunk.Go}) {
		t.Fatal("enabled Go file was excluded")
	}
	if enabledStatusLanguage("view.tsx", []chunk.Language{chunk.Go}) || enabledStatusLanguage("view.ts", []chunk.Language{chunk.Go}) {
		t.Fatal("disabled TypeScript language was included")
	}
	if !enabledStatusLanguage("MAIN.GO", []chunk.Language{chunk.Go}) {
		t.Fatal("case-insensitive Phase05 source eligibility was lost")
	}
}

func TestWorktreeDirtyExcludesOnlyRuntimeIndexFiles(t *testing.T) {
	ctx, root := context.Background(), t.TempDir()
	runGit(t, root, "init")
	mustWriteFile(t, filepath.Join(root, ".cidx", "index.db"), "runtime")
	dirty, err := worktreeDirty(ctx, root)
	if err != nil || dirty {
		t.Fatalf("runtime index dirty=%t err=%v", dirty, err)
	}
	mustWriteFile(t, filepath.Join(root, ".cidx", "lab", "note"), "must remain visible")
	dirty, err = worktreeDirty(ctx, root)
	if err != nil || !dirty {
		t.Fatalf("lab state hidden dirty=%t err=%v", dirty, err)
	}
}
