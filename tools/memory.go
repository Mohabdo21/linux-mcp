package tools

import (
	"context"
	"errors"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/mem"
)

type GetMemoryInfoInput struct{}

type MemoryInfoOutput struct {
	Total           uint64   `json:"total"`
	Used            uint64   `json:"used"`
	Free            uint64   `json:"free"`
	UsedPercent     float64  `json:"used_percent"`
	SwapTotal       uint64   `json:"swap_total"`
	SwapUsed        uint64   `json:"swap_used"`
	SwapFree        uint64   `json:"swap_free"`
	SwapUsedPercent float64  `json:"swap_used_percent"`
	Errors          []string `json:"errors,omitempty"`
}

func GatherMemoryInfo(ctx context.Context) (MemoryInfoOutput, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return MemoryInfoOutput{}, err
	}
	s, err := mem.SwapMemory()
	if err != nil {
		return MemoryInfoOutput{}, err
	}
	return MemoryInfoOutput{
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

func HandleGetMemoryInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetMemoryInfoInput,
) (*mcp.CallToolResult, MemoryInfoOutput, error) {
	if config.IsDisabled("get_memory_info") {
		return nil, MemoryInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_memory_info", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherMemoryInfo(ctx)
	LogToolCall(ctx, "get_memory_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
