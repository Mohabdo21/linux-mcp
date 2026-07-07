package tools

import (
	"context"
	"fmt"
	"strings"

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
		"short-iso",
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
	logLines, err := execLines(ctx, "journalctl", args...)
	if err != nil {
		return nil, err
	}
	entries := make([]JournalLogEntry, 0)
	for _, line := range logLines {
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		entries = append(entries, JournalLogEntry{
			Timestamp: parts[0],
			Message:   parts[2],
		})
	}
	return &JournalLogsOutput{Entries: entries}, nil
}

func HandleGetJournalLogs(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetJournalLogsInput,
) (*mcp.CallToolResult, *JournalLogsOutput, error) {
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
