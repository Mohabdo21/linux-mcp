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

type getSystemInfoInput struct{}

type systemInfoOutput struct {
	Hostname      string `json:"hostname"`
	OSName        string `json:"os_name"`
	OSVersion     string `json:"os_version"`
	KernelVersion string `json:"kernel_version"`
	Architecture  string `json:"architecture"`
	UptimeSeconds uint64 `json:"uptime_seconds"`
}

func gatherSystemInfo() (systemInfoOutput, error) {
	info, err := host.Info()
	if err != nil {
		return systemInfoOutput{}, err
	}
	return systemInfoOutput{
		Hostname:      info.Hostname,
		OSName:        info.OS,
		OSVersion:     info.PlatformVersion,
		KernelVersion: info.KernelVersion,
		Architecture:  info.KernelArch,
		UptimeSeconds: info.Uptime,
	}, nil
}

func handleGetSystemInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ getSystemInfoInput,
) (*mcp.CallToolResult, systemInfoOutput, error) {
	out, err := gatherSystemInfo()
	if err != nil {
		return nil, systemInfoOutput{}, err
	}
	return nil, out, nil
}

type getCPUInfoInput struct{}

type cpuDetails struct {
	ModelName string  `json:"model_name"`
	CoreCount int32   `json:"core_count"`
	MHz       float64 `json:"mhz"`
}

type cpuInfoOutput struct {
	UsagePercent      float64      `json:"usage_percent"`
	PhysicalCoreCount int32        `json:"physical_core_count"`
	Cores             []cpuDetails `json:"cores"`
}

func gatherCPUInfo() (cpuInfoOutput, error) {
	info, err := cpu.Info()
	if err != nil {
		return cpuInfoOutput{}, err
	}
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return cpuInfoOutput{}, err
	}
	physCount, err := cpu.Counts(true)
	if err != nil {
		return cpuInfoOutput{}, err
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
	return cpuInfoOutput{
		UsagePercent:      usage,
		PhysicalCoreCount: int32(physCount),
		Cores:             cores,
	}, nil
}

func handleGetCPUInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ getCPUInfoInput,
) (*mcp.CallToolResult, cpuInfoOutput, error) {
	out, err := gatherCPUInfo()
	if err != nil {
		return nil, cpuInfoOutput{}, err
	}
	return nil, out, nil
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

func gatherCPUTemperature() cpuTemperatureOutput {
	temps, err := sensors.SensorsTemperatures()
	if len(temps) == 0 {
		msg := "No temperature sensors available"
		if err != nil {
			msg = err.Error()
		}
		return cpuTemperatureOutput{Message: msg}
	}
	var result []temperatureStat
	for _, t := range temps {
		result = append(result, temperatureStat{
			SensorKey:   t.SensorKey,
			Temperature: t.Temperature,
		})
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return cpuTemperatureOutput{Temperatures: result, Message: msg}
}

func handleGetCPUTemperature(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ getCPUTemperatureInput,
) (*mcp.CallToolResult, cpuTemperatureOutput, error) {
	return nil, gatherCPUTemperature(), nil
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

func gatherMemoryInfo() (memoryInfoOutput, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return memoryInfoOutput{}, err
	}
	s, err := mem.SwapMemory()
	if err != nil {
		return memoryInfoOutput{}, err
	}
	return memoryInfoOutput{
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

func handleGetMemoryInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ getMemoryInfoInput,
) (*mcp.CallToolResult, memoryInfoOutput, error) {
	out, err := gatherMemoryInfo()
	if err != nil {
		return nil, memoryInfoOutput{}, err
	}
	return nil, out, nil
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

func gatherDiskInfo(mountPoint string) (diskInfoOutput, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return diskInfoOutput{}, err
	}

	var result []diskUsageStat
	for _, p := range partitions {
		if mountPoint != "" && p.Mountpoint != mountPoint {
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
	return diskInfoOutput{Partitions: result}, nil
}

func handleGetDiskInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input getDiskInfoInput,
) (*mcp.CallToolResult, diskInfoOutput, error) {
	out, err := gatherDiskInfo(input.MountPoint)
	if err != nil {
		return nil, diskInfoOutput{}, err
	}
	return nil, out, nil
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

func gatherNetworkInfo() (networkInfoOutput, error) {
	counters, err := net.IOCounters(true)
	if err != nil {
		return networkInfoOutput{}, err
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
	return networkInfoOutput{Interfaces: result}, nil
}

func handleGetNetworkInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ getNetworkInfoInput,
) (*mcp.CallToolResult, networkInfoOutput, error) {
	out, err := gatherNetworkInfo()
	if err != nil {
		return nil, networkInfoOutput{}, err
	}
	return nil, out, nil
}

type getProcessInfoInput struct {
	SortBy string `json:"sort_by" jsonschema:"sort by 'cpu' or 'memory' (default: cpu)"`
	Limit  int    `json:"limit"   jsonschema:"max results (default: 10, max: 100)"`
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

func gatherProcessInfo(sortBy string, limit int) (processInfoOutput, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	if sortBy == "" {
		sortBy = "cpu"
	}

	procs, err := process.Processes()
	if err != nil {
		return processInfoOutput{}, err
	}

	var result []processStat
	for _, p := range procs {
		name, _ := p.Name()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		status, _ := p.Status()
		statusStr := strings.Join(status, ",")
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

	return processInfoOutput{Processes: result}, nil
}

func handleGetProcessInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input getProcessInfoInput,
) (*mcp.CallToolResult, processInfoOutput, error) {
	out, err := gatherProcessInfo(input.SortBy, input.Limit)
	if err != nil {
		return nil, processInfoOutput{}, err
	}
	return nil, out, nil
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

func gatherDockerInfo() (dockerInfoOutput, error) {
	containers, err := listDockerContainers()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return dockerInfoOutput{}, errors.New("docker is not installed")
		}
		return dockerInfoOutput{}, err
	}
	images, err := listDockerImages()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return dockerInfoOutput{}, errors.New("docker is not installed")
		}
		return dockerInfoOutput{}, err
	}
	return dockerInfoOutput{Containers: containers, Images: images}, nil
}

func handleGetDockerInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ getDockerInfoInput,
) (*mcp.CallToolResult, dockerInfoOutput, error) {
	out, err := gatherDockerInfo()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, dockerInfoOutput{}, errors.New(
				"docker is not installed",
			)
		}
		return nil, dockerInfoOutput{}, err
	}
	return nil, out, nil
}

type getSystemSnapshotInput struct{}

type systemSnapshotOutput struct {
	System      systemInfoOutput     `json:"system"`
	CPU         cpuInfoOutput        `json:"cpu"`
	Temperature cpuTemperatureOutput `json:"temperature"`
	Memory      memoryInfoOutput     `json:"memory"`
	Disk        diskInfoOutput       `json:"disk"`
	Network     networkInfoOutput    `json:"network"`
	Processes   processInfoOutput    `json:"processes"`
	Docker      dockerInfoOutput     `json:"docker"`
	Errors      []string             `json:"errors,omitempty"`
}

func handleGetSystemSnapshot(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ getSystemSnapshotInput,
) (*mcp.CallToolResult, systemSnapshotOutput, error) {
	var snapshot systemSnapshotOutput
	var errs []string

	if out, err := gatherSystemInfo(); err == nil {
		snapshot.System = out
	} else {
		errs = append(errs, "system: "+err.Error())
	}

	if out, err := gatherCPUInfo(); err == nil {
		snapshot.CPU = out
	} else {
		errs = append(errs, "cpu: "+err.Error())
	}

	snapshot.Temperature = gatherCPUTemperature()

	if out, err := gatherMemoryInfo(); err == nil {
		snapshot.Memory = out
	} else {
		errs = append(errs, "memory: "+err.Error())
	}

	if out, err := gatherDiskInfo(""); err == nil {
		snapshot.Disk = out
	} else {
		errs = append(errs, "disk: "+err.Error())
	}

	if out, err := gatherNetworkInfo(); err == nil {
		snapshot.Network = out
	} else {
		errs = append(errs, "network: "+err.Error())
	}

	if out, err := gatherProcessInfo("cpu", 10); err == nil {
		snapshot.Processes = out
	} else {
		errs = append(errs, "processes: "+err.Error())
	}

	if out, err := gatherDockerInfo(); err == nil {
		snapshot.Docker = out
	} else {
		errs = append(errs, "docker: "+err.Error())
		snapshot.Docker = dockerInfoOutput{}
	}

	snapshot.Errors = errs
	return nil, snapshot, nil
}

func listDockerContainers() ([]dockerContainer, error) {
	cmd := exec.Command(
		"docker",
		"ps",
		"-a",
		"--format",
		"{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var containers []dockerContainer
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
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
	cmd := exec.Command(
		"docker",
		"images",
		"--format",
		"{{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var images []dockerImage
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
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
