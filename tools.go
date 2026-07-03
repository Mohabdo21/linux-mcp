package main

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/host"
)

// getSystemInfoInput is intentionally empty (no required params).
type getSystemInfoInput struct{}

type systemInfoOutput struct {
	Hostname      string `json:"hostname"`
	OSName        string `json:"os_name"`
	OSVersion     string `json:"os_version"`
	KernelVersion string `json:"kernel_version"`
	Architecture  string `json:"architecture"`
	UptimeSeconds uint64 `json:"uptime_seconds"`
}

func handleGetSystemInfo(ctx context.Context, req *mcp.CallToolRequest, _ getSystemInfoInput) (*mcp.CallToolResult, systemInfoOutput, error) {
	info, err := host.Info()
	if err != nil {
		return nil, systemInfoOutput{}, err
	}
	return nil, systemInfoOutput{
		Hostname:      info.Hostname,
		OSName:        info.OS,
		OSVersion:     info.PlatformVersion,
		KernelVersion: info.KernelVersion,
		Architecture:  info.KernelArch,
		UptimeSeconds: info.Uptime,
	}, nil
}
