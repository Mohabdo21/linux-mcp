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
}

func ParseLastbOutput(output string) []FailedLoginEntry {
	var entries []FailedLoginEntry
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
	var entries []FailedLoginEntry
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
	lines int,
) (FailedLoginsOutput, error) {
	out, err := exec.Command(
		"journalctl", "-u", "sshd", "-u", "systemd-logind",
		"--grep", "Failed password|authentication failure|Failed login",
		"--no-pager", "-o", "short-iso",
		"-n", fmt.Sprintf("%d", lines),
	).Output()
	entries := ParseJournalctlFailedLogins(string(out))
	if err != nil && errors.Is(err, exec.ErrNotFound) {
		return FailedLoginsOutput{}, err
	}
	return FailedLoginsOutput{Entries: entries}, nil
}

func GatherFailedLogins(lines int) (FailedLoginsOutput, error) {
	if lines <= 0 {
		lines = 20
	}
	out, err := exec.Command(
		"lastb", "-n", fmt.Sprintf("%d", lines),
	).Output()
	if err == nil {
		return FailedLoginsOutput{
			Entries: ParseLastbOutput(string(out)),
		}, nil
	}
	if !errors.Is(err, exec.ErrNotFound) {
		if entries := ParseLastbOutput(string(out)); len(entries) > 0 {
			return FailedLoginsOutput{Entries: entries}, nil
		}
	}
	return GatherFailedLoginsJournalctl(lines)
}

type GetLoggedInUsersInput struct{}

type LoggedInUser struct {
	Username  string `json:"username"`
	Terminal  string `json:"terminal"`
	From      string `json:"from"`
	LoginTime string `json:"login_time"`
}

type LoggedInUsersOutput struct {
	Users []LoggedInUser `json:"users"`
}

func GatherLoggedInUsers() (LoggedInUsersOutput, error) {
	out, err := exec.Command("who", "-u").Output()
	if err != nil {
		return LoggedInUsersOutput{}, err
	}
	var users []LoggedInUser
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
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
	return LoggedInUsersOutput{Users: users}, nil
}

func HandleGetLoggedInUsers(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetLoggedInUsersInput,
) (*mcp.CallToolResult, LoggedInUsersOutput, error) {
	out, err := GatherLoggedInUsers()
	if err != nil {
		return nil, LoggedInUsersOutput{}, err
	}
	return nil, out, nil
}

func HandleGetFailedLogins(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetFailedLoginsInput,
) (*mcp.CallToolResult, FailedLoginsOutput, error) {
	out, err := GatherFailedLogins(input.Lines)
	if err != nil {
		return nil, FailedLoginsOutput{}, err
	}
	return nil, out, nil
}
