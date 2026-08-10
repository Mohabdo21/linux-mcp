package tools

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// timespanUnits maps systemd time suffixes to seconds.
var timespanUnits = map[string]float64{
	"ns":    1e-9,
	"us":    1e-6,
	"ms":    1e-3,
	"s":     1,
	"min":   60,
	"h":     3600,
	"d":     86400,
	"w":     7 * 86400,
	"month": 30 * 86400,
	"y":     365 * 86400,
}

// parseTimespan parses a systemd human-readable timespan such as "6.455s",
// "973ms", "886us", or "1min 9.608s" into seconds.
func parseTimespan(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty timespan")
	}
	total := 0.0
	for token := range strings.FieldsSeq(s) {
		idx := 0
		for idx < len(token) &&
			(token[idx] == '.' || token[idx] >= '0' && token[idx] <= '9') {
			idx++
		}
		if idx == 0 {
			return 0, fmt.Errorf("invalid timespan %q", s)
		}
		num, err := strconv.ParseFloat(token[:idx], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid timespan %q: %w", s, err)
		}
		mult, ok := timespanUnits[token[idx:]]
		if !ok {
			return 0, fmt.Errorf("invalid timespan %q: unknown unit", s)
		}
		total += num * mult
	}
	return total, nil
}

type BootBlameEntry struct {
	Unit        string  `json:"unit"`
	Time        string  `json:"time"`
	TimeSeconds float64 `json:"time_seconds"`
}

type BootBlameOutput struct {
	Entries []BootBlameEntry `json:"entries"`
	OutputErrors
}

// parseBootBlame parses "systemd-analyze blame" output: one "<time> <unit>"
// per line, ordered by descending init time.
func parseBootBlame(out string) []BootBlameEntry {
	var entries []BootBlameEntry
	for line := range strings.SplitSeq(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		unit := fields[len(fields)-1]
		timeStr := strings.Join(fields[:len(fields)-1], " ")
		secs, err := parseTimespan(timeStr)
		if err != nil {
			continue
		}
		entries = append(entries, BootBlameEntry{
			Unit:        unit,
			Time:        timeStr,
			TimeSeconds: secs,
		})
	}
	return entries
}

func GatherBootBlame(ctx context.Context) (*BootBlameOutput, error) {
	out, err := execOutput(ctx, "systemd-analyze", "blame")
	if err != nil {
		return nil, err
	}
	return &BootBlameOutput{Entries: parseBootBlame(out)}, nil
}

type BootChainLink struct {
	Unit          string  `json:"unit"`
	Depth         int     `json:"depth"`
	ActiveTime    string  `json:"active_time,omitempty"`
	ActiveSeconds float64 `json:"active_seconds,omitempty"`
	StartTime     string  `json:"start_time,omitempty"`
	StartSeconds  float64 `json:"start_seconds,omitempty"`
}

type BootCriticalChainOutput struct {
	Target string          `json:"target"`
	Chain  []BootChainLink `json:"chain"`
	OutputErrors
}

type GetBootCriticalChainInput struct {
	Unit string `json:"unit,omitempty" jsonschema:"optional unit name to start the chain from (e.g. 'graphical.target')"`
}

var bootChainLineRe = regexp.MustCompile(`^(\S+)(?: @(\S+))?(?: \+(\S+))?$`)

// parseBootChainLine parses one "systemd-analyze critical-chain" line. The
// tree prefix sets the depth (two characters per level); after it comes the
// unit name and optionally "@<active-time>" and "+<start-duration>".
func parseBootChainLine(line string) (BootChainLink, bool) {
	depth := 0
	content := line
	if idx := strings.LastIndex(line, "─"); idx >= 0 {
		content = line[idx+len("─"):]
		depth = (idx - 1) / 2
	}
	m := bootChainLineRe.FindStringSubmatch(content)
	if m == nil {
		return BootChainLink{}, false
	}
	link := BootChainLink{Unit: m[1], Depth: depth}
	if m[2] != "" {
		link.ActiveTime = m[2]
		link.ActiveSeconds, _ = parseTimespan(m[2])
	}
	if m[3] != "" {
		link.StartTime = m[3]
		link.StartSeconds, _ = parseTimespan(m[3])
	}
	return link, true
}

// parseBootCriticalChain parses "systemd-analyze critical-chain" output,
// skipping the two explanatory header lines.
func parseBootCriticalChain(out string) []BootChainLink {
	var chain []BootChainLink
	for line := range strings.SplitSeq(out, "\n") {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "The time when unit became active") ||
			strings.HasPrefix(line, "The time the unit took to start") {
			continue
		}
		if link, ok := parseBootChainLine(line); ok {
			chain = append(chain, link)
		}
	}
	return chain
}

func GatherBootCriticalChain(
	ctx context.Context, unit string,
) (*BootCriticalChainOutput, error) {
	args := []string{"critical-chain"}
	if unit != "" {
		args = append(args, unit)
	}
	out, err := execOutput(ctx, "systemd-analyze", args...)
	if err != nil {
		return nil, err
	}
	chain := parseBootCriticalChain(out)
	result := &BootCriticalChainOutput{Chain: chain}
	if len(chain) > 0 {
		result.Target = chain[0].Unit
	}
	return result, nil
}

type BootPhase struct {
	Name    string  `json:"name"`
	Time    string  `json:"time"`
	Seconds float64 `json:"seconds"`
}

type BootTimeOutput struct {
	Phases               []BootPhase `json:"phases"`
	Total                string      `json:"total,omitempty"`
	TotalSeconds         float64     `json:"total_seconds,omitempty"`
	Target               string      `json:"target,omitempty"`
	TargetReachedTime    string      `json:"target_reached_time,omitempty"`
	TargetReachedSeconds float64     `json:"target_reached_seconds,omitempty"`
	OutputErrors
}

// parseBootPhases parses the phase portion of the "Startup finished in ..."
// line, e.g. "12.067s (firmware) + 7.428s (loader)".
func parseBootPhases(s string) []BootPhase {
	var phases []BootPhase
	for phase := range strings.SplitSeq(s, " + ") {
		timeStr := phase
		name := ""
		if idx := strings.LastIndex(phase, " ("); idx >= 0 &&
			strings.HasSuffix(phase, ")") {
			name = phase[idx+2 : len(phase)-1]
			timeStr = phase[:idx]
		}
		secs, err := parseTimespan(timeStr)
		if err != nil {
			continue
		}
		phases = append(phases, BootPhase{
			Name:    name,
			Time:    timeStr,
			Seconds: secs,
		})
	}
	return phases
}

// parseBootTime parses "systemd-analyze time" output: the per-phase breakdown
// line and the reached-target line.
func parseBootTime(out string) *BootTimeOutput {
	result := &BootTimeOutput{}
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Startup finished in "):
			rest := strings.TrimPrefix(line, "Startup finished in ")
			if before, after, ok := strings.Cut(rest, " = "); ok {
				rest = before
				result.Total = after
				result.TotalSeconds, _ = parseTimespan(after)
			}
			result.Phases = parseBootPhases(rest)
		case strings.Contains(line, " reached after ") &&
			strings.Contains(line, " in userspace"):
			rest := strings.TrimSuffix(line, ".")
			rest = strings.TrimSuffix(rest, " in userspace")
			if before, after, ok := strings.Cut(rest, " reached after "); ok {
				result.Target = before
				ts := strings.TrimSpace(after)
				result.TargetReachedTime = ts
				result.TargetReachedSeconds, _ = parseTimespan(ts)
			}
		}
	}
	return result
}

func GatherBootTime(ctx context.Context) (*BootTimeOutput, error) {
	out, err := execOutput(ctx, "systemd-analyze", "time")
	if err != nil {
		return nil, err
	}
	return parseBootTime(out), nil
}

func HandleGetBootBlame(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *BootBlameOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetBootBlame,
		0,
		GatherBootBlame,
	)
}

func HandleGetBootCriticalChain(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetBootCriticalChainInput,
) (*mcp.CallToolResult, *BootCriticalChainOutput, error) {
	if input.Unit != "" && !validServiceName.MatchString(input.Unit) {
		return nil, nil, fmt.Errorf("invalid unit name: %q", input.Unit)
	}
	return handleToolCall(
		ctx,
		config.ToolNameGetBootCriticalChain,
		0,
		func(ctx context.Context) (*BootCriticalChainOutput, error) {
			return GatherBootCriticalChain(ctx, input.Unit)
		},
	)
}

func HandleGetBootTime(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *BootTimeOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetBootTime,
		0,
		GatherBootTime,
	)
}
