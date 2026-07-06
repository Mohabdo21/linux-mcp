package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/shirou/gopsutil/v4/disk"
)

type GetDiskInfoInput struct {
	MountPoint string `json:"mount_point" jsonschema:"optional mount point filter"`
}

type DiskUsageStat struct {
	MountPoint  string  `json:"mount_point"`
	Filesystem  string  `json:"filesystem"`
	Device      string  `json:"device"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type DiskInfoOutput struct {
	Partitions []DiskUsageStat `json:"partitions"`
	OutputErrors
}

func GatherDiskInfo(
	ctx context.Context,
	mountPoint string,
) (*DiskInfoOutput, error) {
	partitions, err := disk.Partitions(false)
	if err != nil {
		return nil, err
	}

	result := make([]DiskUsageStat, 0)
	for _, p := range partitions {
		if mountPoint != "" && p.Mountpoint != mountPoint {
			continue
		}
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		result = append(result, DiskUsageStat{
			MountPoint:  p.Mountpoint,
			Filesystem:  p.Fstype,
			Device:      p.Device,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}
	return &DiskInfoOutput{Partitions: result}, nil
}

func HandleGetDiskInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetDiskInfoInput,
) (*mcp.CallToolResult, *DiskInfoOutput, error) {
	return handleToolCall(
		ctx,
		"get_disk_info",
		0,
		func(ctx context.Context) (*DiskInfoOutput, error) {
			return GatherDiskInfo(ctx, input.MountPoint)
		},
	)
}

type GetInodeUsageInput struct {
	MountPoint string `json:"mount_point,omitempty" jsonschema:"optional mount point filter"`
}

type InodeUsageStat struct {
	Filesystem  string `json:"filesystem"`
	Inodes      uint64 `json:"inodes"`
	IUsed       uint64 `json:"iused"`
	IFree       uint64 `json:"ifree"`
	IUsePercent string `json:"iuse_percent"`
	MountedOn   string `json:"mounted_on"`
}

type InodeUsageOutput struct {
	Mounts []InodeUsageStat `json:"mounts"`
	OutputErrors
}

func GatherInodeUsage(
	ctx context.Context,
	mountPoint string,
) (*InodeUsageOutput, error) {
	lines, err := execLines(ctx, "df", "-i")
	if err != nil {
		return nil, err
	}
	mounts := make([]InodeUsageStat, 0)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 6 || fields[0] == "Filesystem" {
			continue
		}
		inodes, _ := strconv.ParseUint(fields[1], 10, 64)
		iused, _ := strconv.ParseUint(fields[2], 10, 64)
		ifree, _ := strconv.ParseUint(fields[3], 10, 64)
		mounted := fields[5]

		if mountPoint != "" && mounted != mountPoint {
			continue
		}
		mounts = append(mounts, InodeUsageStat{
			Filesystem:  fields[0],
			Inodes:      inodes,
			IUsed:       iused,
			IFree:       ifree,
			IUsePercent: fields[4],
			MountedOn:   mounted,
		})
	}
	return &InodeUsageOutput{Mounts: mounts}, nil
}

func HandleGetInodeUsage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetInodeUsageInput,
) (*mcp.CallToolResult, *InodeUsageOutput, error) {
	return handleToolCall(
		ctx,
		"get_inode_usage",
		0,
		func(ctx context.Context) (*InodeUsageOutput, error) {
			return GatherInodeUsage(ctx, input.MountPoint)
		},
	)
}

type GetLargestFilesInput struct {
	Path  string `json:"path,omitempty"  jsonschema:"directory to scan (default: current dir)"`
	Limit int    `json:"limit,omitempty" jsonschema:"max results (default: 10, max: 100)"`
}

type LargestFileEntry struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	SizeHuman string `json:"size_human"`
	IsDir     bool   `json:"is_dir"`
}

type LargestFilesOutput struct {
	Path    string             `json:"path"`
	Entries []LargestFileEntry `json:"entries"`
	OutputErrors
}

func GatherLargestFiles(
	ctx context.Context,
	path string,
	limit int,
) (*LargestFilesOutput, error) {
	if path == "" {
		path = "."
	}
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf(
		"du -sb %s/* %s/.[!.]* 2>/dev/null | sort -rn",
		ShellQuote(path), ShellQuote(path),
	))
	out, err := cmd.Output()
	if err != nil {
		return &LargestFilesOutput{
			Path:    path,
			Entries: []LargestFileEntry{},
		}, nil
	}
	entries := make([]LargestFileEntry, 0, limit)
	for line := range strings.SplitSeq(
		strings.TrimSpace(string(out)), "\n",
	) {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		size, _ := strconv.ParseInt(fields[0], 10, 64)
		name := strings.Join(fields[1:], " ")
		isDir := false
		if info, err := os.Stat(name); err == nil {
			isDir = info.IsDir()
		}
		entries = append(entries, LargestFileEntry{
			Name:      name,
			SizeBytes: size,
			SizeHuman: HumanSize(size),
			IsDir:     isDir,
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SizeBytes > entries[j].SizeBytes
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	return &LargestFilesOutput{Path: path, Entries: entries}, nil
}

func HandleGetLargestFiles(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetLargestFilesInput,
) (*mcp.CallToolResult, *LargestFilesOutput, error) {
	return handleToolCall(
		ctx,
		"get_largest_files",
		0,
		func(ctx context.Context) (*LargestFilesOutput, error) {
			return GatherLargestFiles(ctx, input.Path, input.Limit)
		},
	)
}

type GetMountOptionsInput struct {
	MountPoint string `json:"mount_point,omitempty" jsonschema:"optional mount point filter (e.g. '/')"`
}

type MountEntry struct {
	Source  string   `json:"source"`
	Target  string   `json:"target"`
	FSType  string   `json:"fs_type"`
	Options []string `json:"options"`
}

type MountOptionsOutput struct {
	Mounts []MountEntry `json:"mounts"`
	OutputErrors
}

func GatherMountOptions(
	ctx context.Context,
	mountPoint string,
) (*MountOptionsOutput, error) {
	lines, err := execLines(ctx, "findmnt",
		"--noheadings", "-o", "SOURCE,TARGET,FSTYPE,OPTIONS")
	if err != nil {
		return nil, err
	}
	mounts := make([]MountEntry, 0)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if mountPoint != "" && fields[1] != mountPoint {
			continue
		}
		mounts = append(mounts, MountEntry{
			Source:  fields[0],
			Target:  fields[1],
			FSType:  fields[2],
			Options: strings.Split(fields[3], ","),
		})
	}
	return &MountOptionsOutput{Mounts: mounts}, nil
}

func HandleGetMountOptions(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetMountOptionsInput,
) (*mcp.CallToolResult, *MountOptionsOutput, error) {
	return handleToolCall(
		ctx,
		"get_mount_options",
		0,
		func(ctx context.Context) (*MountOptionsOutput, error) {
			return GatherMountOptions(ctx, input.MountPoint)
		},
	)
}
