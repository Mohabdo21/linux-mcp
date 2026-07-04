package tools

import (
	"context"
	"errors"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/sensors"
)

type GetCPUInfoInput struct{}

type CPUDetails struct {
	ModelName string  `json:"model_name"`
	CoreCount int32   `json:"core_count"`
	MHz       float64 `json:"mhz"`
}

type CPUInfoOutput struct {
	UsagePercent      float64      `json:"usage_percent"`
	PhysicalCoreCount int32        `json:"physical_core_count"`
	Cores             []CPUDetails `json:"cores"`
	Errors            []string     `json:"errors,omitempty"`
}

func GatherCPUInfo(ctx context.Context) (CPUInfoOutput, error) {
	info, err := cpu.Info()
	if err != nil {
		return CPUInfoOutput{}, err
	}
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return CPUInfoOutput{}, err
	}
	physCount, err := cpu.Counts(true)
	if err != nil {
		return CPUInfoOutput{}, err
	}
	var cores []CPUDetails
	for _, c := range info {
		cores = append(cores, CPUDetails{
			ModelName: c.ModelName,
			CoreCount: c.Cores,
			MHz:       c.Mhz,
		})
	}
	usage := 0.0
	if len(percent) > 0 {
		usage = percent[0]
	}
	return CPUInfoOutput{
		UsagePercent:      usage,
		PhysicalCoreCount: int32(physCount),
		Cores:             cores,
	}, nil
}

func HandleGetCPUInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetCPUInfoInput,
) (*mcp.CallToolResult, CPUInfoOutput, error) {
	if config.IsDisabled("get_cpu_info") {
		return nil, CPUInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_cpu_info", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherCPUInfo(ctx)
	LogToolCall(ctx, "get_cpu_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}

type GetCPUTemperatureInput struct{}

type TemperatureStat struct {
	SensorKey   string  `json:"sensor_key"`
	Temperature float64 `json:"temperature_celsius"`
}

type CPUTemperatureOutput struct {
	Temperatures []TemperatureStat `json:"temperatures"`
	Message      string            `json:"message,omitempty"`
	Errors       []string          `json:"errors,omitempty"`
}

func GatherCPUTemperature(ctx context.Context) (CPUTemperatureOutput, error) {
	temps, err := sensors.SensorsTemperatures()
	if len(temps) == 0 {
		msg := "No temperature sensors available"
		if err != nil {
			msg = err.Error()
		}
		return CPUTemperatureOutput{Message: msg}, err
	}
	var result []TemperatureStat
	for _, t := range temps {
		result = append(result, TemperatureStat{
			SensorKey:   t.SensorKey,
			Temperature: t.Temperature,
		})
	}
	return CPUTemperatureOutput{Temperatures: result}, err
}

func HandleGetCPUTemperature(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetCPUTemperatureInput,
) (*mcp.CallToolResult, CPUTemperatureOutput, error) {
	if config.IsDisabled("get_cpu_temperature") {
		return nil, CPUTemperatureOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_cpu_temperature", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherCPUTemperature(ctx)
	LogToolCall(ctx, "get_cpu_temperature",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
