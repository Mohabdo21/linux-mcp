package tools

import (
	"context"

	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetProcDiagnosticsInput struct {
	Sections string `json:"sections,omitempty" jsonschema:"comma-separated sections: interrupts,softirqs,vmstat,diskstats,filesystems,version,slabinfo. Empty=all"`
}

type SoftIRQInfo struct {
	Type  string `json:"type"`
	Total uint64 `json:"total"`
}

type VMStatInfo struct {
	Metrics map[string]uint64 `json:"metrics"`
}

type ProcDiskStat struct {
	Device          string `json:"device"`
	ReadsCompleted  uint64 `json:"reads_completed"`
	ReadsMerged     uint64 `json:"reads_merged"`
	SectorsRead     uint64 `json:"sectors_read"`
	ReadMs          uint64 `json:"read_ms"`
	WritesCompleted uint64 `json:"writes_completed"`
	WritesMerged    uint64 `json:"writes_merged"`
	SectorsWritten  uint64 `json:"sectors_written"`
	WriteMs         uint64 `json:"write_ms"`
	IOsInProgress   uint64 `json:"ios_in_progress"`
	IoMs            uint64 `json:"io_ms"`
	WeightedIoMs    uint64 `json:"weighted_io_ms"`
}

type FilesystemInfo struct {
	Name  string `json:"name"`
	Nodev bool   `json:"nodev"`
}

type SlabInfoEntry struct {
	Name       string `json:"name"`
	ActiveObjs uint64 `json:"active_objs"`
	NumObjs    uint64 `json:"num_objs"`
	ObjSize    uint64 `json:"obj_size"`
}

type ProcDiagnosticsOutput struct {
	Interrupts  []string         `json:"interrupts,omitempty"`
	SoftIRQs    []SoftIRQInfo    `json:"softirqs,omitempty"`
	VMStat      *VMStatInfo      `json:"vmstat,omitempty"`
	DiskStats   []ProcDiskStat   `json:"diskstats,omitempty"`
	Filesystems []FilesystemInfo `json:"filesystems,omitempty"`
	Version     string           `json:"version,omitempty"`
	SlabInfo    []SlabInfoEntry  `json:"slabinfo,omitempty"`
	OutputErrors
}

func parseProcInterrupts(content string) []string {
	type irqLine struct {
		line  string
		total uint64
	}
	var lines []irqLine
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "CPU") || strings.HasPrefix(line, "IWI") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		irqName := fields[0]
		if irqName == "" || irqName[len(irqName)-1] != ':' {
			continue
		}
		irqName = irqName[:len(irqName)-1]
		var total uint64
		for _, f := range fields[1 : len(fields)-1] {
			if v, err := strconv.ParseUint(f, 10, 64); err == nil {
				total += v
			}
		}
		devName := fields[len(fields)-1]
		lines = append(
			lines,
			irqLine{line: irqName + " " + devName, total: total},
		)
	}
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].total > lines[j].total
	})
	result := make([]string, 0, min(len(lines), 10))
	for i := 0; i < len(lines) && i < 10; i++ {
		result = append(result, lines[i].line)
	}
	return result
}

func parseProcSoftIRQs(content string) []SoftIRQInfo {
	var result []SoftIRQInfo
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "CPU") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		irqType := strings.TrimSuffix(fields[0], ":")
		var total uint64
		for _, f := range fields[1:] {
			if v, err := strconv.ParseUint(f, 10, 64); err == nil {
				total += v
			}
		}
		result = append(result, SoftIRQInfo{Type: irqType, Total: total})
	}
	return result
}

func parseProcVMStat(content string) map[string]uint64 {
	metrics := make(map[string]uint64)
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 2 {
			if v, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
				metrics[fields[0]] = v
			}
		}
	}
	return metrics
}

func parseProcDiskStats(content string) []ProcDiskStat {
	var result []ProcDiskStat
	for line := range strings.SplitSeq(content, "\n") {
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
		d := ProcDiskStat{Device: name}
		d.ReadsCompleted, _ = strconv.ParseUint(fields[3], 10, 64)
		d.ReadsMerged, _ = strconv.ParseUint(fields[4], 10, 64)
		d.SectorsRead, _ = strconv.ParseUint(fields[5], 10, 64)
		d.ReadMs, _ = strconv.ParseUint(fields[6], 10, 64)
		d.WritesCompleted, _ = strconv.ParseUint(fields[7], 10, 64)
		d.WritesMerged, _ = strconv.ParseUint(fields[8], 10, 64)
		d.SectorsWritten, _ = strconv.ParseUint(fields[9], 10, 64)
		d.WriteMs, _ = strconv.ParseUint(fields[10], 10, 64)
		d.IOsInProgress, _ = strconv.ParseUint(fields[11], 10, 64)
		d.IoMs, _ = strconv.ParseUint(fields[12], 10, 64)
		d.WeightedIoMs, _ = strconv.ParseUint(fields[13], 10, 64)
		result = append(result, d)
	}
	return result
}

func parseProcFilesystems(content string) []FilesystemInfo {
	var result []FilesystemInfo
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		nodev := false
		if strings.HasPrefix(line, "nodev") {
			nodev = true
			line = strings.TrimSpace(strings.TrimPrefix(line, "nodev"))
		}
		if line != "" {
			result = append(result, FilesystemInfo{Name: line, Nodev: nodev})
		}
	}
	return result
}

func parseProcSlabinfo(content string) []SlabInfoEntry {
	type rawEntry struct {
		name       string
		activeObjs uint64
		numObjs    uint64
		objSize    uint64
	}
	var entries []rawEntry
	sawHeader := false
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if !sawHeader {
			sawHeader = true
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		e := rawEntry{name: fields[0]}
		e.activeObjs, _ = strconv.ParseUint(fields[1], 10, 64)
		e.numObjs, _ = strconv.ParseUint(fields[2], 10, 64)
		e.objSize, _ = strconv.ParseUint(fields[3], 10, 64)
		entries = append(entries, e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].activeObjs > entries[j].activeObjs
	})
	result := make([]SlabInfoEntry, 0, min(len(entries), 20))
	for i := 0; i < len(entries) && i < 20; i++ {
		result = append(result, SlabInfoEntry{
			Name:       entries[i].name,
			ActiveObjs: entries[i].activeObjs,
			NumObjs:    entries[i].numObjs,
			ObjSize:    entries[i].objSize,
		})
	}
	return result
}

func readProcFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GatherProcDiagnostics(
	ctx context.Context, sections string,
) (*ProcDiagnosticsOutput, error) {
	out := &ProcDiagnosticsOutput{}
	var errs []string

	wanted := map[string]bool{
		"interrupts":  true,
		"softirqs":    true,
		"vmstat":      true,
		"diskstats":   true,
		"filesystems": true,
		"version":     true,
		"slabinfo":    true,
	}
	if sections != "" {
		wanted = make(map[string]bool)
		for s := range strings.SplitSeq(sections, ",") {
			wanted[strings.TrimSpace(s)] = true
		}
	}

	if wanted["interrupts"] {
		if content, err := readProcFile("/proc/interrupts"); err == nil {
			out.Interrupts = parseProcInterrupts(content)
		} else {
			appendErr(&errs, "interrupts", err)
		}
	}
	if wanted["softirqs"] {
		if content, err := readProcFile("/proc/softirqs"); err == nil {
			out.SoftIRQs = parseProcSoftIRQs(content)
		} else {
			appendErr(&errs, "softirqs", err)
		}
	}
	if wanted["vmstat"] {
		if content, err := readProcFile("/proc/vmstat"); err == nil {
			out.VMStat = &VMStatInfo{Metrics: parseProcVMStat(content)}
		} else {
			appendErr(&errs, "vmstat", err)
		}
	}
	if wanted["diskstats"] {
		if content, err := readProcFile("/proc/diskstats"); err == nil {
			out.DiskStats = parseProcDiskStats(content)
		} else {
			appendErr(&errs, "diskstats", err)
		}
	}
	if wanted["filesystems"] {
		if content, err := readProcFile("/proc/filesystems"); err == nil {
			out.Filesystems = parseProcFilesystems(content)
		} else {
			appendErr(&errs, "filesystems", err)
		}
	}
	if wanted["version"] {
		if content, err := readProcFile("/proc/version"); err == nil {
			out.Version = strings.TrimSpace(content)
		} else {
			appendErr(&errs, "version", err)
		}
	}
	if wanted["slabinfo"] {
		if content, err := readProcFile("/proc/slabinfo"); err == nil {
			out.SlabInfo = parseProcSlabinfo(content)
		} else {
			appendErr(&errs, "slabinfo", err)
		}
	}

	out.Errors = errs
	return out, out.Err()
}

func HandleGetProcDiagnostics(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetProcDiagnosticsInput,
) (*mcp.CallToolResult, *ProcDiagnosticsOutput, error) {
	sections := strings.TrimSpace(input.Sections)
	return handleToolCall(
		ctx,
		config.ToolNameGetProcDiagnostics,
		0,
		func(ctx context.Context) (*ProcDiagnosticsOutput, error) {
			return GatherProcDiagnostics(ctx, sections)
		},
	)
}
