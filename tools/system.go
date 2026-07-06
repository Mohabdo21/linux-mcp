package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/host"
)

type SystemInfoOutput struct {
	Hostname             string `json:"hostname"`
	OSName               string `json:"os_name"`
	OSVersion            string `json:"os_version"`
	KernelVersion        string `json:"kernel_version"`
	Architecture         string `json:"architecture"`
	UptimeSeconds        uint64 `json:"uptime_seconds"`
	Platform             string `json:"platform,omitempty"`
	PlatformFamily       string `json:"platform_family,omitempty"`
	BootTime             uint64 `json:"boot_time,omitempty"`
	Procs                uint64 `json:"procs,omitempty"`
	VirtualizationSystem string `json:"virtualization_system,omitempty"`
	VirtualizationRole   string `json:"virtualization_role,omitempty"`
	HostID               string `json:"host_id,omitempty"`
	Manufacturer         string `json:"manufacturer,omitempty"`
	ProductName          string `json:"product_name,omitempty"`
	ProductVersion       string `json:"product_version,omitempty"`
	BIOSVersion          string `json:"bios_version,omitempty"`
	BIOSDate             string `json:"bios_date,omitempty"`
	TPMVersion           string `json:"tpm_version,omitempty"`
	OutputErrors
}

func readDMIField(path string) string {
	s, _ := readSysfsFile(path)
	return s
}

func readTPMVersion(ctx context.Context) string {
	data, err := os.ReadFile("/sys/class/tpm/tpm0/tpm_version_str")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func GatherSystemInfo(ctx context.Context) (*SystemInfoOutput, error) {
	info, err := host.Info()
	if err != nil {
		return nil, err
	}
	return &SystemInfoOutput{
		Hostname:             info.Hostname,
		OSName:               info.OS,
		OSVersion:            info.PlatformVersion,
		KernelVersion:        info.KernelVersion,
		Architecture:         info.KernelArch,
		UptimeSeconds:        info.Uptime,
		Platform:             info.Platform,
		PlatformFamily:       info.PlatformFamily,
		BootTime:             info.BootTime,
		Procs:                info.Procs,
		VirtualizationSystem: info.VirtualizationSystem,
		VirtualizationRole:   info.VirtualizationRole,
		HostID:               info.HostID,
		Manufacturer: readDMIField(
			"/sys/devices/virtual/dmi/id/sys_vendor",
		),
		ProductName: readDMIField(
			"/sys/devices/virtual/dmi/id/product_name",
		),
		ProductVersion: readDMIField(
			"/sys/devices/virtual/dmi/id/product_version",
		),
		BIOSVersion: readDMIField(
			"/sys/devices/virtual/dmi/id/bios_version",
		),
		BIOSDate: readDMIField(
			"/sys/devices/virtual/dmi/id/bios_date",
		),
		TPMVersion: readTPMVersion(ctx),
	}, nil
}

func HandleGetSystemInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *SystemInfoOutput, error) {
	return handleToolCall(
		ctx,
		"get_system_info",
		0,
		GatherSystemInfo,
	)
}

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
	OutputErrors
}

func HandleGetSystemSnapshot(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *SystemSnapshotOutput, error) {
	return handleToolCall(
		ctx,
		"get_system_snapshot",
		0,
		func(ctx context.Context) (*SystemSnapshotOutput, error) {
			var snapshot SystemSnapshotOutput
			var errs []string

			snapshot.System = collectOrFallback(ctx,
				"system", GatherSystemInfo,
				SystemInfoOutput{}, &errs)
			snapshot.CPU = collectOrFallback(ctx,
				"cpu", GatherCPUInfo,
				CPUInfoOutput{Cores: []CPUDetails{}}, &errs)
			snapshot.Temperature = collectOrFallback(ctx,
				"temperature", GatherCPUTemperature,
				CPUTemperatureOutput{Temperatures: []TemperatureStat{}}, &errs)
			snapshot.Memory = collectOrFallback(ctx,
				"memory", GatherMemoryInfo,
				MemoryInfoOutput{}, &errs)
			snapshot.Disk = collectOrFallback(ctx,
				"disk", func(ctx context.Context) (*DiskInfoOutput, error) {
					return GatherDiskInfo(ctx, "")
				}, DiskInfoOutput{Partitions: []DiskUsageStat{}}, &errs)
			snapshot.Network = collectOrFallback(ctx,
				"network", GatherNetworkInfo,
				NetworkInfoOutput{Interfaces: []InterfaceStats{}}, &errs)
			snapshot.LoadAverage = collectOrFallback(ctx,
				"load_average", GatherLoadAverage,
				LoadAverageOutput{}, &errs)
			snapshot.Processes = collectOrFallback(
				ctx,
				"processes",
				func(ctx context.Context) (*ProcessInfoOutput, error) {
					return GatherProcessInfo(ctx, "cpu", 10)
				},
				ProcessInfoOutput{Processes: []ProcessStat{}},
				&errs,
			)
			snapshot.Docker = collectOrFallback(ctx,
				"docker", GatherDockerInfo,
				DockerInfoOutput{
					Containers: []DockerContainer{},
					Images:     []DockerImage{},
				}, &errs)

			snapshot.Errors = errs
			return &snapshot, nil
		},
	)
}

type LoadAverageOutput struct {
	Load1  float64 `json:"load_1"`
	Load5  float64 `json:"load_5"`
	Load15 float64 `json:"load_15"`
	OutputErrors
}

func GatherLoadAverage(ctx context.Context) (*LoadAverageOutput, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return &LoadAverageOutput{}, nil
	}
	load1, _ := strconv.ParseFloat(fields[0], 64)
	load5, _ := strconv.ParseFloat(fields[1], 64)
	load15, _ := strconv.ParseFloat(fields[2], 64)
	return &LoadAverageOutput{Load1: load1, Load5: load5, Load15: load15}, nil
}

type GetEnvironmentVariablesInput struct {
	Search string `json:"search,omitempty" jsonschema:"optional search string to filter by name (matches prefix or substring, case-insensitive)"`
}

type EnvironmentVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type EnvironmentVariablesOutput struct {
	Variables []EnvironmentVariable `json:"variables"`
	Count     int                   `json:"count"`
	OutputErrors
}

func GatherEnvironmentVariables(
	ctx context.Context,
	search string,
) (*EnvironmentVariablesOutput, error) {
	env := os.Environ()
	variables := make([]EnvironmentVariable, 0, len(env))
	for _, pair := range env {
		for i := 0; i < len(pair); i++ {
			if pair[i] == '=' {
				name := pair[:i]
				if search != "" {
					s := strings.ToLower(search)
					lower := strings.ToLower(name)
					if !strings.HasPrefix(lower, s) &&
						!strings.Contains(lower, s) {
						continue
					}
				}
				variables = append(variables, EnvironmentVariable{
					Name:  name,
					Value: pair[i+1:],
				})
				break
			}
		}
	}
	sort.Slice(variables, func(i, j int) bool {
		return variables[i].Name < variables[j].Name
	})
	return &EnvironmentVariablesOutput{
		Variables: variables,
		Count:     len(variables),
	}, nil
}

func HandleGetEnvironmentVariables(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetEnvironmentVariablesInput,
) (*mcp.CallToolResult, *EnvironmentVariablesOutput, error) {
	return handleToolCall(
		ctx,
		"get_environment_variables",
		0,
		func(ctx context.Context) (*EnvironmentVariablesOutput, error) {
			return GatherEnvironmentVariables(ctx, input.Search)
		},
	)
}

type GetHardwareBusInfoInput struct {
	Search string `json:"search,omitempty" jsonschema:"optional search string to filter devices by any field (bus, slot, class, vendor, device)"`
}

type BusDevice struct {
	Bus    string `json:"bus"`
	Slot   string `json:"slot,omitempty"`
	Class  string `json:"class,omitempty"`
	Vendor string `json:"vendor,omitempty"`
	Device string `json:"device"`
}

type HardwareBusInfoOutput struct {
	PCIDevices []BusDevice `json:"pci_devices"`
	USBDevices []BusDevice `json:"usb_devices"`
	OutputErrors
}

const (
	pciSysfsPath = "/sys/bus/pci/devices"
	usbSysfsPath = "/sys/bus/usb/devices"
)

var pciBaseClasses = map[uint8]string{
	0x00: "Unclassified device",
	0x01: "Mass storage controller",
	0x02: "Network controller",
	0x03: "Display controller",
	0x04: "Multimedia controller",
	0x05: "Memory controller",
	0x06: "Bridge",
	0x07: "Communication controller",
	0x08: "Base system peripheral",
	0x09: "Input device controller",
	0x0a: "Docking station",
	0x0b: "Processor",
	0x0c: "Serial bus controller",
	0x0d: "Wireless controller",
	0x0e: "Intelligent controller",
	0x0f: "Satellite communication controller",
	0x10: "Encryption controller",
	0x11: "Signal processing controller",
	0x12: "Processing accelerators",
	0x13: "Non-Essential Instrumentation",
	0x14: "Coprocessor",
}

func parsePCIDevicesSysfs(ctx context.Context) ([]BusDevice, error) {
	entries, err := os.ReadDir(pciSysfsPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", pciSysfsPath, err)
	}

	var devices []BusDevice
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		base := filepath.Join(pciSysfsPath, name)

		vendor, _ := readSysfsFile(filepath.Join(base, "vendor"))
		device, _ := readSysfsFile(filepath.Join(base, "device"))
		classHex, _ := readSysfsFile(filepath.Join(base, "class"))

		className := ""
		if len(classHex) >= 4 {
			val, err := strconv.ParseUint(classHex[2:4], 16, 8)
			if err == nil {
				if n, ok := pciBaseClasses[uint8(val)]; ok {
					className = n
				}
			}
		}

		devices = append(devices, BusDevice{
			Bus:    "pci",
			Slot:   name,
			Class:  className,
			Vendor: vendor,
			Device: device,
		})
	}

	if devices == nil {
		devices = []BusDevice{}
	}
	return devices, nil
}

func parseUSBDevicesSysfs(ctx context.Context) ([]BusDevice, error) {
	entries, err := os.ReadDir(usbSysfsPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", usbSysfsPath, err)
	}

	var devices []BusDevice
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		base := filepath.Join(usbSysfsPath, entry.Name())

		vendor, err := readSysfsFile(filepath.Join(base, "idVendor"))
		if err != nil {
			continue // skip interfaces without vendor info
		}
		product, _ := readSysfsFile(filepath.Join(base, "idProduct"))
		desc, _ := readSysfsFile(filepath.Join(base, "product"))
		busnum, _ := readSysfsFile(filepath.Join(base, "busnum"))
		devnum, _ := readSysfsFile(filepath.Join(base, "devnum"))

		deviceDesc := desc
		if deviceDesc == "" {
			deviceDesc = vendor + ":" + product
		}

		devices = append(devices, BusDevice{
			Bus:    busnum,
			Slot:   devnum,
			Vendor: vendor,
			Device: deviceDesc,
		})
	}

	if devices == nil {
		devices = []BusDevice{}
	}
	return devices, nil
}

func parseLspciOutput(ctx context.Context) ([]BusDevice, error) {
	lines, err := execLines(ctx, "lspci")
	if err != nil {
		return nil, err
	}
	var devices []BusDevice
	for _, line := range lines {
		line = strings.TrimSpace(line)
		slot, rest, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		class, desc, _ := strings.Cut(rest, ": ")
		devices = append(devices, BusDevice{
			Bus:    "pci",
			Slot:   slot,
			Class:  strings.TrimSpace(class),
			Device: strings.TrimSpace(desc),
		})
	}
	if devices == nil {
		devices = []BusDevice{}
	}
	return devices, nil
}

func parseLsusbOutput(ctx context.Context) ([]BusDevice, error) {
	lines, err := execLines(ctx, "lsusb")
	if err != nil {
		return nil, err
	}
	var devices []BusDevice
	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.Fields(line)
		if len(parts) < 6 {
			continue
		}
		bus := parts[0]
		device := parts[3]
		vendor := strings.TrimRight(parts[5], ":")
		desc := strings.Join(parts[6:], " ")
		if desc == "" {
			desc = vendor
			vendor = ""
		}
		devices = append(devices, BusDevice{
			Bus:    bus,
			Slot:   device,
			Vendor: vendor,
			Device: desc,
		})
	}
	if devices == nil {
		devices = []BusDevice{}
	}
	return devices, nil
}

func deviceMatches(d BusDevice, search string) bool {
	if search == "" {
		return true
	}
	search = strings.ToLower(search)
	return strings.Contains(strings.ToLower(d.Bus), search) ||
		strings.Contains(strings.ToLower(d.Slot), search) ||
		strings.Contains(strings.ToLower(d.Class), search) ||
		strings.Contains(strings.ToLower(d.Vendor), search) ||
		strings.Contains(strings.ToLower(d.Device), search)
}

func filterDevices(devices []BusDevice, search string) []BusDevice {
	if search == "" {
		return devices
	}
	var filtered []BusDevice
	for _, d := range devices {
		if deviceMatches(d, search) {
			filtered = append(filtered, d)
		}
	}
	if filtered == nil {
		filtered = []BusDevice{}
	}
	return filtered
}

func GatherHardwareBusInfo(
	ctx context.Context,
	search string,
) (*HardwareBusInfoOutput, error) {
	var out HardwareBusInfoOutput
	var errs []string

	pci, err := parseLspciOutput(ctx)
	if err != nil {
		pci, err = parsePCIDevicesSysfs(ctx)
		if err != nil {
			appendErr(&errs, "pci", err)
		}
	}
	if pci != nil {
		out.PCIDevices = filterDevices(pci, search)
	}

	usb, err := parseLsusbOutput(ctx)
	if err != nil {
		usb, err = parseUSBDevicesSysfs(ctx)
		if err != nil {
			appendErr(&errs, "usb", err)
		}
	}
	if usb != nil {
		out.USBDevices = filterDevices(usb, search)
	}

	out.Errors = errs
	return &out, out.Err()
}

func HandleGetHardwareBusInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetHardwareBusInfoInput,
) (*mcp.CallToolResult, *HardwareBusInfoOutput, error) {
	return handleToolCall(
		ctx,
		"get_hardware_bus_info",
		0,
		func(ctx context.Context) (*HardwareBusInfoOutput, error) {
			return GatherHardwareBusInfo(ctx, input.Search)
		},
	)
}

func HandleGetLoadAverage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *LoadAverageOutput, error) {
	return handleToolCall(
		ctx,
		"get_load_average",
		0,
		GatherLoadAverage,
	)
}

type CronJob struct {
	Schedule string `json:"schedule"`
	Command  string `json:"command"`
}

type SystemdTimer struct {
	Unit      string `json:"unit"`
	Activates string `json:"activates"`
	Next      string `json:"next,omitempty"`
	Last      string `json:"last,omitempty"`
}

type UserAutomationOutput struct {
	CronJobs      []CronJob      `json:"cron_jobs"`
	SystemdTimers []SystemdTimer `json:"systemd_timers"`
	OutputErrors
}

func GatherUserAutomation(
	ctx context.Context,
) (*UserAutomationOutput, error) {
	var out UserAutomationOutput
	var errs []string

	out.CronJobs = []CronJob{}
	out.SystemdTimers = []SystemdTimer{}

	crontab, err := execOutput(ctx, "crontab", "-l")
	if err == nil {
		for line := range strings.SplitSeq(crontab, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) < 6 {
				continue
			}
			out.CronJobs = append(out.CronJobs, CronJob{
				Schedule: strings.Join(parts[:5], " "),
				Command:  strings.Join(parts[5:], " "),
			})
		}
	} else {
		appendErr(&errs, "crontab",
			fmt.Errorf("crontab -l: %w", err))
	}

	timers, err := execOutput(ctx, "systemctl", "--user",
		"list-timers", "--output=json",
	)
	if err == nil {
		var rawTimers []struct {
			Unit      string `json:"unit"`
			Next      string `json:"next"`
			Left      string `json:"left"`
			Last      string `json:"last"`
			Passed    string `json:"passed"`
			Activates string `json:"activates"`
		}
		if err := json.Unmarshal([]byte(timers), &rawTimers); err == nil {
			for _, t := range rawTimers {
				out.SystemdTimers = append(
					out.SystemdTimers, SystemdTimer{
						Unit:      t.Unit,
						Activates: t.Activates,
						Next:      t.Next,
						Last:      t.Last,
					},
				)
			}
		} else {
			appendErr(&errs, "systemd-timers",
				fmt.Errorf("parse json: %w", err))
		}
	} else {
		appendErr(&errs, "systemd-timers",
			fmt.Errorf("systemctl --user list-timers: %w", err))
	}

	out.Errors = errs
	return &out, out.Err()
}

func HandleGetUserAutomation(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *UserAutomationOutput, error) {
	return handleToolCall(
		ctx,
		"get_user_automation",
		0,
		GatherUserAutomation,
	)
}

type DesktopSessionOutput struct {
	SessionType    string `json:"session_type"`
	CurrentDesktop string `json:"current_desktop"`
	RuntimeDir     string `json:"runtime_dir"`
	Display        string `json:"display"`
	WaylandDisplay string `json:"wayland_display"`
	OutputErrors
}

func GatherDesktopSessionInfo(
	ctx context.Context,
) (*DesktopSessionOutput, error) {
	return &DesktopSessionOutput{
		SessionType:    os.Getenv("XDG_SESSION_TYPE"),
		CurrentDesktop: os.Getenv("XDG_CURRENT_DESKTOP"),
		RuntimeDir:     os.Getenv("XDG_RUNTIME_DIR"),
		Display:        os.Getenv("DISPLAY"),
		WaylandDisplay: os.Getenv("WAYLAND_DISPLAY"),
	}, nil
}

func HandleGetDesktopSessionInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *DesktopSessionOutput, error) {
	return handleToolCall(
		ctx,
		"get_desktop_session_info",
		0,
		GatherDesktopSessionInfo,
	)
}
