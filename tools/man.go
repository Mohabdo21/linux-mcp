package tools

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetManPageInput struct {
	Command           string `json:"command"                       jsonschema:"command name to get the man page for"`
	MaxLines          int    `json:"max_lines,omitempty"           jsonschema:"maximum number of lines to return (default: 0 = no limit, max: 10000)"`
	CleanSpecialChars bool   `json:"clean_special_chars,omitempty" jsonschema:"clean backspace formatting characters (default: true)"`
}

type ManPageOutput struct {
	Command   string   `json:"command"`
	Content   string   `json:"content"`
	Truncated bool     `json:"truncated,omitempty"`
	Errors    []string `json:"errors,omitempty"`
}

func cleanManOutput(raw string) string {
	var buf bytes.Buffer
	b := []byte(raw)
	for i := range b {
		if b[i] == '\b' && i > 0 {
			// remove the char before the backspace
			buf.Truncate(buf.Len() - 1)
			continue
		}
		buf.WriteByte(b[i])
	}
	return buf.String()
}

func GatherManPage(
	ctx context.Context,
	command string,
	maxLines int,
	cleanSpecialChars bool,
) (ManPageOutput, error) {
	if _, err := exec.LookPath("man"); err != nil {
		return ManPageOutput{}, errors.New("man command not found")
	}
	cmd := exec.CommandContext(ctx, "man", "-P", "cat", command)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			msg := strings.TrimSpace(string(exitErr.Stderr))
			if msg == "" {
				msg = string(out)
			}
			if strings.Contains(msg, "No manual entry") ||
				strings.Contains(string(out), "No manual entry") {
				return ManPageOutput{},
					errors.New("No manual page found for '" + command + "'")
			}
		}
		return ManPageOutput{}, err
	}
	content := string(out)
	if cleanSpecialChars {
		content = cleanManOutput(content)
	}
	truncated := false
	if maxLines > 0 {
		lines := strings.Split(content, "\n")
		if len(lines) > maxLines {
			lines = lines[:maxLines]
			truncated = true
		}
		content = strings.Join(lines, "\n")
	}
	return ManPageOutput{
		Command:   command,
		Content:   content,
		Truncated: truncated,
	}, nil
}

func HandleGetManPage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetManPageInput,
) (*mcp.CallToolResult, ManPageOutput, error) {
	if config.IsDisabled("get_man_page") {
		return nil, ManPageOutput{},
			errors.New("tool disabled by configuration")
	}
	if input.Command == "" {
		return nil, ManPageOutput{},
			errors.New("command is required")
	}
	maxLines := min(max(input.MaxLines, 0), 10000)
	ctx, cancel := WithToolTimeout(ctx, "get_man_page", 15*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherManPage(ctx, input.Command, maxLines, true)
	LogToolCall(ctx, "get_man_page", time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
