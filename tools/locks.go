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

type FileLock struct {
	LockType string `json:"lock_type"`
	Access   string `json:"access"`
	PID      int32  `json:"pid"`
	Start    int64  `json:"start"`
	End      int64  `json:"end"`
	Path     string `json:"path"`
}

type FileLocksOutput struct {
	Locks []FileLock `json:"locks"`
	OutputErrors
}

type GetFileLocksInput struct{}

func GatherFileLocks() (*FileLocksOutput, error) {
	out := &FileLocksOutput{Locks: make([]FileLock, 0)}

	data, err := os.ReadFile("/proc/locks")
	if err != nil {
		return out, nil
	}

	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lock, parseErr := parseLockLine(line)
		if parseErr != nil {
			out.Add("parse", parseErr)
			continue
		}
		out.Locks = append(out.Locks, lock)
	}

	return out, nil
}

func parseLockLine(line string) (FileLock, error) {
	// Format: ID TYPE ACCESS RW PID MAJOR:MINOR INODE START [END] [PATH]
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return FileLock{}, fmt.Errorf(
			"insufficient fields in lock line: %s",
			line,
		)
	}

	pid, err := strconv.ParseInt(fields[4], 10, 32)
	if err != nil {
		return FileLock{}, fmt.Errorf("invalid PID %s: %w", fields[4], err)
	}

	start := int64(-1)
	end := int64(-1)
	path := ""
	if fields[7] != "EOF" {
		start, err = strconv.ParseInt(fields[7], 10, 64)
		if err != nil {
			return FileLock{}, fmt.Errorf(
				"invalid start %s: %w",
				fields[7],
				err,
			)
		}
	}
	if len(fields) >= 9 && fields[7] != "EOF" {
		end, err = strconv.ParseInt(fields[8], 10, 64)
		if err == nil {
			if len(fields) >= 10 {
				path = strings.Join(fields[9:], " ")
			}
		} else {
			end = -1
			path = strings.Join(fields[8:], " ")
		}
	}

	return FileLock{
		LockType: fields[1],
		Access:   fields[2] + " " + fields[3],
		PID:      int32(pid),
		Start:    start,
		End:      end,
		Path:     path,
	}, nil
}

func HandleGetFileLocks(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetFileLocksInput,
) (*mcp.CallToolResult, *FileLocksOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetFileLocks,
		0,
		func(ctx context.Context) (*FileLocksOutput, error) {
			return GatherFileLocks()
		},
	)
}
