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

type ShmSegment struct {
	Key       int64  `json:"key"`
	ID        int64  `json:"id"`
	Owner     string `json:"owner"`
	Bytes     int64  `json:"bytes"`
	Nattch    int32  `json:"nattch"`
	Status    string `json:"status"`
	CPID      int32  `json:"cpid"`
	LPID      int32  `json:"lpid"`
	AttachAt  string `json:"attach_at"`
	DetachAt  string `json:"detach_at"`
	CreatTime string `json:"creat_time"`
}

type SharedMemoryOutput struct {
	Segments []ShmSegment `json:"segments"`
	OutputErrors
}

type GetSharedMemorySegmentsInput struct{}

func GatherSharedMemorySegments() (*SharedMemoryOutput, error) {
	out := &SharedMemoryOutput{Segments: make([]ShmSegment, 0)}

	data, err := os.ReadFile("/proc/sysvipc/shm")
	if err != nil {
		return out, nil
	}

	lines := strings.Split(string(data), "\n")
	if len(lines) < 2 {
		return out, nil
	}
	// Skip header line
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		seg, parseErr := parseShmLine(line)
		if parseErr != nil {
			out.Add("parse", parseErr)
			continue
		}
		out.Segments = append(out.Segments, seg)
	}

	return out, nil
}

func parseShmLine(line string) (ShmSegment, error) {
	// Format: key shmid perms size cpid lpid nattch status uid gid cuid cgid atime dtime ctime
	fields := strings.Fields(line)
	if len(fields) < 14 {
		return ShmSegment{}, fmt.Errorf(
			"insufficient fields in shm line: %s",
			line,
		)
	}

	key, _ := strconv.ParseInt(fields[0], 10, 64)
	id, _ := strconv.ParseInt(fields[1], 10, 64)
	size, _ := strconv.ParseInt(fields[3], 10, 64)
	cpid, _ := strconv.ParseInt(fields[4], 10, 32)
	lpid, _ := strconv.ParseInt(fields[5], 10, 32)
	nattch, _ := strconv.ParseInt(fields[6], 10, 32)

	return ShmSegment{
		Key:       key,
		ID:        id,
		Bytes:     size,
		Nattch:    int32(nattch),
		Status:    fields[7],
		CPID:      int32(cpid),
		LPID:      int32(lpid),
		AttachAt:  fields[11],
		DetachAt:  fields[12],
		CreatTime: fields[13],
	}, nil
}

func HandleGetSharedMemorySegments(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetSharedMemorySegmentsInput,
) (*mcp.CallToolResult, *SharedMemoryOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetSharedMemorySegments,
		0,
		func(ctx context.Context) (*SharedMemoryOutput, error) {
			return GatherSharedMemorySegments()
		},
	)
}
