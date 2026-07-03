package main

import (
	"context"
	"errors"
	"os/exec"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
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

type getMemoryInfoInput struct{}

type memoryInfoOutput struct {
	Total           uint64  `json:"total"`
	Used            uint64  `json:"used"`
	Free            uint64  `json:"free"`
	UsedPercent     float64 `json:"used_percent"`
	SwapTotal       uint64  `json:"swap_total"`
	SwapUsed        uint64  `json:"swap_used"`
	SwapFree        uint64  `json:"swap_free"`
	SwapUsedPercent float64 `json:"swap_used_percent"`
}

type getDiskInfoInput struct {
	MountPoint string `json:"mount_point" jsonschema:"optional mount point filter"`
}

type diskUsageStat struct {
	MountPoint  string  `json:"mount_point"`
	Filesystem  string  `json:"filesystem"`
	Device      string  `json:"device"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type diskInfoOutput struct {
	Partitions []diskUsageStat `json:"partitions"`
}

func handleGetDiskInfo(ctx context.Context, req *mcp.CallToolRequest, input getDiskInfoInput) (*mcp.CallToolResult, diskInfoOutput, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, diskInfoOutput{}, err
	}

	var result []diskUsageStat
	for _, p := range partitions {
		if input.MountPoint != "" && p.Mountpoint != input.MountPoint {
			continue
		}
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		result = append(result, diskUsageStat{
			MountPoint:  p.Mountpoint,
			Filesystem:  p.Fstype,
			Device:      p.Device,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}
	return nil, diskInfoOutput{Partitions: result}, nil
}

func handleGetMemoryInfo(ctx context.Context, req *mcp.CallToolRequest, _ getMemoryInfoInput) (*mcp.CallToolResult, memoryInfoOutput, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return nil, memoryInfoOutput{}, err
	}
	s, err := mem.SwapMemory()
	if err != nil {
		return nil, memoryInfoOutput{}, err
	}
	return nil, memoryInfoOutput{
		Total:           v.Total,
		Used:            v.Used,
		Free:            v.Free,
		UsedPercent:     v.UsedPercent,
		SwapTotal:       s.Total,
		SwapUsed:        s.Used,
		SwapFree:        s.Free,
		SwapUsedPercent: s.UsedPercent,
	}, nil
}

type getNetworkInfoInput struct{}

type interfaceStats struct {
	Name        string `json:"name"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	ErrorsIn    uint64 `json:"errors_in"`
	ErrorsOut   uint64 `json:"errors_out"`
	DropsIn     uint64 `json:"drops_in"`
	DropsOut    uint64 `json:"drops_out"`
}

type networkInfoOutput struct {
	Interfaces []interfaceStats `json:"interfaces"`
}

func handleGetNetworkInfo(ctx context.Context, req *mcp.CallToolRequest, _ getNetworkInfoInput) (*mcp.CallToolResult, networkInfoOutput, error) {
	counters, err := net.IOCounters(true)
	if err != nil {
		return nil, networkInfoOutput{}, err
	}
	var result []interfaceStats
	for _, c := range counters {
		result = append(result, interfaceStats{
			Name:        c.Name,
			BytesSent:   c.BytesSent,
			BytesRecv:   c.BytesRecv,
			PacketsSent: c.PacketsSent,
			PacketsRecv: c.PacketsRecv,
			ErrorsIn:    c.Errin,
			ErrorsOut:   c.Errout,
			DropsIn:     c.Dropin,
			DropsOut:    c.Dropout,
		})
	}
	return nil, networkInfoOutput{Interfaces: result}, nil
}

type getProcessInfoInput struct {
	SortBy string `json:"sort_by" jsonschema:"sort by 'cpu' or 'memory' (default: cpu)"`
	Limit  int    `json:"limit" jsonschema:"max results (default: 10, max: 100)"`
}

type processStat struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
	Status        string  `json:"status"`
}

type processInfoOutput struct {
	Processes []processStat `json:"processes"`
}

func handleGetProcessInfo(ctx context.Context, req *mcp.CallToolRequest, input getProcessInfoInput) (*mcp.CallToolResult, processInfoOutput, error) {
	limit := input.Limit
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	sortBy := input.SortBy
	if sortBy == "" {
		sortBy = "cpu"
	}

	procs, err := process.Processes()
	if err != nil {
		return nil, processInfoOutput{}, err
	}

	var result []processStat
	for _, p := range procs {
		name, _ := p.Name()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		status, _ := p.Status()
		statusStr := ""
		if len(status) > 0 {
			statusStr = status[0]
		}
		result = append(result, processStat{
			PID:           p.Pid,
			Name:          name,
			CPUPercent:    cpu,
			MemoryPercent: mem,
			Status:        statusStr,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if sortBy == "memory" {
			return result[i].MemoryPercent > result[j].MemoryPercent
		}
		return result[i].CPUPercent > result[j].CPUPercent
	})

	if len(result) > limit {
		result = result[:limit]
	}

	return nil, processInfoOutput{Processes: result}, nil
}

type getDockerInfoInput struct{}

type dockerContainer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
}

type dockerImage struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	ID         string `json:"id"`
	Size       string `json:"size"`
}

type dockerInfoOutput struct {
	Containers []dockerContainer `json:"containers"`
	Images     []dockerImage     `json:"images"`
}

func handleGetDockerInfo(ctx context.Context, req *mcp.CallToolRequest, _ getDockerInfoInput) (*mcp.CallToolResult, dockerInfoOutput, error) {
	containers, err := listDockerContainers()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, dockerInfoOutput{}, errors.New("docker is not installed")
		}
		return nil, dockerInfoOutput{}, err
	}
	images, err := listDockerImages()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, dockerInfoOutput{}, errors.New("docker is not installed")
		}
		return nil, dockerInfoOutput{}, err
	}
	return nil, dockerInfoOutput{Containers: containers, Images: images}, nil
}

func listDockerContainers() ([]dockerContainer, error) {
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var containers []dockerContainer
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			continue
		}
		containers = append(containers, dockerContainer{
			ID: parts[0], Name: parts[1], Image: parts[2], Status: parts[3],
		})
	}
	return containers, nil
}

func listDockerImages() ([]dockerImage, error) {
	cmd := exec.Command("docker", "images", "--format", "{{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var images []dockerImage
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			continue
		}
		images = append(images, dockerImage{
			Repository: parts[0], Tag: parts[1], ID: parts[2], Size: parts[3],
		})
	}
	return images, nil
}
