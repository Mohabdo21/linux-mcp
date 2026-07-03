package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/host"
)

type GetSystemInfoInput struct{}

type SystemInfoOutput struct {
	Hostname      string `json:"hostname"`
	OSName        string `json:"os_name"`
	OSVersion     string `json:"os_version"`
	KernelVersion string `json:"kernel_version"`
	Architecture  string `json:"architecture"`
	UptimeSeconds uint64 `json:"uptime_seconds"`
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
		return nil, SystemInfoOutput{}, err
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
	Processes   ProcessInfoOutput    `json:"processes"`
	Docker      DockerInfoOutput     `json:"docker"`
	Errors      []string             `json:"errors,omitempty"`
}

func HandleGetSystemSnapshot(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetSystemSnapshotInput,
) (*mcp.CallToolResult, SystemSnapshotOutput, error) {
	var snapshot SystemSnapshotOutput
	var errs []string

	if out, err := GatherSystemInfo(); err == nil {
		snapshot.System = out
	} else {
		errs = append(errs, "system: "+err.Error())
	}

	if out, err := GatherCPUInfo(); err == nil {
		snapshot.CPU = out
	} else {
		errs = append(errs, "cpu: "+err.Error())
	}

	snapshot.Temperature = GatherCPUTemperature()

	if out, err := GatherMemoryInfo(); err == nil {
		snapshot.Memory = out
	} else {
		errs = append(errs, "memory: "+err.Error())
	}

	if out, err := GatherDiskInfo(""); err == nil {
		snapshot.Disk = out
	} else {
		errs = append(errs, "disk: "+err.Error())
	}

	if out, err := GatherNetworkInfo(); err == nil {
		snapshot.Network = out
	} else {
		errs = append(errs, "network: "+err.Error())
	}

	if out, err := GatherProcessInfo("cpu", 10); err == nil {
		snapshot.Processes = out
	} else {
		errs = append(errs, "processes: "+err.Error())
	}

	if out, err := GatherDockerInfo(); err == nil {
		snapshot.Docker = out
	} else {
		errs = append(errs, "docker: "+err.Error())
		snapshot.Docker = DockerInfoOutput{}
	}

	snapshot.Errors = errs
	return nil, snapshot, nil
}
