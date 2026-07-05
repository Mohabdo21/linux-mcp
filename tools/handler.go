package tools

import (
	"context"
	"errors"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolOutput is implemented by all output types with error accumulation.
type toolOutput interface {
	AppendError(string)
	ErrorCount() int
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
	LogToolCall(ctx, name, time.Since(start), out.ErrorCount())
	if err != nil {
		out.AppendError(err.Error())
	}

	return nil, out, nil
}
