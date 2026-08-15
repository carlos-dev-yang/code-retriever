package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"cidx/internal/config"
	"cidx/internal/ignore"
)

type ReadSpanError struct {
	Code     string
	MaxBytes int
}

func (value ReadSpanError) Error() string { return value.Code }

const (
	ReadSpanFileNotFound = "FILE_NOT_FOUND"
	ReadSpanFileStale    = "FILE_STALE"
	ReadSpanTooLarge     = "SPAN_TOO_LARGE"
	ReadSpanInvalidPath  = "INVALID_PATH"
	ReadSpanInvalidRange = "INVALID_RANGE"
)

type ReadSpanRequest struct {
	Path               string
	StartLine, EndLine int
	ExpectedSHA256     string
}
type ReadSpanResponse struct {
	Path          string `json:"path"`
	StartLine     int    `json:"start_line"`
	EndLine       int    `json:"end_line"`
	Body          []byte `json:"body"`
	IndexedSHA256 string `json:"indexed_sha256"`
}
type ReadSpanService struct {
	Root     string
	Resolved config.ResolvedConfig
}

func (service ReadSpanService) Read(ctx context.Context, request ReadSpanRequest) (ReadSpanResponse, error) {
	if err := ctx.Err(); err != nil {
		return ReadSpanResponse{}, err
	}
	if !validReadPath(request.Path) || !isDigest(request.ExpectedSHA256) {
		return ReadSpanResponse{}, ReadSpanError{Code: ReadSpanInvalidPath}
	}
	if request.StartLine <= 0 || request.EndLine < request.StartLine {
		return ReadSpanResponse{}, ReadSpanError{Code: ReadSpanInvalidRange}
	}
	if !enabledStatusLanguage(request.Path, service.Resolved.Index.Languages) {
		return ReadSpanResponse{}, ReadSpanError{Code: ReadSpanFileNotFound}
	}
	allowed, err := indexedCandidate(ctx, service.Root, request.Path, service.Resolved.Index.MaxSourceFileBytes)
	if err != nil {
		return ReadSpanResponse{}, err
	}
	if !allowed {
		return ReadSpanResponse{}, ReadSpanError{Code: ReadSpanFileNotFound}
	}
	data, err := readRegularNoSymlink(service.Root, request.Path, service.Resolved.Index.MaxSourceFileBytes)
	if errors.Is(err, os.ErrNotExist) {
		return ReadSpanResponse{}, ReadSpanError{Code: ReadSpanFileNotFound}
	}
	if err != nil {
		return ReadSpanResponse{}, ReadSpanError{Code: ReadSpanInvalidPath}
	}
	if !utf8.Valid(data) {
		return ReadSpanResponse{}, ReadSpanError{Code: ReadSpanInvalidPath}
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	if hash != request.ExpectedSHA256 {
		return ReadSpanResponse{}, ReadSpanError{Code: ReadSpanFileStale}
	}
	body, ok := lines(data, request.StartLine, request.EndLine)
	if !ok {
		return ReadSpanResponse{}, ReadSpanError{Code: ReadSpanInvalidRange}
	}
	if len(body) > service.Resolved.MCP.HardMaxInlineBytes {
		return ReadSpanResponse{}, ReadSpanError{Code: ReadSpanTooLarge, MaxBytes: service.Resolved.MCP.HardMaxInlineBytes}
	}
	return ReadSpanResponse{Path: request.Path, StartLine: request.StartLine, EndLine: request.EndLine, Body: body, IndexedSHA256: hash}, nil
}

func validReadPath(path string) bool {
	return path != "" && filepath.ToSlash(filepath.Clean(path)) == path && !filepath.IsAbs(path) && path != "." && path != ".." && !strings.HasPrefix(path, "../") && !strings.ContainsAny(path, "\x00\r\n")
}
func isDigest(value string) bool {
	_, err := hex.DecodeString(value)
	return err == nil && len(value) == 64 && strings.ToLower(value) == value
}
func indexedCandidate(ctx context.Context, root, wanted string, max int) (bool, error) {
	candidates, err := ignore.Enumerate(ctx, root, int64(max))
	if err != nil {
		return false, err
	}
	for _, candidate := range candidates {
		if candidate.Path == wanted {
			return candidate.Exclusion == "", nil
		}
	}
	return false, nil
}
func readRegularNoSymlink(root, relative string, maxBytes int) ([]byte, error) {
	rootHandle, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer rootHandle.Close()
	parts := strings.Split(filepath.ToSlash(relative), "/")
	for index := range parts {
		info, err := rootHandle.Lstat(strings.Join(parts[:index+1], "/"))
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink path")
		}
	}
	file, err := rootHandle.Open(strings.Join(parts, "/"))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not regular")
	}
	if info.Size() > int64(maxBytes) {
		return nil, fmt.Errorf("source file exceeds configured limit")
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBytes {
		return nil, fmt.Errorf("source file exceeds configured limit")
	}
	return data, nil
}
func lines(data []byte, start, end int) ([]byte, bool) {
	line, begin := 1, 0
	for offset := 0; offset < len(data); {
		newline := offset
		for newline < len(data) && data[newline] != '\n' {
			newline++
		}
		lineEnd := newline
		if newline < len(data) {
			lineEnd++
		}
		if line == start {
			begin = offset
		}
		if line == end {
			return append([]byte(nil), data[begin:lineEnd]...), true
		}
		line++
		offset = lineEnd
	}
	return nil, false
}
