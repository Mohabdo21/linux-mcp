package tools

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
)

// NoArgs is used for tools that accept no parameters.
type NoArgs struct{}

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

func appendErr(errors *[]string, context string, err error) {
	if err != nil {
		*errors = append(*errors, context+": "+err.Error())
	}
}

func joinErrs(errs []string) error {
	if len(errs) == 0 {
		return nil
	}
	return errors.New(strings.Join(errs, "; "))
}

// OutputErrors is embedded in output structs for error accumulation.
type OutputErrors struct {
	Errors []string `json:"errors,omitempty"`
}

func (o *OutputErrors) AppendError(s string) {
	o.Errors = append(o.Errors, s)
}

func (o OutputErrors) ErrorCount() int {
	return len(o.Errors)
}

func (o *OutputErrors) Add(context string, err error) {
	appendErr(&o.Errors, context, err)
}

func (o OutputErrors) Err() error {
	return joinErrs(o.Errors)
}

// ErrList collects errors during graceful degradation.
// Kept for internal use in Gather functions that need local error accumulation.
type ErrList []string

func (e *ErrList) Add(context string, err error) {
	appendErr((*[]string)(e), context, err)
}

func (e ErrList) Err() error {
	return joinErrs(e)
}

func (e *ErrList) AppendError(s string) {
	*e = append(*e, s)
}

func (e ErrList) ErrorCount() int {
	return len(e)
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

func readSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
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
