package tools

import (
	"context"
	"errors"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

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
	Errors    []string      `json:"errors,omitempty"`
}

func GatherProcessInfo(
	ctx context.Context,
	sortBy string,
	limit int,
) (ProcessInfoOutput, error) {
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
		return ProcessInfoOutput{}, err
	}

	var result []ProcessStat
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

	return ProcessInfoOutput{Processes: result}, nil
}

func HandleGetProcessInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetProcessInfoInput,
) (*mcp.CallToolResult, ProcessInfoOutput, error) {
	if config.IsDisabled("get_process_info") {
		return nil, ProcessInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_process_info", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherProcessInfo(ctx, input.SortBy, input.Limit)
	LogToolCall(ctx, "get_process_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
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
	Errors    []string        `json:"errors,omitempty"`
}

func GatherTopIOProcesses(
	ctx context.Context,
	limit int,
) (TopIOProcessesOutput, error) {
	if limit <= 0 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}
	cmd := exec.CommandContext(ctx, "pidstat", "-d", "1", "1")
	out, err := cmd.Output()
	if err != nil {
		return TopIOProcessesOutput{}, err
	}
	var procs []IOProcessStat
	for line := range strings.SplitSeq(
		strings.TrimSpace(string(out)), "\n",
	) {
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
	return TopIOProcessesOutput{Processes: procs}, nil
}

func HandleGetTopIOProcesses(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetTopIOProcessesInput,
) (*mcp.CallToolResult, TopIOProcessesOutput, error) {
	if config.IsDisabled("get_top_io_processes") {
		return nil, TopIOProcessesOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_top_io_processes", 15*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherTopIOProcesses(ctx, input.Limit)
	LogToolCall(ctx, "get_top_io_processes",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
