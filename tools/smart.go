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

type IOStatDevice struct {
	Device     string  `json:"device"`
	ReadKBs    float64 `json:"read_kBs"`
	WriteKBs   float64 `json:"write_kBs"`
	DiscardKBs float64 `json:"discard_kBs,omitempty"`
	ReadsPerS  float64 `json:"reads_per_s"`
	WritesPerS float64 `json:"writes_per_s"`
	AVGWaitMs  float64 `json:"avg_wait_ms"`
	AVGReadMs  float64 `json:"avg_read_ms"`
	AVGWriteMs float64 `json:"avg_write_ms"`
	QueueSize  float64 `json:"queue_size,omitempty"`
	UtilPct    float64 `json:"util_pct,omitempty"`
}

type IOStatsOutput struct {
	Devices []IOStatDevice `json:"devices"`
	OutputErrors
}

func GatherIOStats(
	ctx context.Context,
) (*IOStatsOutput, error) {
	lines, err := execLines(ctx, "iostat", "-xd", "1", "1")
	if err != nil {
		return nil, fmt.Errorf("iostat not found or failed: %w", err)
	}

	var devices []IOStatDevice
	headerFound := false
	for _, line := range lines {
		// Skip until we find the device header line.
		if strings.Contains(line, "Device") {
			headerFound = true
			continue
		}
		if !headerFound {
			continue
		}

		fields := strings.Fields(line)
		// Extended stats: device, r_await, w_await, rareq-sz, wareq-sz,
		// aqu-sz, %util, rkB/s, wkB/s, rrqm/s, wrqm/s, r/s, w/s
		// iostat -xd output columns (varies by version, but typical):
		// Device  r/s  rkB/s  rrqm/s  %rrqm  r_await  rareq-sz  ...  w/s  wkB/s  wrqm/s  %wrqm  w_await  wareq-sz  ...  d/s  dkB/s  drqm/s  %drqm  d_await  dareq-sz  ...  f/s  f_await  aqu-sz  %util
		if len(fields) < 4 {
			continue
		}
		name := fields[0]
		if strings.HasPrefix(name, "loop") ||
			strings.HasPrefix(name, "ram") ||
			strings.HasPrefix(name, "zram") {
			continue
		}

		dev := IOStatDevice{Device: name}

		// Parse known fields by searching for the key columns.
		// iostat -xd has many columns; we parse what's available.
		// Column positions depend on the iostat version, so we use
		// a robust approach: find known column headers or parse by position.
		// Typical columns after device name:
		// r/s rkB/s rrqm/s %rrm r_await rareq-sz w/s wkB/s wrqm/s %wrm w_await wareq-sz d/s dkB/s drqm/s %drm d_await dareq-sz f/s f_await aqu-sz %util
		//
		// We handle both 14-column (older) and 20+ column (newer) layouts.
		n := len(fields) - 1 // number of numeric fields after device name

		switch {
		case n >= 19:
			// Newer iostat: device r/s rkB/s rrqm/s %rrqm r_await rareq-sz w/s wkB/s wrqm/s %wrqm w_await wareq-sz d/s dkB/s drqm/s %drqm d_await dareq-sz f/s f_await aqu-sz %util
			dev.ReadsPerS, _ = strconv.ParseFloat(fields[1], 64)
			dev.ReadKBs, _ = strconv.ParseFloat(fields[2], 64)
			dev.AVGReadMs, _ = strconv.ParseFloat(fields[5], 64)
			dev.WritesPerS, _ = strconv.ParseFloat(fields[7], 64)
			dev.WriteKBs, _ = strconv.ParseFloat(fields[8], 64)
			dev.AVGWriteMs, _ = strconv.ParseFloat(fields[11], 64)
			dev.DiscardKBs, _ = strconv.ParseFloat(fields[14], 64)
			dev.AVGWaitMs, _ = strconv.ParseFloat(fields[17], 64)
			dev.QueueSize, _ = strconv.ParseFloat(fields[20], 64)
			dev.UtilPct, _ = strconv.ParseFloat(fields[21], 64)
		case n >= 14:
			// Older iostat -xd: device rrqm/s wrqm/s r/s w/s rkB/s wkB/s avgrq-sz avgqu-sz await r_await w_await svctm %util
			dev.ReadsPerS, _ = strconv.ParseFloat(fields[3], 64)
			dev.ReadKBs, _ = strconv.ParseFloat(fields[5], 64)
			dev.WritesPerS, _ = strconv.ParseFloat(fields[4], 64)
			dev.WriteKBs, _ = strconv.ParseFloat(fields[6], 64)
			dev.QueueSize, _ = strconv.ParseFloat(fields[8], 64)
			dev.AVGWaitMs, _ = strconv.ParseFloat(fields[9], 64)
			dev.AVGReadMs, _ = strconv.ParseFloat(fields[10], 64)
			dev.AVGWriteMs, _ = strconv.ParseFloat(fields[11], 64)
			dev.UtilPct, _ = strconv.ParseFloat(fields[13], 64)
		default:
			continue
		}

		devices = append(devices, dev)
	}

	return &IOStatsOutput{Devices: nilToEmpty(devices)}, nil
}

func HandleGetIOStats(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *IOStatsOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetIOStats,
		0,
		GatherIOStats,
	)
}
