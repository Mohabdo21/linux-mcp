package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type BlockDevice struct {
	Name       string `json:"name"`
	MajorMinor string `json:"major_minor"`
	Size       string `json:"size"`
	Type       string `json:"type"`
	FSType     string `json:"fs_type,omitempty"`
	MountPoint string `json:"mount_point,omitempty"`
	Model      string `json:"model,omitempty"`
	Vendor     string `json:"vendor,omitempty"`
	RO         bool   `json:"ro"`
}

type BlockDevicesOutput struct {
	Devices []BlockDevice `json:"devices"`
	OutputErrors
}

const sysfsBlock = "/sys/block"

func readBlockAttr(dev, attr string) string {
	s, _ := readSysfsFile(filepath.Join(sysfsBlock, dev, attr))
	return s
}

func GatherBlockDevices(ctx context.Context) (*BlockDevicesOutput, error) {
	var out BlockDevicesOutput
	var errs []string

	entries, err := os.ReadDir(sysfsBlock)
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") ||
			name == "zram0" {
			continue
		}

		dev := BlockDevice{
			Name: name,
			RO:   readBlockAttr(name, "ro") == "1",
		}

		sizeRaw := readBlockAttr(name, "size")
		if sizeRaw != "" {
			var sizeKB int64
			if _, serr := fmt.Sscanf(sizeRaw, "%d", &sizeKB); serr == nil {
				dev.Size = HumanSize(sizeKB * 512)
			}
		}

		dev.Model = readBlockAttr(name, "device/model")
		dev.Vendor = readBlockAttr(name, "device/vendor")

		devPath := filepath.Join("/dev", name)
		if info, err := os.Stat(devPath); err == nil {
			if sys, ok := info.Sys().(interface {
				Dev() (uint64, uint64)
			}); ok {
				major, minor := sys.Dev()
				dev.MajorMinor = fmt.Sprintf("%d:%d", major, minor)
			}
		}

		partsDir := filepath.Join(sysfsBlock, name)
		pentries, _ := os.ReadDir(partsDir)
		for _, pe := range pentries {
			pname := pe.Name()
			if !strings.HasPrefix(pname, name) {
				continue
			}
			if _, err := os.Stat(
				filepath.Join(partsDir, pname, "partition"),
			); os.IsNotExist(
				err,
			) {
				continue
			}
			part := BlockDevice{
				Name: pname,
				RO:   readBlockAttr(pname, "ro") == "1",
			}
			pSizeRaw := readBlockAttr(pname, "size")
			if pSizeRaw != "" {
				var sizeKB int64
				if _, perr := fmt.Sscanf(pSizeRaw, "%d", &sizeKB); perr == nil {
					part.Size = HumanSize(sizeKB * 512)
				}
			}
			if fsRaw := readBlockAttr(pname, "queue/rotational"); fsRaw == "0" {
				part.Type = "ssd"
			}
			part.FSType = readBlockAttr(pname, "queue/rotational")
			out.Devices = append(out.Devices, part)
		}

		devices := out.Devices
		out.Devices = append(devices, dev)
	}

	mounts, err := os.Open("/proc/mounts")
	if err == nil {
		defer func() { _ = mounts.Close() }()
		scanner := bufio.NewScanner(mounts)
		for scanner.Scan() {
			line := scanner.Text()
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			devName := filepath.Base(fields[0])
			for i := range out.Devices {
				if out.Devices[i].Name == devName {
					out.Devices[i].MountPoint = fields[1]
					if out.Devices[i].FSType == "" {
						out.Devices[i].FSType = fields[2]
					}
				}
			}
		}
	} else {
		appendErr(&errs, "mounts", err)
	}

	out.Errors = errs
	return &out, out.Err()
}

func HandleGetBlockDevices(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *BlockDevicesOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetBlockDevices,
		0,
		GatherBlockDevices,
	)
}

type SELinuxAppArmorOutput struct {
	SELinux  string `json:"selinux"`
	AppArmor string `json:"apparmor"`
	OutputErrors
}

func GatherSELinuxAppArmorStatus(
	ctx context.Context,
) (*SELinuxAppArmorOutput, error) {
	var out SELinuxAppArmorOutput
	var errs []string

	selinux, err := execOutput(ctx, "getenforce")
	if err == nil {
		out.SELinux = selinux
	} else {
		out.SELinux = "not_enabled"
	}

	enabled, err := readSysfsFile("/sys/module/apparmor/parameters/enabled")
	if err == nil && enabled == "Y" {
		mode, _ := readSysfsFile("/sys/kernel/security/apparmor/profiles")
		if mode != "" {
			out.AppArmor = "enabled"
		} else {
			out.AppArmor = "enabled"
		}
	} else {
		aaOut, err := execOutput(ctx, "aa-status")
		if err == nil && aaOut != "" {
			out.AppArmor = "enabled"
		} else {
			out.AppArmor = "not_enabled"
		}
	}

	out.Errors = errs
	return &out, out.Err()
}

func HandleGetSELinuxAppArmorStatus(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *SELinuxAppArmorOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetSELinuxAppArmorStatus,
		0,
		GatherSELinuxAppArmorStatus,
	)
}

type TimeSyncStatusOutput struct {
	NTPService     string `json:"ntp_service"`
	SyncStatus     string `json:"sync_status"`
	NTPEnabled     bool   `json:"ntp_enabled"`
	SystemClock    string `json:"system_clock_utc"`
	RTCTime        string `json:"rtc_time,omitempty"`
	TimeServer     string `json:"time_server,omitempty"`
	Stratum        int    `json:"stratum,omitempty"`
	LastSyncMs     int    `json:"last_sync_ms,omitempty"`
	ChronyPresent  bool   `json:"chrony_present"`
	NTPDatePresent bool   `json:"ntpdate_present"`
	OutputErrors
}

func GatherTimeSyncStatus(ctx context.Context) (*TimeSyncStatusOutput, error) {
	var out TimeSyncStatusOutput
	var errs []string

	td, err := execOutput(
		ctx,
		"timedatectl",
		"show",
		"--property=NTP",
		"--property=NTPSynchronized",
		"--property=TimeUSec",
		"--property=RTCTimeUSec",
		"--property=NTPService",
	)
	if err == nil {
		for line := range strings.SplitSeq(td, "\n") {
			key, val, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			switch key {
			case "NTP":
				out.NTPEnabled = val == "yes"
			case "NTPSynchronized":
				out.SyncStatus = val
			case "TimeUSec":
				out.SystemClock = val
			case "RTCTimeUSec":
				out.RTCTime = val
			case "NTPService":
				out.NTPService = val
			}
		}
	} else {
		appendErr(&errs, "timedatectl", err)
	}

	chrony, err := execOutput(ctx, "chronyc", "tracking")
	if err == nil {
		out.ChronyPresent = true
		for line := range strings.SplitSeq(chrony, "\n") {
			key, val, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			switch key {
			case "Reference ID":
				out.TimeServer = val
			case "Stratum":
				_, _ = fmt.Sscanf(val, "%d", &out.Stratum)
			case "Last offset":
				var ms float64
				_, _ = fmt.Sscanf(val, "%f", &ms)
				out.LastSyncMs = int(ms * 1000)
			}
		}
	}

	out.Errors = errs
	return &out, out.Err()
}

func HandleGetTimeSyncStatus(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *TimeSyncStatusOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetTimeSyncStatus,
		0,
		GatherTimeSyncStatus,
	)
}

type RAIDDevice struct {
	Name       string `json:"name"`
	Level      string `json:"level"`
	ArraySize  string `json:"array_size"`
	Status     string `json:"status"`
	ActiveDevs int    `json:"active_devices"`
	TotalDevs  int    `json:"total_devices"`
	Devices    string `json:"devices,omitempty"`
}

type RAIDStatusOutput struct {
	Devices []RAIDDevice `json:"devices"`
	OutputErrors
}

func GatherRAIDStatus(ctx context.Context) (*RAIDStatusOutput, error) {
	out := RAIDStatusOutput{Devices: make([]RAIDDevice, 0)}

	data, err := os.ReadFile("/proc/mdstat")
	if err != nil {
		return &out, nil
	}

	var current *RAIDDevice
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Personalities") ||
			strings.HasPrefix(line, "md") {
			if current != nil {
				out.Devices = append(out.Devices, *current)
				current = nil
			}
		}
		if line == "" {
			continue
		}
		if current == nil && !strings.HasPrefix(line, "unused") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				current = &RAIDDevice{Name: fields[0]}
			}
			continue
		}
		if current != nil {
			fields := strings.Fields(line)
			for i, f := range fields {
				switch {
				case strings.HasPrefix(f, "raid"):
					current.Level = f
				case f == "blocks":
					if i > 0 {
						current.ArraySize = fields[i-1]
					}
				case f == "super":

				case strings.Count(f, "/") == 1:
					parts := strings.Split(f, "/")
					if len(parts) == 2 {
						_, _ = fmt.Sscanf(parts[0], "%d", &current.ActiveDevs)
						_, _ = fmt.Sscanf(parts[1], "%d", &current.TotalDevs)
					}
				case f == "[" && i+2 < len(fields):
					if fields[i+1] == "U" || fields[i+1] == "_" {
						current.Devices = strings.Join(fields[i:], " ")
					}
				}
			}
			if current.ActiveDevs == current.TotalDevs {
				current.Status = "active"
			} else if current.ActiveDevs > 0 {
				current.Status = "degraded"
			} else {
				current.Status = "inactive"
			}
		}
	}
	if current != nil {
		out.Devices = append(out.Devices, *current)
	}

	return &out, nil
}

func HandleGetRAIDStatus(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *RAIDStatusOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetRAIDStatus,
		0,
		GatherRAIDStatus,
	)
}

type LogrotateConfig struct {
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
}

type LogrotateStatusOutput struct {
	Configs   []LogrotateConfig `json:"configs"`
	StateFile string            `json:"state_file,omitempty"`
	OutputErrors
}

func GatherLogrotateStatus(
	ctx context.Context,
) (*LogrotateStatusOutput, error) {
	var out LogrotateStatusOutput
	var errs []string

	confDirs := []string{"/etc/logrotate.conf", "/etc/logrotate.d"}
	for _, p := range confDirs {
		info, err := os.Stat(p)
		if err != nil {
			continue
		}
		if info.IsDir() {
			entries, err := os.ReadDir(p)
			if err != nil {
				appendErr(&errs, "read "+p, err)
				continue
			}
			for _, e := range entries {
				if !e.IsDir() &&
					(strings.HasSuffix(e.Name(), ".conf") || !strings.Contains(e.Name(), ".")) {
					out.Configs = append(
						out.Configs,
						LogrotateConfig{Path: filepath.Join(p, e.Name())},
					)
				}
			}
		} else {
			out.Configs = append(out.Configs, LogrotateConfig{Path: p})
		}
	}

	for _, statePath := range []string{"/var/lib/logrotate/status", "/var/lib/logrotate/logrotate.status"} {
		data, err := os.ReadFile(statePath)
		if err == nil {
			out.StateFile = statePath
			_ = data
			break
		}
	}

	out.Errors = errs
	return &out, out.Err()
}

func HandleGetLogrotateStatus(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *LogrotateStatusOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetLogrotateStatus,
		0,
		GatherLogrotateStatus,
	)
}

type CronJobsOutput struct {
	SystemCrontab []CronEntry `json:"system_crontab"`
	DailyJobs     []string    `json:"daily_jobs"`
	WeeklyJobs    []string    `json:"weekly_jobs"`
	HourlyJobs    []string    `json:"hourly_jobs"`
	OutputErrors
}

type CronEntry struct {
	Schedule string `json:"schedule,omitempty"`
	Command  string `json:"command"`
}

func GatherCronJobs(ctx context.Context) (*CronJobsOutput, error) {
	out := CronJobsOutput{
		SystemCrontab: make([]CronEntry, 0),
		DailyJobs:     make([]string, 0),
		WeeklyJobs:    make([]string, 0),
		HourlyJobs:    make([]string, 0),
	}
	var errs []string

	cronDirs := map[string]*[]string{
		"/etc/cron.daily":  &out.DailyJobs,
		"/etc/cron.weekly": &out.WeeklyJobs,
		"/etc/cron.hourly": &out.HourlyJobs,
	}
	for dir, list := range cronDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			appendErr(&errs, "read "+dir, err)
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				*list = append(*list, e.Name())
			}
		}
	}

	crontab, err := os.ReadFile("/etc/crontab")
	if err == nil {
		for line := range strings.SplitSeq(string(crontab), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) > 0 && parts[0] == "SHELL" {
				continue
			}
			if len(parts) > 5 {
				out.SystemCrontab = append(out.SystemCrontab, CronEntry{
					Schedule: strings.Join(parts[:5], " "),
					Command:  strings.Join(parts[5:], " "),
				})
			}
		}
	} else {
		appendErr(&errs, "/etc/crontab", err)
	}

	out.Errors = errs
	return &out, out.Err()
}

func HandleGetCronJobs(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *CronJobsOutput, error) {
	return handleToolCall(ctx, config.ToolNameGetCronJobs, 0, GatherCronJobs)
}

type HealthCheckItem struct {
	Component string `json:"component"`
	Status    string `json:"status"`
	Detail    string `json:"detail,omitempty"`
}

type SystemHealthCheckOutput struct {
	Overall string            `json:"overall"`
	Checks  []HealthCheckItem `json:"checks"`
	OutputErrors
}

const (
	healthOK       = "ok"
	healthWarning  = "warning"
	healthCritical = "critical"
)

func GatherSystemHealthCheck(
	ctx context.Context,
) (*SystemHealthCheckOutput, error) {
	var out SystemHealthCheckOutput
	var errs []string
	hasWarning := false
	hasCritical := false

	mem, err := GatherMemoryInfo(ctx)
	if err == nil {
		if mem.UsedPercent > 90 {
			hasCritical = true
			out.Checks = append(out.Checks, HealthCheckItem{
				Component: "memory", Status: healthCritical,
				Detail: fmt.Sprintf("%.0f%% used", mem.UsedPercent),
			})
		} else if mem.UsedPercent > 80 {
			hasWarning = true
			out.Checks = append(out.Checks, HealthCheckItem{
				Component: "memory", Status: healthWarning,
				Detail: fmt.Sprintf("%.0f%% used", mem.UsedPercent),
			})
		} else {
			out.Checks = append(out.Checks, HealthCheckItem{
				Component: "memory", Status: healthOK,
				Detail: fmt.Sprintf("%.0f%% used", mem.UsedPercent),
			})
		}
	} else {
		appendErr(&errs, "memory", err)
	}

	disk, err := GatherDiskInfo(ctx, "", 80)
	if err == nil {
		for _, p := range disk.Partitions {
			item := HealthCheckItem{Component: "disk:" + p.MountPoint}
			if p.UsedPercent >= 90 {
				hasCritical = true
				item.Status = healthCritical
			} else {
				hasWarning = true
				item.Status = healthWarning
			}
			item.Detail = fmt.Sprintf(
				"%.0f%% used on %s",
				p.UsedPercent,
				p.MountPoint,
			)
			out.Checks = append(out.Checks, item)
		}
		if len(disk.Partitions) == 0 {
			out.Checks = append(out.Checks, HealthCheckItem{
				Component: "disk",
				Status:    healthOK,
				Detail:    "no partitions above 80%",
			})
		}
	} else {
		appendErr(&errs, "disk", err)
	}

	load, err := GatherLoadAverage(ctx)
	if err == nil {
		cpu, err2 := GatherCPUInfo(ctx)
		if err2 == nil && cpu.PhysicalCoreCount > 0 {
			ratio := load.Load15 / float64(cpu.PhysicalCoreCount)
			item := HealthCheckItem{Component: "load"}
			if ratio > 2 {
				hasCritical = true
				item.Status = healthCritical
			} else if ratio > 1 {
				hasWarning = true
				item.Status = healthWarning
			} else {
				item.Status = healthOK
			}
			item.Detail = fmt.Sprintf(
				"load15/core=%.2f (load=%.1f cores=%d)",
				ratio,
				load.Load15,
				cpu.PhysicalCoreCount,
			)
			out.Checks = append(out.Checks, item)
		}
	} else {
		appendErr(&errs, "load", err)
	}

	units, err := GatherSystemdUnits(ctx, "failed")
	if err == nil && len(units.Units) > 0 {
		hasCritical = true
		var failed []string
		for _, u := range units.Units {
			failed = append(failed, u.Unit)
		}
		out.Checks = append(out.Checks, HealthCheckItem{
			Component: "systemd_failed", Status: healthCritical,
			Detail: strings.Join(failed, ", "),
		})
	} else {
		out.Checks = append(out.Checks, HealthCheckItem{
			Component: "systemd", Status: healthOK, Detail: "no failed units",
		})
	}

	switch {
	case hasCritical:
		out.Overall = "CRITICAL"
	case hasWarning:
		out.Overall = "WARNING"
	default:
		out.Overall = "OK"
	}

	out.Errors = errs
	return &out, out.Err()
}

func HandleGetSystemHealthCheck(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *SystemHealthCheckOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetSystemHealthCheck,
		0,
		GatherSystemHealthCheck,
	)
}
