package tools

import (
	"context"
	"errors"
	"fmt"
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
