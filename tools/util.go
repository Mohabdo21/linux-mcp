// Package tools provides MCP tool implementations for system inspection,
// including CPU, memory, disk, network, Docker, GPU, and other utilities.
package tools

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
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

func collectOrFallback[T any](
	ctx context.Context,
	name string,
	gather func(context.Context) (*T, error),
	fallback T,
	errs *[]string,
) T {
	out, err := gather(ctx)
	if err != nil {
		appendErr(errs, name, err)
		return fallback
	}
	return *out
}

func clampVal[T ~int](val, min, max T) T {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func clampZero[T ~int](val, def, max T) T {
	if val <= 0 {
		return def
	}
	if val > max {
		return max
	}
	return val
}

func nilToEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
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

func execOutput(
	ctx context.Context,
	binary string,
	args ...string,
) (string, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}

func execCombinedOutput(
	ctx context.Context,
	binary string,
	args ...string,
) (string, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func execLines(
	ctx context.Context,
	binary string,
	args ...string,
) ([]string, error) {
	out, err := execOutput(ctx, binary, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	var lines []string
	for line := range strings.SplitSeq(out, "\n") {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func readSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

const shortIDLen = 12

func shortID(id string) string {
	if len(id) > shortIDLen {
		return id[:shortIDLen]
	}
	return id
}

var sensitiveEnvPatterns = []string{
	"SECRET", "TOKEN", "PASSWORD", "CREDENTIAL",
	"API_KEY", "PRIVATE_KEY", "DATABASE_URL",
}

func isSensitiveEnvVar(name string) bool {
	upper := strings.ToUpper(name)
	for _, pattern := range sensitiveEnvPatterns {
		if strings.Contains(upper, pattern) {
			return true
		}
	}
	return false
}

func requireField(val, name string) error {
	if val == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
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
