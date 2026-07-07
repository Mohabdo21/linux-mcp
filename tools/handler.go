package tools

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolOutput is implemented by all output types with error accumulation.
type toolOutput interface {
	AppendError(string)
	ErrorCount() int
}

// isOutNil returns true when v is nil (typed or untyped).
func isOutNil(v any) bool {
	if v == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map,
		reflect.Slice, reflect.Chan, reflect.Func:
		return rv.IsNil()
	}
	return false
}

// handleToolCall wraps the standard handler boilerplate for a gather function.
func handleToolCall[Out toolOutput](
	ctx context.Context,
	name string,
	fallback time.Duration,
	gather func(context.Context) (Out, error),
) (*mcp.CallToolResult, Out, error) {
	if config.IsDisabled(name) {
		var zero Out
		return nil, zero, errors.New("tool disabled by configuration")
	}

	ctx, cancel := WithToolTimeout(ctx, name, fallback)
	defer cancel()

	start := time.Now()
	out, err := gather(ctx)
	if !isOutNil(out) {
		LogToolCall(ctx, name, time.Since(start), out.ErrorCount())
		if err != nil {
			out.AppendError(err.Error())
		}
		return nil, out, nil
	}

	LogToolCall(ctx, name, time.Since(start), 0)
	if err != nil {
		slog.LogAttrs(ctx, slog.LevelError, "tool returned nil output",
			slog.String("tool", name), slog.Any("error", err))
		var zero Out
		return nil, zero, err
	}
	var zero Out
	return nil, zero, nil
}
