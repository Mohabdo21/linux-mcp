package tools

import (
	"context"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetServiceStatusInput struct {
	Name string `json:"name"           jsonschema:"service name (e.g. 'nginx.service' or 'sshd')"`
	User bool   `json:"user,omitempty" jsonschema:"query user-level service (default: false)"`
}

type ServiceStatusOutput struct {
	Name   string `json:"name"`
	Loaded string `json:"loaded,omitempty"`
	Active string `json:"active,omitempty"`
	PID    string `json:"pid,omitempty"`
	Output string `json:"output"`
}

func ExtractField(output, prefix string) string {
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, prefix); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

func GatherServiceStatus(
	name string, user bool,
) (ServiceStatusOutput, error) {
	args := []string{"status", name, "--no-pager", "-l"}
	if user {
		args = append([]string{"--user"}, args...)
	}
	cmd := exec.Command("systemctl", args...)
	out, err := cmd.CombinedOutput()
	output := string(out)
	loaded := ExtractField(output, "Loaded:")
	active := ExtractField(output, "Active:")
	pid := ExtractField(output, "Main PID:")
	return ServiceStatusOutput{
		Name:   name,
		Loaded: strings.TrimSpace(loaded),
		Active: strings.TrimSpace(active),
		PID:    strings.TrimSpace(pid),
		Output: output,
	}, err
}

func HandleGetServiceStatus(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetServiceStatusInput,
) (*mcp.CallToolResult, ServiceStatusOutput, error) {
	out, err := GatherServiceStatus(input.Name, input.User)
	if err != nil {
		return nil, out, nil
	}
	return nil, out, nil
}
