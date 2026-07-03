package main

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/cpu"
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

type getCPUInfoInput struct{}

type cpuDetails struct {
	ModelName string  `json:"model_name"`
	CoreCount int32   `json:"core_count"`
	MHz       float64 `json:"mhz"`
}

type cpuInfoOutput struct {
	UsagePercent float64      `json:"usage_percent"`
	PhysicalCoreCount int32   `json:"physical_core_count"`
	Cores        []cpuDetails `json:"cores"`
}

func handleGetCPUInfo(ctx context.Context, req *mcp.CallToolRequest, _ getCPUInfoInput) (*mcp.CallToolResult, cpuInfoOutput, error) {
	info, err := cpu.Info()
	if err != nil {
		return nil, cpuInfoOutput{}, err
	}
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return nil, cpuInfoOutput{}, err
	}
	physCount, err := cpu.Counts(true)
	if err != nil {
		return nil, cpuInfoOutput{}, err
	}
	var cores []cpuDetails
	for _, c := range info {
		cores = append(cores, cpuDetails{
			ModelName: c.ModelName,
			CoreCount: c.Cores,
			MHz:       c.Mhz,
		})
	}
	usage := 0.0
	if len(percent) > 0 {
		usage = percent[0]
	}
	return nil, cpuInfoOutput{UsagePercent: usage, PhysicalCoreCount: int32(physCount), Cores: cores}, nil
}
