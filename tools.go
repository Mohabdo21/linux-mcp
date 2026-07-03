package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
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

type getJournalLogsInput struct {
	Unit     string `json:"unit,omitempty"     jsonschema:"optional systemd unit name (e.g. 'nginx.service')"`
	Priority string `json:"priority,omitempty" jsonschema:"optional log priority: emerg,alert,crit,err,warning,notice,info,debug"`
	Since    string `json:"since,omitempty"    jsonschema:"optional start time (e.g. '1 hour ago', '2024-07-03')"`
	Until    string `json:"until,omitempty"    jsonschema:"optional end time"`
	Lines    int    `json:"lines,omitempty"    jsonschema:"number of recent lines (default: 50)"`
}

type journalLogEntry struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

type journalLogsOutput struct {
	Entries []journalLogEntry `json:"entries"`
}

func gatherJournalLogs(
	unit, priority, since, until string,
	lines int,
) (journalLogsOutput, error) {
	if lines <= 0 {
		lines = 50
	}
	args := []string{
		"--no-pager",
		"-n",
		fmt.Sprintf("%d", lines),
		"-o",
		"short-iso",
	}
	if unit != "" {
		args = append(args, "-u", unit)
	}
	if priority != "" {
		args = append(args, "-p", priority)
	}
	if since != "" {
		args = append(args, "--since", since)
	}
	if until != "" {
		args = append(args, "--until", until)
	}
	cmd := exec.Command("journalctl", args...)
	out, err := cmd.Output()
	if err != nil {
		return journalLogsOutput{}, err
	}
	var entries []journalLogEntry
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		entries = append(entries, journalLogEntry{
			Timestamp: parts[0],
			Message:   parts[2],
		})
	}
	return journalLogsOutput{Entries: entries}, nil
}

func handleGetJournalLogs(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input getJournalLogsInput,
) (*mcp.CallToolResult, journalLogsOutput, error) {
	out, err := gatherJournalLogs(
		input.Unit,
		input.Priority,
		input.Since,
		input.Until,
		input.Lines,
	)
	if err != nil {
		return nil, journalLogsOutput{}, err
	}
	return nil, out, nil
}

type getInodeUsageInput struct {
	MountPoint string `json:"mount_point,omitempty" jsonschema:"optional mount point filter"`
}

type inodeUsageStat struct {
	Filesystem  string `json:"filesystem"`
	Inodes      uint64 `json:"inodes"`
	IUsed       uint64 `json:"iused"`
	IFree       uint64 `json:"ifree"`
	IUsePercent string `json:"iuse_percent"`
	MountedOn   string `json:"mounted_on"`
}

type inodeUsageOutput struct {
	Mounts []inodeUsageStat `json:"mounts"`
}

func gatherInodeUsage(mountPoint string) (inodeUsageOutput, error) {
	cmd := exec.Command("df", "-i")
	out, err := cmd.Output()
	if err != nil {
		return inodeUsageOutput{}, err
	}
	var mounts []inodeUsageStat
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[0] == "Filesystem" {
			continue
		}
		inodes, _ := strconv.ParseUint(fields[1], 10, 64)
		iused, _ := strconv.ParseUint(fields[2], 10, 64)
		ifree, _ := strconv.ParseUint(fields[3], 10, 64)
		mounted := fields[5]

		if mountPoint != "" && mounted != mountPoint {
			continue
		}
		mounts = append(mounts, inodeUsageStat{
			Filesystem:  fields[0],
			Inodes:      inodes,
			IUsed:       iused,
			IFree:       ifree,
			IUsePercent: fields[4],
			MountedOn:   mounted,
		})
	}
	return inodeUsageOutput{Mounts: mounts}, nil
}

func handleGetInodeUsage(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input getInodeUsageInput,
) (*mcp.CallToolResult, inodeUsageOutput, error) {
	out, err := gatherInodeUsage(input.MountPoint)
	if err != nil {
		return nil, inodeUsageOutput{}, err
	}
	return nil, out, nil
}

type getListeningPortsInput struct {
	Protocol string `json:"protocol,omitempty" jsonschema:"optional protocol filter: tcp, udp"`
}

type listeningPort struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     string `json:"port"`
	Process  string `json:"process,omitempty"`
}

type listeningPortsOutput struct {
	Ports []listeningPort `json:"ports"`
}

func gatherListeningPorts(protocol string) (listeningPortsOutput, error) {
	cmd := exec.Command("ss", "-tulnp")
	out, err := cmd.Output()
	if err != nil {
		return listeningPortsOutput{}, err
	}
	var ports []listeningPort
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] == "Netid" {
			continue
		}
		netid := fields[0]
		if protocol != "" && netid != protocol {
			continue
		}
		addr, port, _ := splitHostPort(fields[4])
		proc := ""
		if len(fields) >= 7 {
			proc = parseProcessField(fields[6])
		}
		ports = append(ports, listeningPort{
			Protocol: netid,
			Address:  addr,
			Port:     port,
			Process:  proc,
		})
	}
	return listeningPortsOutput{Ports: ports}, nil
}

func splitHostPort(s string) (string, string, bool) {
	idx := strings.LastIndex(s, ":")
	if idx >= 0 {
		return s[:idx], s[idx+1:], true
	}
	return s, "", false
}

func parseProcessField(s string) string {
	if trimmed, ok := strings.CutPrefix(s, "users:(("); ok {
		if start := strings.Index(trimmed, "\""); start >= 0 {
			trimmed = trimmed[start+1:]
			if before, _, ok := strings.Cut(trimmed, "\""); ok {
				return before
			}
		}
	}
	return s
}

func handleGetListeningPorts(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input getListeningPortsInput,
) (*mcp.CallToolResult, listeningPortsOutput, error) {
	out, err := gatherListeningPorts(input.Protocol)
	if err != nil {
		return nil, listeningPortsOutput{}, err
	}
	return nil, out, nil
}

type getServiceStatusInput struct {
	Name string `json:"name" jsonschema:"service name (e.g. 'nginx.service' or 'sshd')"`
}

type serviceStatusOutput struct {
	Name   string `json:"name"`
	Loaded string `json:"loaded,omitempty"`
	Active string `json:"active,omitempty"`
	PID    string `json:"pid,omitempty"`
	Output string `json:"output"`
}

func gatherServiceStatus(name string) (serviceStatusOutput, error) {
	cmd := exec.Command("systemctl", "status", name, "--no-pager", "-l")
	out, err := cmd.CombinedOutput()
	output := string(out)
	loaded := extractField(output, "Loaded:")
	active := extractField(output, "Active:")
	pid := extractField(output, "Main PID:")
	return serviceStatusOutput{
		Name:   name,
		Loaded: strings.TrimSpace(loaded),
		Active: strings.TrimSpace(active),
		PID:    strings.TrimSpace(pid),
		Output: output,
	}, err
}

func extractField(output, prefix string) string {
	for line := range strings.SplitSeq(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(trimmed, prefix); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}

func handleGetServiceStatus(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input getServiceStatusInput,
) (*mcp.CallToolResult, serviceStatusOutput, error) {
	out, err := gatherServiceStatus(input.Name)
	if err != nil {
		return nil, out, nil
	}
	return nil, out, nil
}

type getTopIOProcessesInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"max results (default: 10, max: 50)"`
}

type ioProcessStat struct {
	Time    string  `json:"time"`
	PID     int     `json:"pid"`
	KbRdS   float64 `json:"kb_rd_s"`
	KbWrS   float64 `json:"kb_wr_s"`
	Command string  `json:"command"`
}

type topIOProcessesOutput struct {
	Processes []ioProcessStat `json:"processes"`
}

func gatherTopIOProcesses(limit int) (topIOProcessesOutput, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}
	cmd := exec.Command("pidstat", "-d", "1", "1")
	out, err := cmd.Output()
	if err != nil {
		return topIOProcessesOutput{}, err
	}
	var procs []ioProcessStat
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[0] == "Linux" || fields[0] == "#" {
			continue
		}
		pid, _ := strconv.Atoi(fields[1])
		rdS, _ := strconv.ParseFloat(fields[2], 64)
		wrS, _ := strconv.ParseFloat(fields[3], 64)
		procs = append(procs, ioProcessStat{
			Time:    fields[0],
			PID:     pid,
			KbRdS:   rdS,
			KbWrS:   wrS,
			Command: strings.Join(fields[5:], " "),
		})
	}
	sort.Slice(procs, func(i, j int) bool {
		return procs[i].KbRdS+procs[i].KbWrS > procs[j].KbRdS+procs[j].KbWrS
	})
	if len(procs) > limit {
		procs = procs[:limit]
	}
	return topIOProcessesOutput{Processes: procs}, nil
}

func handleGetTopIOProcesses(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input getTopIOProcessesInput,
) (*mcp.CallToolResult, topIOProcessesOutput, error) {
	out, err := gatherTopIOProcesses(input.Limit)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, topIOProcessesOutput{}, errors.New(
				"pidstat not installed (install sysstat package)",
			)
		}
		return nil, topIOProcessesOutput{}, err
	}
	return nil, out, nil
}

type getFailedLoginsInput struct {
	Lines int `json:"lines,omitempty" jsonschema:"number of recent entries (default: 20)"`
}

type failedLoginEntry struct {
	Username  string `json:"username"`
	Terminal  string `json:"terminal"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
}

type failedLoginsOutput struct {
	Entries []failedLoginEntry `json:"entries"`
}

func gatherFailedLogins(lines int) (failedLoginsOutput, error) {
	if lines <= 0 {
		lines = 20
	}
	out, err := exec.Command("lastb", "-n", fmt.Sprintf("%d", lines)).Output()
	if err == nil {
		return failedLoginsOutput{Entries: parseLastbOutput(string(out))}, nil
	}
	if !errors.Is(err, exec.ErrNotFound) {
		if entries := parseLastbOutput(string(out)); len(entries) > 0 {
			return failedLoginsOutput{Entries: entries}, nil
		}
	}
	return gatherFailedLoginsJournalctl(lines)
}

func parseLastbOutput(output string) []failedLoginEntry {
	var entries []failedLoginEntry
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line == "" || strings.HasPrefix(line, "btmp begins") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		entries = append(entries, failedLoginEntry{
			Username:  fields[0],
			Terminal:  fields[1],
			Source:    fields[2],
			Timestamp: strings.Join(fields[3:], " "),
		})
	}
	return entries
}

func parseJournalctlFailedLogins(output string) []failedLoginEntry {
	var entries []failedLoginEntry
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		entries = append(entries, failedLoginEntry{
			Timestamp: parts[0],
			Terminal:  parts[1],
			Source:    "",
			Username:  parts[2],
		})
	}
	return entries
}

func gatherFailedLoginsJournalctl(lines int) (failedLoginsOutput, error) {
	out, err := exec.Command(
		"journalctl", "-u", "sshd", "-u", "systemd-logind",
		"--grep", "Failed password|authentication failure|Failed login",
		"--no-pager", "-o", "short-iso", "-n", fmt.Sprintf("%d", lines),
	).Output()
	entries := parseJournalctlFailedLogins(string(out))
	if err != nil && errors.Is(err, exec.ErrNotFound) {
		return failedLoginsOutput{}, err
	}
	return failedLoginsOutput{Entries: entries}, nil
}

func handleGetFailedLogins(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input getFailedLoginsInput,
) (*mcp.CallToolResult, failedLoginsOutput, error) {
	out, err := gatherFailedLogins(input.Lines)
	if err != nil {
		return nil, failedLoginsOutput{}, err
	}
	return nil, out, nil
}
