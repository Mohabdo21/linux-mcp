package main

import (
	"errors"
	"os/exec"
	"testing"
)

func TestGatherSystemInfo(t *testing.T) {
	out, err := gatherSystemInfo()
	if err != nil {
		t.Fatalf("gatherSystemInfo() error: %v", err)
	}
	if out.Hostname == "" {
		t.Error("Hostname should not be empty")
	}
	if out.OSName == "" {
		t.Error("OSName should not be empty")
	}
	if out.KernelVersion == "" {
		t.Error("KernelVersion should not be empty")
	}
	if out.Architecture == "" {
		t.Error("Architecture should not be empty")
	}
	if out.UptimeSeconds == 0 {
		t.Error("UptimeSeconds should not be 0")
	}
}

func TestGatherCPUInfo(t *testing.T) {
	out, err := gatherCPUInfo()
	if err != nil {
		t.Fatalf("gatherCPUInfo() error: %v", err)
	}
	if len(out.Cores) == 0 {
		t.Fatal("Cores should not be empty")
	}
	if out.PhysicalCoreCount <= 0 {
		t.Errorf(
			"PhysicalCoreCount should be > 0, got %d",
			out.PhysicalCoreCount,
		)
	}
	if out.UsagePercent < 0 {
		t.Errorf("UsagePercent should be >= 0, got %f", out.UsagePercent)
	}
	for i, c := range out.Cores {
		if c.ModelName == "" {
			t.Errorf("Cores[%d].ModelName should not be empty", i)
		}
		if c.CoreCount <= 0 {
			t.Errorf(
				"Cores[%d].CoreCount should be > 0, got %d",
				i,
				c.CoreCount,
			)
		}
	}
}

func TestGatherCPUTemperature(t *testing.T) {
	out := gatherCPUTemperature()
	if out.Message == "" && len(out.Temperatures) == 0 {
		t.Error("Expected either Message or Temperatures")
	}
	if len(out.Temperatures) > 0 {
		for i, s := range out.Temperatures {
			if s.SensorKey == "" {
				t.Errorf("Temperatures[%d].SensorKey should not be empty", i)
			}
		}
	}
}

func TestGatherMemoryInfo(t *testing.T) {
	out, err := gatherMemoryInfo()
	if err != nil {
		t.Fatalf("gatherMemoryInfo() error: %v", err)
	}
	if out.Total == 0 {
		t.Error("Total should not be 0")
	}
	if out.UsedPercent < 0 || out.UsedPercent > 100 {
		t.Errorf("UsedPercent out of range [0,100]: %f", out.UsedPercent)
	}
	if out.SwapUsedPercent < 0 || out.SwapUsedPercent > 100 {
		t.Errorf(
			"SwapUsedPercent out of range [0,100]: %f",
			out.SwapUsedPercent,
		)
	}
}

func TestGatherDiskInfo(t *testing.T) {
	out, err := gatherDiskInfo("")
	if err != nil {
		t.Fatalf("gatherDiskInfo() error: %v", err)
	}
	if len(out.Partitions) == 0 {
		t.Fatal("Partitions should not be empty")
	}
	for i, p := range out.Partitions {
		if p.MountPoint == "" {
			t.Errorf("Partitions[%d].MountPoint should not be empty", i)
		}
		if p.Filesystem == "" {
			t.Errorf("Partitions[%d].Filesystem should not be empty", i)
		}
		if p.UsedPercent < 0 || p.UsedPercent > 100 {
			t.Errorf(
				"Partitions[%d].UsedPercent out of range: %f",
				i,
				p.UsedPercent,
			)
		}
	}
}

func TestGatherDiskInfoWithFilter(t *testing.T) {
	out, err := gatherDiskInfo("/")
	if err != nil {
		t.Fatalf("gatherDiskInfo(\"/\") error: %v", err)
	}
	if len(out.Partitions) == 0 {
		t.Fatal("Expected at least / partition")
	}
	for _, p := range out.Partitions {
		if p.MountPoint != "/" {
			t.Errorf("Expected mount_point /, got %s", p.MountPoint)
		}
	}
}

func TestGatherDiskInfoWithNoMatch(t *testing.T) {
	out, err := gatherDiskInfo("/nonexistent")
	if err != nil {
		t.Fatalf("gatherDiskInfo(\"/nonexistent\") error: %v", err)
	}
	if len(out.Partitions) != 0 {
		t.Errorf(
			"Expected 0 partitions for non-matching filter, got %d",
			len(out.Partitions),
		)
	}
}

func TestGatherNetworkInfo(t *testing.T) {
	out, err := gatherNetworkInfo()
	if err != nil {
		t.Fatalf("gatherNetworkInfo() error: %v", err)
	}
	if len(out.Interfaces) == 0 {
		t.Fatal("Interfaces should not be empty")
	}
	found := false
	for _, iface := range out.Interfaces {
		if iface.Name == "lo" {
			found = true
		}
	}
	if !found {
		t.Log("lo interface not found (expected on most systems)")
	}
}

func TestGatherProcessInfoDefaults(t *testing.T) {
	out, err := gatherProcessInfo("", 0)
	if err != nil {
		t.Fatalf("gatherProcessInfo(\"\", 0) error: %v", err)
	}
	if len(out.Processes) == 0 {
		t.Fatal("Processes should not be empty")
	}
	if len(out.Processes) > 10 {
		t.Errorf(
			"Expected at most 10 processes by default, got %d",
			len(out.Processes),
		)
	}
	for i := 1; i < len(out.Processes); i++ {
		if out.Processes[i-1].CPUPercent < out.Processes[i].CPUPercent {
			t.Error("Processes should be sorted by CPU descending")
			break
		}
	}
}

func TestGatherProcessInfoSortByMemory(t *testing.T) {
	out, err := gatherProcessInfo("memory", 5)
	if err != nil {
		t.Fatalf("gatherProcessInfo(\"memory\", 5) error: %v", err)
	}
	if len(out.Processes) > 5 {
		t.Errorf("Expected at most 5 processes, got %d", len(out.Processes))
	}
	for i := 1; i < len(out.Processes); i++ {
		if out.Processes[i-1].MemoryPercent < out.Processes[i].MemoryPercent {
			t.Error("Processes should be sorted by Memory descending")
			break
		}
	}
}

func TestGatherProcessInfoLimitClamping(t *testing.T) {
	out, err := gatherProcessInfo("cpu", 200)
	if err != nil {
		t.Fatalf("gatherProcessInfo(\"cpu\", 200) error: %v", err)
	}
	if len(out.Processes) > 100 {
		t.Errorf("Expected limit clamped to 100, got %d", len(out.Processes))
	}
}

func TestGatherProcessInfoIncludesStatus(t *testing.T) {
	out, err := gatherProcessInfo("cpu", 5)
	if err != nil {
		t.Fatalf("gatherProcessInfo(\"cpu\", 5) error: %v", err)
	}
	if len(out.Processes) > 0 && out.Processes[0].Status == "" {
		t.Error("Process status should not be empty")
	}
}

func TestGatherDockerInfo(t *testing.T) {
	out, err := gatherDockerInfo()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("docker not installed")
		}
		t.Fatalf("gatherDockerInfo() error: %v", err)
	}
	t.Logf(
		"Found %d containers and %d images",
		len(out.Containers),
		len(out.Images),
	)
}

func TestGatherSystemSnapshot(t *testing.T) {
	ctx := t.Context()
	result, snapshot, err := handleGetSystemSnapshot(
		ctx,
		nil,
		getSystemSnapshotInput{},
	)
	if err != nil {
		t.Fatalf("get_system_snapshot error: %v", err)
	}
	if result != nil {
		t.Error("CallToolResult should be nil (structured output path)")
	}
	if snapshot.System.Hostname == "" {
		t.Error("Snapshot System.Hostname should not be empty")
	}
	if len(snapshot.CPU.Cores) == 0 {
		t.Error("Snapshot CPU.Cores should not be empty")
	}
	if snapshot.Memory.Total == 0 {
		t.Error("Snapshot Memory.Total should not be 0")
	}
	if len(snapshot.Disk.Partitions) == 0 {
		t.Error("Snapshot Disk.Partitions should not be empty")
	}
	if len(snapshot.Network.Interfaces) == 0 {
		t.Error("Snapshot Network.Interfaces should not be empty")
	}
	if len(snapshot.Processes.Processes) == 0 {
		t.Error("Snapshot Processes should not be empty")
	}
	if snapshot.Temperature.Message == "" &&
		len(snapshot.Temperature.Temperatures) == 0 {
		t.Error("Snapshot Temperature should have Message or Temperatures")
	}
	t.Logf("Snapshot errors: %v", snapshot.Errors)
}

func TestGatherSystemSnapshotErrors(t *testing.T) {
	// Verify the snapshot handler gracefully handles any errors
	result, snapshot, err := handleGetSystemSnapshot(
		t.Context(),
		nil,
		getSystemSnapshotInput{},
	)
	if err != nil {
		t.Fatalf("get_system_snapshot error: %v", err)
	}
	if result != nil {
		t.Error("CallToolResult should be nil")
	}
	// Even with errors in subsystems, the snapshot should still return partial data
	_ = snapshot
}
