package main

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/sensors"
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

type getCPUTemperatureInput struct{}

type temperatureStat struct {
	SensorKey   string  `json:"sensor_key"`
	Temperature float64 `json:"temperature_celsius"`
}

type cpuTemperatureOutput struct {
	Temperatures []temperatureStat `json:"temperatures"`
	Message      string            `json:"message,omitempty"`
}

func handleGetCPUTemperature(ctx context.Context, req *mcp.CallToolRequest, _ getCPUTemperatureInput) (*mcp.CallToolResult, cpuTemperatureOutput, error) {
	temps, err := sensors.SensorsTemperatures()
	if err != nil {
		return nil, cpuTemperatureOutput{}, err
	}
	if len(temps) == 0 {
		return nil, cpuTemperatureOutput{Message: "No temperature sensors available"}, nil
	}
	var result []temperatureStat
	for _, t := range temps {
		result = append(result, temperatureStat{
			SensorKey:   t.SensorKey,
			Temperature: t.Temperature,
		})
	}
	return nil, cpuTemperatureOutput{Temperatures: result}, nil
}
