package mcp

import (
	"context"
	"errors"

	"cidx/internal/app"
)

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (value *Error) Error() string { return value.Message }

const (
	parseError     = -32700
	invalidRequest = -32600
	methodNotFound = -32601
	invalidParams  = -32602
	internalError  = -32603
)

func applicationError(err error) *Error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return &Error{Code: -32001, Message: "REQUEST_CANCELLED"}
	}
	var span app.ReadSpanError
	if errors.As(err, &span) {
		data := map[string]any{"code": span.Code}
		if span.MaxBytes > 0 {
			data["max_bytes"] = span.MaxBytes
		}
		return &Error{Code: -32010, Message: span.Code, Data: data}
	}
	return &Error{Code: -32000, Message: "APPLICATION_ERROR"}
}
