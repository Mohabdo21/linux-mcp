package tools

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetManPageInput struct {
	Command           string `json:"command"                       jsonschema:"command name to get the man page for"`
	MaxLines          int    `json:"max_lines,omitempty"           jsonschema:"maximum number of lines to return (default: 500, max: 10000)"`
	CleanSpecialChars bool   `json:"clean_special_chars,omitempty" jsonschema:"clean backspace formatting characters (default: true)"`
	Search            string `json:"search,omitempty"              jsonschema:"search term to grep for in the man page (case-insensitive)"`
	ContextLines      int    `json:"context_lines,omitempty"       jsonschema:"number of context lines before/after each search match (default: 2 when search is used)"`
	Offset            int    `json:"offset,omitempty"              jsonschema:"line offset to start reading from (0-based)"`
}

type ManPageOutput struct {
	Command   string `json:"command"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated,omitempty"`
	OutputErrors
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
	search string,
	contextLines int,
	offset int,
) (*ManPageOutput, error) {
	if _, err := exec.LookPath("man"); err != nil {
		return nil, errors.New("man command not found")
	}
	out, err := execOutput(ctx, "man", "-P", "cat", command)
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			msg := strings.TrimSpace(string(exitErr.Stderr))
			if msg == "" {
				msg = out
			}
			if strings.Contains(msg, "No manual entry") ||
				strings.Contains(out, "No manual entry") {
				return nil,
					errors.New("No manual page found for '" + command + "'")
			}
		}
		return nil, err
	}
	content := out
	if cleanSpecialChars {
		content = cleanManOutput(content)
	}
	lines := strings.Split(content, "\n")

	// Apply search filter with context lines
	if search != "" {
		searchLower := strings.ToLower(search)
		matched := make([]bool, len(lines))
		for i, line := range lines {
			if strings.Contains(strings.ToLower(line), searchLower) {
				matched[i] = true
			}
		}
		include := make([]bool, len(lines))
		for i, m := range matched {
			if m {
				start := max(i-contextLines, 0)
				end := min(i+contextLines+1, len(lines))
				for j := start; j < end; j++ {
					include[j] = true
				}
			}
		}
		var filtered []string
		for i, incl := range include {
			if incl {
				filtered = append(filtered, lines[i])
			}
		}
		lines = filtered
	}

	// Apply offset
	if offset > 0 && offset < len(lines) {
		lines = lines[offset:]
	}

	// Apply maxLines cap
	truncated := false
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}

	return &ManPageOutput{
		Command:   command,
		Content:   strings.Join(lines, "\n"),
		Truncated: truncated,
	}, nil
}

func HandleGetManPage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetManPageInput,
) (*mcp.CallToolResult, *ManPageOutput, error) {
	if err := requireField(input.Command, "command"); err != nil {
		return nil, nil, err
	}
	maxLines := input.MaxLines
	if maxLines <= 0 {
		maxLines = 500
	}
	maxLines = min(maxLines, 10000)
	contextLines := input.ContextLines
	if input.Search != "" && contextLines <= 0 {
		contextLines = 2
	}
	return handleToolCall(
		ctx,
		"get_man_page",
		0,
		func(ctx context.Context) (*ManPageOutput, error) {
			return GatherManPage(ctx, input.Command, maxLines, true,
				input.Search, contextLines, input.Offset)
		},
	)
}
