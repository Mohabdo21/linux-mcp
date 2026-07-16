package tools

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetFailedLoginsInput struct {
	Lines int `json:"lines,omitempty" jsonschema:"number of recent entries (default: 20)"`
}

type FailedLoginEntry struct {
	Username  string `json:"username"`
	Terminal  string `json:"terminal"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
}

type FailedLoginsSummary struct {
	TotalAttempts   int `json:"total_attempts"`
	UniqueUsernames int `json:"unique_usernames"`
	UniqueSources   int `json:"unique_sources"`
}

type FailedLoginsOutput struct {
	Entries []FailedLoginEntry  `json:"entries"`
	Summary FailedLoginsSummary `json:"summary"`
	OutputErrors
}

func isBootEntry(fields []string) bool {
	if len(fields) < 2 {
		return false
	}
	return fields[0] == "reboot" || fields[1] == "Boot"
}

func ParseLastbOutput(output string) []FailedLoginEntry {
	entries := make([]FailedLoginEntry, 0)
	for line := range strings.SplitSeq(
		strings.TrimSpace(output), "\n",
	) {
		if line == "" || strings.HasPrefix(line, "btmp begins") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if isBootEntry(fields) {
			continue
		}
		entries = append(entries, FailedLoginEntry{
			Username:  fields[0],
			Terminal:  fields[1],
			Source:    fields[2],
			Timestamp: strings.Join(fields[3:], " "),
		})
	}
	return entries
}

func ParseJournalctlFailedLogins(output string) []FailedLoginEntry {
	entries := make([]FailedLoginEntry, 0)
	for line := range strings.SplitSeq(
		strings.TrimSpace(output), "\n",
	) {
		if line == "" || strings.HasPrefix(line, "-- ") {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		entries = append(entries, FailedLoginEntry{
			Timestamp: parts[0],
			Terminal:  parts[1],
			Source:    "",
			Username:  parts[2],
		})
	}
	return entries
}

func GatherFailedLoginsJournalctl(
	ctx context.Context, lines int,
) (*FailedLoginsOutput, error) {
	out, err := execOutput(ctx, "journalctl",
		"-u", "sshd", "-u", "systemd-logind",
		"--grep", "Failed password|authentication failure|Failed login",
		"--no-pager", "-o", "short-iso",
		"-n", fmt.Sprintf("%d", lines),
	)
	entries := ParseJournalctlFailedLogins(out)
	if err != nil && errors.Is(err, exec.ErrNotFound) {
		return nil, err
	}
	return &FailedLoginsOutput{Entries: entries}, nil
}

func computeFailedLoginsSummary(
	entries []FailedLoginEntry,
) FailedLoginsSummary {
	userSet := make(map[string]struct{})
	sourceSet := make(map[string]struct{})
	for _, e := range entries {
		userSet[e.Username] = struct{}{}
		if e.Source != "" {
			sourceSet[e.Source] = struct{}{}
		}
	}
	return FailedLoginsSummary{
		TotalAttempts:   len(entries),
		UniqueUsernames: len(userSet),
		UniqueSources:   len(sourceSet),
	}
}

func GatherFailedLogins(
	ctx context.Context, lines int,
) (*FailedLoginsOutput, error) {
	out, lastbErr := execOutput(ctx, "lastb", "-n", fmt.Sprintf("%d", lines))
	if lastbErr == nil {
		entries := ParseLastbOutput(out)
		return &FailedLoginsOutput{
			Entries: entries,
			Summary: computeFailedLoginsSummary(entries),
		}, nil
	}
	if !errors.Is(lastbErr, exec.ErrNotFound) {
		if entries := ParseLastbOutput(out); len(entries) > 0 {
			return &FailedLoginsOutput{
				Entries: entries,
				Summary: computeFailedLoginsSummary(entries),
			}, nil
		}
	}
	jOut, jErr := GatherFailedLoginsJournalctl(ctx, lines)
	if jErr != nil {
		return jOut, errors.Join(lastbErr, jErr)
	}
	jOut.Summary = computeFailedLoginsSummary(jOut.Entries)
	if !errors.Is(lastbErr, exec.ErrNotFound) {
		return jOut, lastbErr
	}
	return jOut, nil
}

type LoggedInUser struct {
	Username  string `json:"username"`
	Terminal  string `json:"terminal"`
	From      string `json:"from"`
	LoginTime string `json:"login_time"`
}

type LoggedInUsersOutput struct {
	Users []LoggedInUser `json:"users"`
	OutputErrors
}

func GatherLoggedInUsers(ctx context.Context) (*LoggedInUsersOutput, error) {
	lines, err := execLines(ctx, "who", "-u")
	if err != nil {
		return nil, err
	}
	users := make([]LoggedInUser, 0)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		from := ""
		if len(fields) > 5 {
			from = fields[5]
		}
		users = append(users, LoggedInUser{
			Username:  fields[0],
			Terminal:  fields[1],
			LoginTime: strings.Join(fields[2:4], " "),
			From:      from,
		})
	}
	return &LoggedInUsersOutput{Users: users}, nil
}

func HandleGetLoggedInUsers(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *LoggedInUsersOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetLoggedInUsers,
		0,
		GatherLoggedInUsers,
	)
}

func HandleGetFailedLogins(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetFailedLoginsInput,
) (*mcp.CallToolResult, *FailedLoginsOutput, error) {
	lines := input.Lines
	if lines <= 0 {
		lines = 20
	}
	return handleToolCall(
		ctx,
		config.ToolNameGetFailedLogins,
		0,
		func(ctx context.Context) (*FailedLoginsOutput, error) {
			return GatherFailedLogins(ctx, lines)
		},
	)
}

type GetAuditLogsInput struct {
	Lines  int    `json:"lines,omitempty"  jsonschema:"number of recent entries (default: 50)"`
	Source string `json:"source,omitempty" jsonschema:"audit source: 'journalctl', 'audit.log', or 'auto' (default: auto)"`
}

type AuditLogEntry struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Message   string `json:"message"`
}

type AuditLogsOutput struct {
	Entries []AuditLogEntry `json:"entries"`
	OutputErrors
}

func GatherAuditLogs(
	ctx context.Context,
	lines int,
	source string,
) (*AuditLogsOutput, error) {
	if lines <= 0 {
		lines = 50
	}

	if source == "" {
		source = "auto"
	}

	switch source {
	case "journalctl":
		return gatherFromJournalctl(ctx, lines)
	case "audit.log":
		return gatherFromAuditLog(lines)
	default:
		out, err := gatherFromJournalctl(ctx, lines)
		if err == nil && len(out.Entries) > 0 {
			return out, nil
		}
		return gatherFromAuditLog(lines)
	}
}

func gatherFromJournalctl(
	ctx context.Context,
	lines int,
) (*AuditLogsOutput, error) {
	cmd := exec.CommandContext(
		ctx,
		"journalctl",
		"-k",
		"--no-pager",
		"-n",
		fmt.Sprintf("%d", lines),
		"--output=short-iso",
	)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("journalctl failed: %w", err)
	}

	out := &AuditLogsOutput{Entries: make([]AuditLogEntry, 0)}
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		entry := parseJournalctlLine(line)
		out.Entries = append(out.Entries, entry)
	}
	return out, nil
}

func parseJournalctlLine(line string) AuditLogEntry {
	// Format: 2024-01-15T10:30:00+0000 hostname kernel: [  123.456] message
	before, after, ok := strings.Cut(line, "kernel:")
	if !ok {
		return AuditLogEntry{Timestamp: line, Type: "unknown", Message: line}
	}
	parts := strings.SplitN(after, "]", 2)
	msg := strings.TrimSpace(parts[len(parts)-1])
	ts := strings.TrimSpace(before)
	return AuditLogEntry{Timestamp: ts, Type: "kernel", Message: msg}
}

func gatherFromAuditLog(lines int) (*AuditLogsOutput, error) {
	f, err := os.Open("/var/log/audit/audit.log")
	if err != nil {
		return nil, fmt.Errorf("cannot open audit log: %w", err)
	}
	defer func() { _ = f.Close() }()

	out := &AuditLogsOutput{Entries: make([]AuditLogEntry, 0)}
	var allLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}

	start := max(len(allLines)-lines, 0)
	for _, line := range allLines[start:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		entry := parseAuditdLine(line)
		out.Entries = append(out.Entries, entry)
	}
	return out, nil
}

func parseAuditdLine(line string) AuditLogEntry {
	// Format: type=AVC msg=audit(1700000000.123:456): avc: denied { read }
	parenIdx := strings.Index(line, "): ")
	if parenIdx == -1 {
		return AuditLogEntry{Type: "unknown", Message: line}
	}

	header := line[:parenIdx+1]
	msg := line[parenIdx+3:]

	// Extract type from header: type=AVC msg=audit(...)
	typeStart := strings.Index(header, "type=")
	if typeStart == -1 {
		return AuditLogEntry{Type: "unknown", Message: line}
	}
	typeEnd := strings.Index(header[typeStart:], " ")
	msgType := header[typeStart+5 : typeEnd]

	// Extract timestamp from msg=audit(...)
	tsStart := strings.Index(header, "audit(")
	tsEnd := strings.LastIndex(header, ")")
	ts := ""
	if tsStart != -1 && tsEnd > tsStart {
		ts = header[tsStart+6 : tsEnd]
	}

	return AuditLogEntry{
		Timestamp: ts,
		Type:      msgType,
		Message:   msg,
	}
}

func HandleGetAuditLogs(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetAuditLogsInput,
) (*mcp.CallToolResult, *AuditLogsOutput, error) {
	lines := input.Lines
	if lines <= 0 {
		lines = 50
	}
	return handleToolCall(
		ctx,
		config.ToolNameGetAuditLogs,
		0,
		func(ctx context.Context) (*AuditLogsOutput, error) {
			return GatherAuditLogs(ctx, lines, input.Source)
		},
	)
}
