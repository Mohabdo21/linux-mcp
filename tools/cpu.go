package tools

import (
	"context"
	"time"

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
	OutputErrors
}

func GatherCPUInfo(ctx context.Context) (*CPUInfoOutput, error) {
	info, err := cpu.Info()
	if err != nil {
		return nil, err
	}
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return nil, err
	}
	physCount, err := cpu.Counts(true)
	if err != nil {
		return nil, err
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
	return &CPUInfoOutput{
		UsagePercent:      usage,
		PhysicalCoreCount: int32(physCount),
		Cores:             cores,
	}, nil
}

func HandleGetCPUInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ GetCPUInfoInput,
) (*mcp.CallToolResult, *CPUInfoOutput, error) {
	return handleToolCall(ctx, "get_cpu_info", 5*time.Second, GatherCPUInfo)
}

type GetCPUTemperatureInput struct{}

type TemperatureStat struct {
	SensorKey   string  `json:"sensor_key"`
	Temperature float64 `json:"temperature_celsius"`
}

type CPUTemperatureOutput struct {
	Temperatures []TemperatureStat `json:"temperatures"`
	Message      string            `json:"message,omitempty"`
	OutputErrors
}

func GatherCPUTemperature(ctx context.Context) (*CPUTemperatureOutput, error) {
	temps, err := sensors.SensorsTemperatures()
	if len(temps) == 0 {
		msg := "No temperature sensors available"
		if err != nil {
			msg = err.Error()
		}
		return &CPUTemperatureOutput{Message: msg}, err
	}
	var result []TemperatureStat
	for _, t := range temps {
		result = append(result, TemperatureStat{
			SensorKey:   t.SensorKey,
			Temperature: t.Temperature,
		})
	}
	return &CPUTemperatureOutput{Temperatures: result}, err
}

func HandleGetCPUTemperature(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ GetCPUTemperatureInput,
) (*mcp.CallToolResult, *CPUTemperatureOutput, error) {
	return handleToolCall(
		ctx,
		"get_cpu_temperature",
		5*time.Second,
		GatherCPUTemperature,
	)
}
