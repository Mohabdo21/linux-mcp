package tools

import (
	"context"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/mem"
)

type GetMemoryInfoInput struct{}

type MemoryInfoOutput struct {
	Total           uint64  `json:"total"`
	Used            uint64  `json:"used"`
	Free            uint64  `json:"free"`
	UsedPercent     float64 `json:"used_percent"`
	SwapTotal       uint64  `json:"swap_total"`
	SwapUsed        uint64  `json:"swap_used"`
	SwapFree        uint64  `json:"swap_free"`
	SwapUsedPercent float64 `json:"swap_used_percent"`
	OutputErrors
}

func GatherMemoryInfo(ctx context.Context) (*MemoryInfoOutput, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	s, err := mem.SwapMemory()
	if err != nil {
		return nil, err
	}
	return &MemoryInfoOutput{
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
	_ *mcp.CallToolRequest,
	_ GetMemoryInfoInput,
) (*mcp.CallToolResult, *MemoryInfoOutput, error) {
	return handleToolCall(
		ctx,
		"get_memory_info",
		5*time.Second,
		GatherMemoryInfo,
	)
}
