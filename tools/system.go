package tools

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
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

func GatherSystemInfo(ctx context.Context) (SystemInfoOutput, error) {
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
	if config.IsDisabled("get_system_info") {
		return nil, SystemInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_system_info", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherSystemInfo(ctx)
	LogToolCall(ctx, "get_system_info",
		time.Since(start), len(out.Errors))
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
	if config.IsDisabled("get_system_snapshot") {
		return nil, SystemSnapshotOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_system_snapshot", 120*time.Second)
	defer cancel()

	start := time.Now()

	var snapshot SystemSnapshotOutput
	var errs ErrList

	if out, err := GatherSystemInfo(ctx); err == nil {
		snapshot.System = out
	} else {
		errs.Add("system", err)
	}

	if out, err := GatherCPUInfo(ctx); err == nil {
		snapshot.CPU = out
	} else {
		errs.Add("cpu", err)
	}

	if out, err := GatherCPUTemperature(ctx); err == nil {
		snapshot.Temperature = out
	} else {
		errs.Add("temperature", err)
	}

	if out, err := GatherMemoryInfo(ctx); err == nil {
		snapshot.Memory = out
	} else {
		errs.Add("memory", err)
	}

	if out, err := GatherDiskInfo(ctx, ""); err == nil {
		snapshot.Disk = out
	} else {
		errs.Add("disk", err)
	}

	if out, err := GatherNetworkInfo(ctx); err == nil {
		snapshot.Network = out
	} else {
		errs.Add("network", err)
	}

	if out, err := GatherLoadAverage(ctx); err == nil {
		snapshot.LoadAverage = out
	} else {
		errs.Add("load_average", err)
	}

	if out, err := GatherProcessInfo(ctx, "cpu", 10); err == nil {
		snapshot.Processes = out
	} else {
		errs.Add("processes", err)
	}

	if out, err := GatherDockerInfo(ctx); err == nil {
		snapshot.Docker = out
	} else {
		errs.Add("docker", err)
		snapshot.Docker = DockerInfoOutput{}
	}

	snapshot.Errors = errs
	LogToolCall(ctx, "get_system_snapshot",
		time.Since(start), len(errs))
	return nil, snapshot, nil
}

type GetLoadAverageInput struct{}

type LoadAverageOutput struct {
	Load1  float64  `json:"load_1"`
	Load5  float64  `json:"load_5"`
	Load15 float64  `json:"load_15"`
	Errors []string `json:"errors,omitempty"`
}

func GatherLoadAverage(ctx context.Context) (LoadAverageOutput, error) {
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
	if config.IsDisabled("get_load_average") {
		return nil, LoadAverageOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_load_average", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherLoadAverage(ctx)
	LogToolCall(ctx, "get_load_average",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
