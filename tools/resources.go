package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func RegisterResources(server *mcp.Server) {
	resources := []*mcp.Resource{
		{
			URI:         scheme("info"),
			Name:        "System Information",
			Description: "Hostname, OS, kernel version, architecture, and uptime",
		},
		{URI: scheme("cpu"), Name: "CPU Information",
			Description: "CPU usage, model, frequency, and core counts"},
		{URI: scheme("memory"), Name: "Memory Information",
			Description: "RAM and swap usage statistics"},
		{URI: scheme("disk"), Name: "Disk Information",
			Description: "Disk usage for all mounted partitions"},
		{URI: scheme("network"), Name: "Network Information",
			Description: "Network I/O statistics per interface"},
		{URI: scheme("load"), Name: "Load Average",
			Description: "1-, 5-, and 15-minute load averages"},
		{URI: scheme("temperature"), Name: "CPU Temperature",
			Description: "Current CPU temperature from available sensors"},
		{
			URI:         scheme("gpu"),
			Name:        "GPU Information",
			Description: "GPU usage, memory, temperature, and power (NVIDIA/AMD/Intel)",
		},
		{URI: scheme("logged_in_users"), Name: "Logged In Users",
			Description: "Active user sessions"},
		{URI: scheme("listening_ports"), Name: "Listening Ports",
			Description: "Listening ports and associated processes"},
		{URI: scheme("failed_logins"), Name: "Failed Logins",
			Description: "Recent failed login attempts"},
	}
	for _, r := range resources {
		r.MIMEType = "application/json"
		server.AddResource(r, handleReadResource)
	}

	templates := []*mcp.ResourceTemplate{
		{URITemplate: scheme("disk/{mount_point}"),
			Name: "Disk Information (filtered)",
			Description: "Disk usage for a specific mount point, " +
				"e.g. system:///disk// or system:///disk//home"},
		{URITemplate: scheme("service/{name}"),
			Name: "Service Status",
			Description: "Detailed status of a systemd service, " +
				"e.g. system:///service/sshd or system:///service/nginx.service"},
	}
	for _, t := range templates {
		t.MIMEType = "application/json"
		server.AddResourceTemplate(t, handleReadResource)
	}
}

func scheme(path string) string {
	return "system:///" + path
}

func handleReadResource(
	ctx context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid resource URI: %w", err)
	}
	if u.Scheme != "system" {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var (
		data any
		nerr error
	)

	switch path := u.Path; {
	case path == "/info":
		out, e := GatherSystemInfo(ctx)
		data, nerr = &out, e
	case path == "/cpu":
		out, e := GatherCPUInfo(ctx)
		data, nerr = &out, e
	case path == "/memory":
		out, e := GatherMemoryInfo(ctx)
		data, nerr = &out, e
	case path == "/disk":
		out, e := GatherDiskInfo(ctx, "")
		data, nerr = &out, e
	case path == "/network":
		out, e := GatherNetworkInfo(ctx)
		data, nerr = &out, e
	case path == "/load":
		out, e := GatherLoadAverage(ctx)
		data, nerr = &out, e
	case path == "/temperature":
		out, e := GatherCPUTemperature(ctx)
		data, nerr = &out, e
	case path == "/gpu":
		out, e := GatherGPUInfo(ctx)
		data, nerr = &out, e
	case path == "/logged_in_users":
		out, e := GatherLoggedInUsers(ctx)
		data, nerr = &out, e
	case path == "/listening_ports":
		out, e := GatherListeningPorts(ctx, "")
		data, nerr = &out, e
	case path == "/failed_logins":
		out, e := GatherFailedLogins(ctx, 20)
		data, nerr = &out, e
	case strings.HasPrefix(path, "/disk/"):
		out, e := GatherDiskInfo(ctx, strings.TrimPrefix(path, "/disk/"))
		data, nerr = &out, e
	case strings.HasPrefix(path, "/service/"):
		out, e := GatherServiceStatus(
			ctx,
			strings.TrimPrefix(path, "/service/"),
			false,
		)
		data, nerr = &out, e
	default:
		return nil, mcp.ResourceNotFoundError(uri)
	}

	if nerr != nil {
		// If the gather function returned an empty/invalid struct,
		// return a proper error instead of marshaling zeros.
		return nil, fmt.Errorf("resource unavailable: %w", nerr)
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource: %w", err)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}
