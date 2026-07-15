package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetJournalLogsInput struct {
	Unit     string `json:"unit,omitempty"     jsonschema:"optional systemd unit name (e.g. 'nginx.service')"`
	Priority string `json:"priority,omitempty" jsonschema:"optional log priority: emerg,alert,crit,err,warning,notice,info,debug"`
	Since    string `json:"since,omitempty"    jsonschema:"optional start time (e.g. '1 hour ago', '2024-07-03')"`
	Until    string `json:"until,omitempty"    jsonschema:"optional end time"`
	Lines    int    `json:"lines,omitempty"    jsonschema:"number of recent lines (default: 50)"`
	User     bool   `json:"user,omitempty"     jsonschema:"query user-level journal (default: false)"`
}

type JournalLogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Priority  string `json:"priority,omitempty"`
	Unit      string `json:"unit,omitempty"`
	PID       int    `json:"pid,omitempty"`
}

var journalPriorityNames = map[int64]string{
	0: "emerg",
	1: "alert",
	2: "crit",
	3: "err",
	4: "warning",
	5: "notice",
	6: "info",
	7: "debug",
}

type JournalLogsOutput struct {
	Entries []JournalLogEntry `json:"entries"`
	OutputErrors
}

func GatherJournalLogs(
	ctx context.Context, unit, priority, since, until string,
	lines int, user bool,
) (*JournalLogsOutput, error) {
	args := []string{
		"--no-pager",
		"-n",
		fmt.Sprintf("%d", lines),
		"-o",
		"json",
	}
	if user {
		args = append(args, "--user")
	}
	if unit != "" {
		args = append(args, "-u", unit)
	}
	if priority != "" {
		args = append(args, "-p", priority)
	}
	if since != "" {
		args = append(args, "--since", since)
	}
	if until != "" {
		args = append(args, "--until", until)
	}
	out, err := execOutput(ctx, "journalctl", args...)
	if err != nil {
		return nil, err
	}
	entries := make([]JournalLogEntry, 0)
	for line := range strings.SplitSeq(out, "\n") {
		if line == "" {
			continue
		}
		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}
		entry := JournalLogEntry{}

		if msg, ok := raw["MESSAGE"]; ok {
			entry.Message, _ = msg.(string)
		}

		if ts, ok := raw["__REALTIME_TIMESTAMP"]; ok {
			if tsStr, ok := ts.(string); ok {
				if micro, err := strconv.ParseInt(tsStr, 10, 64); err == nil {
					entry.Timestamp = time.UnixMicro(micro).Format(time.RFC3339)
				}
			}
		}

		if prio, ok := raw["PRIORITY"]; ok {
			if prioStr, ok := prio.(string); ok {
				if p, err := strconv.ParseInt(prioStr, 10, 64); err == nil {
					if name, ok := journalPriorityNames[p]; ok {
						entry.Priority = name
					}
				}
			}
		}

		if unit, ok := raw["_SYSTEMD_UNIT"]; ok {
			entry.Unit, _ = unit.(string)
		} else if unit, ok := raw["UNIT"]; ok {
			entry.Unit, _ = unit.(string)
		}

		if pid, ok := raw["_PID"]; ok {
			if pidStr, ok := pid.(string); ok {
				if p, err := strconv.Atoi(pidStr); err == nil {
					entry.PID = p
				}
			}
		}

		entries = append(entries, entry)
	}
	return &JournalLogsOutput{Entries: entries}, nil
}

func isValidUnitName(unit string) bool {
	if unit == "" {
		return true
	}
	for _, c := range unit {
		if !strings.ContainsRune(
			"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_.@-",
			c,
		) {
			return false
		}
	}
	return true
}

func HandleGetJournalLogs(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetJournalLogsInput,
) (*mcp.CallToolResult, *JournalLogsOutput, error) {
	if !isValidUnitName(input.Unit) {
		return nil, nil, fmt.Errorf("invalid unit name: %q", input.Unit)
	}
	lines := input.Lines
	if lines <= 0 {
		lines = 50
	}
	return handleToolCall(
		ctx,
		config.ToolNameGetJournalLogs,
		0,
		func(ctx context.Context) (*JournalLogsOutput, error) {
			return GatherJournalLogs(ctx, input.Unit, input.Priority,
				input.Since, input.Until,
				lines, input.User)
		},
	)
}
