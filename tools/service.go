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
	Name   string   `json:"name"`
	Loaded string   `json:"loaded,omitempty"`
	Active string   `json:"active,omitempty"`
	PID    string   `json:"pid,omitempty"`
	Output string   `json:"output"`
	Errors []string `json:"errors,omitempty"`
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

type GetSystemdUnitsInput struct{}

type SystemdUnit struct {
	Unit        string `json:"unit"`
	Load        string `json:"load"`
	Active      string `json:"active"`
	Sub         string `json:"sub"`
	Description string `json:"description"`
}

type SystemdUnitsOutput struct {
	Units  []SystemdUnit `json:"units"`
	Errors []string      `json:"errors,omitempty"`
}

func GatherSystemdUnits() (SystemdUnitsOutput, error) {
	cmd := exec.Command(
		"systemctl",
		"list-units",
		"--all",
		"--no-pager",
		"--no-legend",
	)
	out, err := cmd.Output()
	if err != nil {
		return SystemdUnitsOutput{}, err
	}
	var units []SystemdUnit
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		desc := ""
		if len(fields) > 4 {
			desc = strings.Join(fields[4:], " ")
		}
		units = append(units, SystemdUnit{
			Unit:        fields[0],
			Load:        fields[1],
			Active:      fields[2],
			Sub:         fields[3],
			Description: desc,
		})
	}
	return SystemdUnitsOutput{Units: units}, nil
}

func HandleGetSystemdUnits(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetSystemdUnitsInput,
) (*mcp.CallToolResult, SystemdUnitsOutput, error) {
	out, err := GatherSystemdUnits()
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}

func HandleGetServiceStatus(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetServiceStatusInput,
) (*mcp.CallToolResult, ServiceStatusOutput, error) {
	out, err := GatherServiceStatus(input.Name, input.User)
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
