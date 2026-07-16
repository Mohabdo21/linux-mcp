package tools

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetSMARTHealthInput struct {
	Device string `json:"device,omitempty" jsonschema:"optional device name (e.g. sda, nvme0n1). If empty, checks all devices"`
}

type SMARTDeviceHealth struct {
	Device       string            `json:"device"`
	Model        string            `json:"model,omitempty"`
	Serial       string            `json:"serial,omitempty"`
	HealthStatus string            `json:"health_status"`
	Temperature  int               `json:"temperature,omitempty"`
	PowerOnHours int               `json:"power_on_hours,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	RawOutput    string            `json:"raw_output,omitempty"`
}

type SMARTHealthOutput struct {
	Devices []SMARTDeviceHealth `json:"devices"`
	OutputErrors
}

func discoverBlockDevices() []string {
	entries, err := os.ReadDir("/sys/block")
	if err != nil {
		return nil
	}
	var devs []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "loop") ||
			strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "zram") {
			continue
		}
		devs = append(devs, name)
	}
	return devs
}

func parseSMARTHealth(output string) string {
	upper := strings.ToUpper(output)
	if strings.Contains(upper, "PASSED") || strings.Contains(upper, "OK") {
		return "PASSED"
	}
	if strings.Contains(upper, "FAILED") {
		return "FAILED"
	}
	return "unknown"
}

func parseSMARTAttributes(output string) map[string]string {
	attrs := make(map[string]string)
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) < 10 {
			continue
		}
		attrID, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		name := fields[1]
		raw := fields[len(fields)-1]
		switch attrID {
		case 194:
			attrs["Temperature"] = raw
		case 5:
			attrs["Reallocated_Sector_Ct"] = raw
		case 197:
			attrs["Current_Pending_Sector"] = raw
		case 198:
			attrs["Offline_Uncorrectable"] = raw
		case 9:
			attrs["Power_On_Hours"] = raw
		case 177:
			attrs["Wear_Leveling_Count"] = raw
		case 12:
			attrs["Percentage_Used"] = raw
		default:
			attrs[name] = raw
		}
	}
	return attrs
}

func gatherSMARTDevice(ctx context.Context, device string) SMARTDeviceHealth {
	dev := SMARTDeviceHealth{Device: device}

	devPath := "/dev/" + device

	healthOut, err := execOutput(ctx, "smartctl", "-H", devPath)
	if err != nil {
		dev.HealthStatus = "unknown"
		dev.RawOutput = fmt.Sprintf("smartctl -H failed: %v", err)
		return dev
	}
	dev.HealthStatus = parseSMARTHealth(healthOut)
	dev.RawOutput = healthOut

	attrOut, err := execOutput(ctx, "smartctl", "-A", devPath)
	if err == nil {
		dev.Attributes = parseSMARTAttributes(attrOut)
	}

	if val, ok := dev.Attributes["Temperature"]; ok {
		if t, err := strconv.Atoi(val); err == nil {
			dev.Temperature = t
		}
	}
	if val, ok := dev.Attributes["Power_On_Hours"]; ok {
		if h, err := strconv.Atoi(val); err == nil {
			dev.PowerOnHours = h
		}
	}

	if strings.HasPrefix(device, "nvme") {
		infoOut, err := execOutput(ctx, "smartctl", "-i", devPath)
		if err == nil {
			for line := range strings.SplitSeq(infoOut, "\n") {
				line = strings.TrimSpace(line)
				if before, after, ok := strings.Cut(line, ":"); ok {
					key := strings.TrimSpace(before)
					val := strings.TrimSpace(after)
					switch key {
					case "Model Number":
						dev.Model = val
					case "Serial Number":
						dev.Serial = val
					}
				}
			}
		}
	} else {
		infoOut, err := execOutput(ctx, "smartctl", "-i", devPath)
		if err == nil {
			for line := range strings.SplitSeq(infoOut, "\n") {
				line = strings.TrimSpace(line)
				if before, after, ok := strings.Cut(line, ":"); ok {
					key := strings.TrimSpace(before)
					val := strings.TrimSpace(after)
					switch key {
					case "Device Model":
						dev.Model = val
					case "Serial Number":
						dev.Serial = val
					}
				}
			}
		}
	}

	return dev
}

func GatherSMARTHealth(
	ctx context.Context,
	device string,
) (*SMARTHealthOutput, error) {
	_, err := execOutput(ctx, "smartctl", "--version")
	if err != nil {
		return nil, fmt.Errorf("smartctl not found or not executable")
	}

	var devices []string
	if device != "" {
		devices = []string{device}
	} else {
		devices = discoverBlockDevices()
	}

	var out SMARTHealthOutput
	for _, d := range devices {
		select {
		case <-ctx.Done():
			return &out, ctx.Err()
		default:
		}
		dev := gatherSMARTDevice(ctx, d)
		out.Devices = append(out.Devices, dev)
	}

	return &out, nil
}

func HandleGetSMARTHealth(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetSMARTHealthInput,
) (*mcp.CallToolResult, *SMARTHealthOutput, error) {
	device := strings.TrimSpace(input.Device)
	return handleToolCall(
		ctx,
		config.ToolNameGetSMARTHealth,
		0,
		func(ctx context.Context) (*SMARTHealthOutput, error) {
			return GatherSMARTHealth(ctx, device)
		},
	)
}

type DiskIOMetric struct {
	Device          string `json:"device"`
	ReadsCompleted  uint64 `json:"reads_completed"`
	SectorsRead     uint64 `json:"sectors_read"`
	WritesCompleted uint64 `json:"writes_completed"`
	SectorsWritten  uint64 `json:"sectors_written"`
	IOsInProgress   uint64 `json:"ios_in_progress"`
	ReadTimeMs      uint64 `json:"read_time_ms"`
	WriteTimeMs     uint64 `json:"write_time_ms"`
}

type DiskIOMetricsOutput struct {
	Metrics []DiskIOMetric `json:"metrics"`
	OutputErrors
}

func GatherDiskIOMetrics(
	ctx context.Context,
) (*DiskIOMetricsOutput, error) {
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return nil, err
	}

	var out DiskIOMetricsOutput
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 14 {
			continue
		}
		name := fields[2]
		if strings.HasPrefix(name, "loop") ||
			strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "zram") {
			continue
		}
		m := DiskIOMetric{Device: name}
		m.ReadsCompleted, _ = strconv.ParseUint(fields[3], 10, 64)
		m.SectorsRead, _ = strconv.ParseUint(fields[5], 10, 64)
		m.WritesCompleted, _ = strconv.ParseUint(fields[7], 10, 64)
		m.SectorsWritten, _ = strconv.ParseUint(fields[9], 10, 64)
		m.IOsInProgress, _ = strconv.ParseUint(fields[11], 10, 64)
		m.ReadTimeMs, _ = strconv.ParseUint(fields[6], 10, 64)
		m.WriteTimeMs, _ = strconv.ParseUint(fields[10], 10, 64)
		out.Metrics = append(out.Metrics, m)
	}

	return &out, nil
}

func HandleGetDiskIOMetrics(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *DiskIOMetricsOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetDiskIOMetrics,
		0,
		GatherDiskIOMetrics,
	)
}
