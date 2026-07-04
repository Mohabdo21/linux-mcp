package tools

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
)

func HumanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf(
		"%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func SplitHostPort(s string) (string, string, bool) {
	idx := strings.LastIndex(s, ":")
	if idx >= 0 {
		return s[:idx], s[idx+1:], true
	}
	return s, "", false
}

func ParseProcessField(s string) string {
	if trimmed, ok := strings.CutPrefix(s, "users:(("); ok {
		if start := strings.Index(trimmed, "\""); start >= 0 {
			trimmed = trimmed[start+1:]
			if before, _, ok := strings.Cut(trimmed, "\""); ok {
				return before
			}
		}
	}
	return s
}

// ErrList collects errors during graceful degradation.
type ErrList []string

func (e *ErrList) Add(context string, err error) {
	if err != nil {
		*e = append(*e, context+": "+err.Error())
	}
}

func (e ErrList) Err() error {
	if len(e) == 0 {
		return nil
	}
	return errors.New(strings.Join(e, "; "))
}

func WithToolTimeout(
	ctx context.Context,
	name string,
	fallback time.Duration,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		ctx,
		config.ToolTimeout(name, fallback),
	)
}

func LogToolCall(
	ctx context.Context,
	tool string,
	dur time.Duration,
	errs int,
) {
	slog.LogAttrs(ctx, slog.LevelInfo, "tool call",
		slog.String("tool", tool),
		slog.Duration("duration", dur),
		slog.Int("errors", errs),
	)
}
