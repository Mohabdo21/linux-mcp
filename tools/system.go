package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	Errors    []string              `json:"errors,omitempty"`
}

func GatherEnvironmentVariables(
	ctx context.Context,
	search string,
) (EnvironmentVariablesOutput, error) {
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
	return EnvironmentVariablesOutput{
		Variables: variables,
		Count:     len(variables),
	}, nil
}

func HandleGetEnvironmentVariables(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetEnvironmentVariablesInput,
) (*mcp.CallToolResult, EnvironmentVariablesOutput, error) {
	if config.IsDisabled("get_environment_variables") {
		return nil, EnvironmentVariablesOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_environment_variables", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherEnvironmentVariables(ctx, input.Search)
	LogToolCall(ctx, "get_environment_variables",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
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
	Errors     []string    `json:"errors,omitempty"`
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

func readSysfsFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
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
	cmd := exec.CommandContext(ctx, "lspci")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var devices []BusDevice
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
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
	cmd := exec.CommandContext(ctx, "lsusb")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var devices []BusDevice
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
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
) (HardwareBusInfoOutput, error) {
	var out HardwareBusInfoOutput
	var errs ErrList

	pci, err := parseLspciOutput(ctx)
	if err != nil {
		pci, err = parsePCIDevicesSysfs(ctx)
		if err != nil {
			errs.Add("pci", err)
		}
	}
	if pci != nil {
		out.PCIDevices = filterDevices(pci, search)
	}

	usb, err := parseLsusbOutput(ctx)
	if err != nil {
		usb, err = parseUSBDevicesSysfs(ctx)
		if err != nil {
			errs.Add("usb", err)
		}
	}
	if usb != nil {
		out.USBDevices = filterDevices(usb, search)
	}

	out.Errors = errs
	return out, errs.Err()
}

func HandleGetHardwareBusInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetHardwareBusInfoInput,
) (*mcp.CallToolResult, HardwareBusInfoOutput, error) {
	if config.IsDisabled("get_hardware_bus_info") {
		return nil, HardwareBusInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_hardware_bus_info", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherHardwareBusInfo(ctx, input.Search)
	LogToolCall(ctx, "get_hardware_bus_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
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
