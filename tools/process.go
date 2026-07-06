package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/process"
)

type GetProcessInfoInput struct {
	SortBy string `json:"sort_by" jsonschema:"sort by 'cpu' or 'memory' (default: cpu)"`
	Limit  int    `json:"limit"   jsonschema:"max results (default: 10, max: 100)"`
}

type ProcessStat struct {
	PID           int32   `json:"pid"`
	Name          string  `json:"name"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float32 `json:"memory_percent"`
	Status        string  `json:"status"`
}

type ProcessInfoOutput struct {
	Processes []ProcessStat `json:"processes"`
	OutputErrors
}

func GatherProcessInfo(
	ctx context.Context,
	sortBy string,
	limit int,
) (*ProcessInfoOutput, error) {
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
		return nil, err
	}

	result := make([]ProcessStat, 0)
	for _, p := range procs {
		name, _ := p.Name()
		cpu, _ := p.CPUPercent()
		mem, _ := p.MemoryPercent()
		status, _ := p.Status()
		statusStr := strings.Join(status, ",")
		result = append(result, ProcessStat{
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

	return &ProcessInfoOutput{Processes: result}, nil
}

func HandleGetProcessInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetProcessInfoInput,
) (*mcp.CallToolResult, *ProcessInfoOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetProcessInfo,
		0,
		func(ctx context.Context) (*ProcessInfoOutput, error) {
			return GatherProcessInfo(ctx, input.SortBy, input.Limit)
		},
	)
}

type GetTopIOProcessesInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"max results (default: 10, max: 50)"`
}

type IOProcessStat struct {
	Time    string  `json:"time"`
	PID     int     `json:"pid"`
	KbRdS   float64 `json:"kb_rd_s"`
	KbWrS   float64 `json:"kb_wr_s"`
	Command string  `json:"command"`
}

type TopIOProcessesOutput struct {
	Processes []IOProcessStat `json:"processes"`
	OutputErrors
}

func GatherTopIOProcesses(
	ctx context.Context,
	limit int,
) (*TopIOProcessesOutput, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}
	lines, err := execLines(ctx, "pidstat", "-d", "1", "1")
	if err != nil {
		return nil, err
	}
	procs := make([]IOProcessStat, 0)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[0] == "Linux" || fields[0] == "#" {
			continue
		}
		pid, _ := strconv.Atoi(fields[1])
		rdS, _ := strconv.ParseFloat(fields[2], 64)
		wrS, _ := strconv.ParseFloat(fields[3], 64)
		procs = append(procs, IOProcessStat{
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
	return &TopIOProcessesOutput{Processes: procs}, nil
}

func classifyFD(target string) string {
	switch {
	case strings.HasPrefix(target, "socket:"):
		return "socket"
	case strings.HasPrefix(target, "pipe:"):
		return "pipe"
	case strings.HasPrefix(target, "anon_inode:"):
		return "anon_inode"
	default:
		return "file"
	}
}

type GetProcessFDsInput struct {
	PID int32 `json:"pid" jsonschema:"process ID to list open file descriptors for"`
}

type ProcessFD struct {
	FD     uint64 `json:"fd"`
	Type   string `json:"type"`
	Target string `json:"target"`
}

type ProcessFDsOutput struct {
	PID   int         `json:"pid"`
	Name  string      `json:"name"`
	Count int         `json:"fd_count"`
	FDs   []ProcessFD `json:"file_descriptors"`
	OutputErrors
}

func GatherProcessFDs(
	ctx context.Context,
	pid int32,
) (*ProcessFDsOutput, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("process %d: %w", pid, err)
	}

	name, _ := p.Name()

	openFiles, _ := p.OpenFiles()

	fdMap := make(map[uint64]ProcessFD)
	for _, f := range openFiles {
		fdMap[f.Fd] = ProcessFD{
			FD:     f.Fd,
			Type:   "file",
			Target: f.Path,
		}
	}

	fdDir := fmt.Sprintf("/proc/%d/fd", pid)
	entries, readErr := os.ReadDir(fdDir)
	if readErr == nil {
		for _, entry := range entries {
			fdNum, parseErr := strconv.ParseUint(entry.Name(), 10, 64)
			if parseErr != nil {
				continue
			}
			if _, exists := fdMap[fdNum]; exists {
				continue
			}
			linkPath := filepath.Join(fdDir, entry.Name())
			target, linkErr := os.Readlink(linkPath)
			if linkErr != nil {
				continue
			}
			fdMap[fdNum] = ProcessFD{
				FD:     fdNum,
				Type:   classifyFD(target),
				Target: target,
			}
		}
	}

	result := make([]ProcessFD, 0, len(fdMap))
	for _, fd := range fdMap {
		result = append(result, fd)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].FD < result[j].FD
	})

	return &ProcessFDsOutput{
		PID:   int(pid),
		Name:  name,
		Count: len(result),
		FDs:   result,
	}, nil
}

func HandleGetProcessFDs(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetProcessFDsInput,
) (*mcp.CallToolResult, *ProcessFDsOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetProcessFDs,
		0,
		func(ctx context.Context) (*ProcessFDsOutput, error) {
			return GatherProcessFDs(ctx, input.PID)
		},
	)
}

func HandleGetTopIOProcesses(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetTopIOProcessesInput,
) (*mcp.CallToolResult, *TopIOProcessesOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetTopIOProcesses,
		0,
		func(ctx context.Context) (*TopIOProcessesOutput, error) {
			return GatherTopIOProcesses(ctx, input.Limit)
		},
	)
}
