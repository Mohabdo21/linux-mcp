package tools

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

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

type FailedLoginsOutput struct {
	Entries []FailedLoginEntry `json:"entries"`
	OutputErrors
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
		if line == "" {
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

func GatherFailedLogins(
	ctx context.Context, lines int,
) (*FailedLoginsOutput, error) {
	if lines <= 0 {
		lines = 20
	}
	out, lastbErr := execOutput(ctx, "lastb", "-n", fmt.Sprintf("%d", lines))
	if lastbErr == nil {
		return &FailedLoginsOutput{
			Entries: ParseLastbOutput(out),
		}, nil
	}
	if !errors.Is(lastbErr, exec.ErrNotFound) {
		if entries := ParseLastbOutput(out); len(entries) > 0 {
			return &FailedLoginsOutput{Entries: entries}, nil
		}
	}
	jOut, jErr := GatherFailedLoginsJournalctl(ctx, lines)
	if jErr != nil {
		return jOut, errors.Join(lastbErr, jErr)
	}
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
		"get_logged_in_users",
		0,
		GatherLoggedInUsers,
	)
}

func HandleGetFailedLogins(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetFailedLoginsInput,
) (*mcp.CallToolResult, *FailedLoginsOutput, error) {
	return handleToolCall(
		ctx,
		"get_failed_logins",
		0,
		func(ctx context.Context) (*FailedLoginsOutput, error) {
			return GatherFailedLogins(ctx, input.Lines)
		},
	)
}
