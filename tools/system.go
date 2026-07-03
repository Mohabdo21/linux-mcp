package tools

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/host"
)

type GetSystemInfoInput struct{}

type SystemInfoOutput struct {
	Hostname      string   `json:"hostname"`
	OSName        string   `json:"os_name"`
	OSVersion     string   `json:"os_version"`
	KernelVersion string   `json:"kernel_version"`
	Architecture  string   `json:"architecture"`
	UptimeSeconds uint64   `json:"uptime_seconds"`
	Errors        []string `json:"errors,omitempty"`
}

func GatherSystemInfo() (SystemInfoOutput, error) {
	info, err := host.Info()
	if err != nil {
		return SystemInfoOutput{}, err
	}
	return SystemInfoOutput{
		Hostname:      info.Hostname,
		OSName:        info.OS,
		OSVersion:     info.PlatformVersion,
		KernelVersion: info.KernelVersion,
		Architecture:  info.KernelArch,
		UptimeSeconds: info.Uptime,
	}, nil
}

func HandleGetSystemInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetSystemInfoInput,
) (*mcp.CallToolResult, SystemInfoOutput, error) {
	out, err := GatherSystemInfo()
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}

type GetSystemSnapshotInput struct{}

type SystemSnapshotOutput struct {
	System      SystemInfoOutput     `json:"system"`
	CPU         CPUInfoOutput        `json:"cpu"`
	Temperature CPUTemperatureOutput `json:"temperature"`
	Memory      MemoryInfoOutput     `json:"memory"`
	Disk        DiskInfoOutput       `json:"disk"`
	Network     NetworkInfoOutput    `json:"network"`
	LoadAverage LoadAverageOutput    `json:"load_average"`
	Processes   ProcessInfoOutput    `json:"processes"`
	Docker      DockerInfoOutput     `json:"docker"`
	Errors      ErrList              `json:"errors,omitempty"`
}

func HandleGetSystemSnapshot(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetSystemSnapshotInput,
) (*mcp.CallToolResult, SystemSnapshotOutput, error) {
	var snapshot SystemSnapshotOutput
	var errs ErrList

	if out, err := GatherSystemInfo(); err == nil {
		snapshot.System = out
	} else {
		errs.Add("system", err)
	}

	if out, err := GatherCPUInfo(); err == nil {
		snapshot.CPU = out
	} else {
		errs.Add("cpu", err)
	}

	snapshot.Temperature = GatherCPUTemperature()

	if out, err := GatherMemoryInfo(); err == nil {
		snapshot.Memory = out
	} else {
		errs.Add("memory", err)
	}

	if out, err := GatherDiskInfo(""); err == nil {
		snapshot.Disk = out
	} else {
		errs.Add("disk", err)
	}

	if out, err := GatherNetworkInfo(); err == nil {
		snapshot.Network = out
	} else {
		errs.Add("network", err)
	}

	if out, err := GatherLoadAverage(); err == nil {
		snapshot.LoadAverage = out
	} else {
		errs.Add("load_average", err)
	}

	if out, err := GatherProcessInfo("cpu", 10); err == nil {
		snapshot.Processes = out
	} else {
		errs.Add("processes", err)
	}

	if out, err := GatherDockerInfo(); err == nil {
		snapshot.Docker = out
	} else {
		errs.Add("docker", err)
		snapshot.Docker = DockerInfoOutput{}
	}

	snapshot.Errors = errs
	return nil, snapshot, nil
}

type GetLoadAverageInput struct{}

type LoadAverageOutput struct {
	Load1  float64  `json:"load_1"`
	Load5  float64  `json:"load_5"`
	Load15 float64  `json:"load_15"`
	Errors []string `json:"errors,omitempty"`
}

func GatherLoadAverage() (LoadAverageOutput, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return LoadAverageOutput{}, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return LoadAverageOutput{}, nil
	}
	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)
	return LoadAverageOutput{Load1: load1, Load5: load5, Load15: load15}, nil
}

func HandleGetLoadAverage(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetLoadAverageInput,
) (*mcp.CallToolResult, LoadAverageOutput, error) {
	out, err := GatherLoadAverage()
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
