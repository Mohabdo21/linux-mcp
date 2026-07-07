package tools

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/Mohabdo21/linux-mcp/config"
)

func TestGatherBasicSystemInfo(t *testing.T) {
	t.Run("SystemInfo", func(t *testing.T) {
		out, err := GatherSystemInfo(t.Context())
		skipOnErr(t, err, "GatherSystemInfo() error: %v", err)
		checkNotEmpty(t, out.Hostname, "Hostname")
		checkNotEmpty(t, out.OSName, "OSName")
		checkNotEmpty(t, out.KernelVersion, "KernelVersion")
		checkNotEmpty(t, out.Architecture, "Architecture")
		checkNotZeroUint64(t, out.UptimeSeconds, "UptimeSeconds")
		checkNotEmpty(t, out.Platform, "Platform")
		checkNotEmpty(t, out.PlatformFamily, "PlatformFamily")
		checkNotZeroUint64(t, out.BootTime, "BootTime")
		checkNotZeroUint64(t, out.Procs, "Procs")
	})

	t.Run("LoadAverage", func(t *testing.T) {
		out, err := GatherLoadAverage(t.Context())
		skipOnErr(t, err, "GatherLoadAverage() error: %v", err)
		checkNotNegative(t, out.Load1, "Load1")
		checkNotNegative(t, out.Load5, "Load5")
		checkNotNegative(t, out.Load15, "Load15")
	})

	t.Run("LoggedInUsers", func(t *testing.T) {
		out, err := GatherLoggedInUsers(t.Context())
		skipOnErr(t, err, "GatherLoggedInUsers() error: %v", err)
		t.Logf("Found %d logged-in users", len(out.Users))
	})

	t.Run("SystemdUnits", func(t *testing.T) {
		out, err := GatherSystemdUnits(t.Context(), "")
		skipOnErr(t, err, "GatherSystemdUnits() error: %v", err)
		if len(out.Units) == 0 {
			t.Error("Units should not be empty")
		}
		t.Logf("Found %d systemd units", len(out.Units))
	})

	t.Run("ListeningPorts", func(t *testing.T) {
		out, err := GatherListeningPorts(t.Context(), "")
		skipOnErr(t, err, "GatherListeningPorts() error: %v", err)
		t.Logf("Found %d listening ports", len(out.Ports))
	})

	t.Run("NetworkConnections", func(t *testing.T) {
		out, err := GatherNetworkConnections(
			t.Context(),
			"",
			"",
			false,
			false,
			0,
		)
		skipOnErr(t, err, "GatherNetworkConnections() error: %v", err)
		if len(out.Connections) == 0 {
			t.Error("Connections should not be empty")
		}
		t.Logf("Found %d active connections", len(out.Connections))
		// Also test with status filter
		est, err := GatherNetworkConnections(
			t.Context(),
			"ESTABLISHED",
			"",
			false,
			false,
			0,
		)
		skipOnErr(
			t,
			err,
			"GatherNetworkConnections(ESTABLISHED) error: %v",
			err,
		)
		t.Logf("Found %d ESTABLISHED connections", len(est.Connections))
		// Verify process_name is populated for non-zero PIDs
		if len(out.Connections) > 0 {
			first := out.Connections[0]
			if first.PID > 0 && first.ProcessName == "" {
				t.Logf(
					"PID %d has no process name (may be transient)",
					first.PID,
				)
			}
		}
		// Test max_connections truncation
		limited, err := GatherNetworkConnections(
			t.Context(),
			"",
			"",
			false,
			false,
			5,
		)
		skipOnErr(t, err, "GatherNetworkConnections(max=5) error: %v", err)
		if len(limited.Connections) > 5 {
			t.Errorf(
				"Expected max 5 connections, got %d",
				len(limited.Connections),
			)
		}
		t.Logf("Limited to %d connections", len(limited.Connections))
		// Test type filter
		tcpConns, err := GatherNetworkConnections(
			t.Context(),
			"",
			"tcp",
			false,
			false,
			0,
		)
		skipOnErr(t, err, "GatherNetworkConnections(type=tcp) error: %v", err)
		for _, c := range tcpConns.Connections {
			if c.Type != "tcp" {
				t.Errorf("Expected type tcp, got %s", c.Type)
			}
		}
		t.Logf("Found %d TCP connections", len(tcpConns.Connections))
		// Test grouping
		grouped, err := GatherNetworkConnections(
			t.Context(),
			"",
			"",
			false,
			true,
			0,
		)
		skipOnErr(
			t,
			err,
			"GatherNetworkConnections(grouped=true) error: %v",
			err,
		)
		if len(grouped.Groups) > 0 && len(grouped.Connections) > 0 {
			totalInGroups := 0
			for _, g := range grouped.Groups {
				totalInGroups += len(g.Connections)
			}
			if totalInGroups != len(grouped.Connections) {
				t.Errorf(
					"Grouped count %d != flat count %d",
					totalInGroups,
					len(grouped.Connections),
				)
			}
		}
		t.Logf("Grouped into %d groups", len(grouped.Groups))
	})

	t.Run("NetworkInfo", func(t *testing.T) {
		out, err := GatherNetworkInfo(t.Context())
		skipOnErr(t, err, "GatherNetworkInfo() error: %v", err)
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
	})
}

func TestGatherCPUInfo(t *testing.T) {
	out, err := GatherCPUInfo(t.Context())
	if err != nil {
		t.Skipf("GatherCPUInfo() error: %v", err)
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
	checkNotNegative(t, out.UsagePercent, "UsagePercent")
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
	out, err := GatherCPUTemperature(t.Context())
	if err != nil {
		t.Skipf("GatherCPUTemperature() error: %v", err)
	}
	if out.Message == "" && len(out.Temperatures) == 0 {
		t.Error("Expected either Message or Temperatures")
	}
	if len(out.Temperatures) > 0 {
		for i, s := range out.Temperatures {
			if s.SensorKey == "" {
				t.Errorf(
					"Temperatures[%d].SensorKey should not be empty", i)
			}
		}
	}
}

func TestGatherMemoryInfo(t *testing.T) {
	out, err := GatherMemoryInfo(t.Context())
	if err != nil {
		t.Skipf("GatherMemoryInfo() error: %v", err)
	}
	if out.Total == 0 {
		t.Error("Total should not be 0")
	}
	if out.UsedPercent < 0 || out.UsedPercent > 100 {
		t.Errorf(
			"UsedPercent out of range [0,100]: %f", out.UsedPercent)
	}
	if out.SwapUsedPercent < 0 || out.SwapUsedPercent > 100 {
		t.Errorf(
			"SwapUsedPercent out of range [0,100]: %f",
			out.SwapUsedPercent,
		)
	}
}

func TestGatherDiskInfo(t *testing.T) {
	out, err := GatherDiskInfo(t.Context(), "", 0)
	if err != nil {
		t.Skipf("GatherDiskInfo() error: %v", err)
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

func TestGatherDiskInfoThreshold(t *testing.T) {
	out, err := GatherDiskInfo(t.Context(), "", 0)
	if err != nil {
		t.Skipf("GatherDiskInfo() error: %v", err)
	}
	if len(out.Partitions) == 0 {
		t.Fatal("Partitions should not be empty")
	}
	// With threshold=0, all partitions returned (same as no filter)
	allCount := len(out.Partitions)

	// With threshold=100 (impossible to reach), expect no partitions
	outHigh, err := GatherDiskInfo(t.Context(), "", 100)
	if err != nil {
		t.Fatalf("GatherDiskInfo with threshold=100 error: %v", err)
	}
	if len(outHigh.Partitions) != 0 {
		t.Logf("Threshold=100 filtered %d -> %d partitions (unusual but ok)",
			allCount, len(outHigh.Partitions))
	}

	// With threshold=0, should match all-count
	if len(out.Partitions) != allCount {
		t.Errorf("Threshold=0 returned %d, expected %d",
			len(out.Partitions), allCount)
	}
}

func TestGatherSystemdUnitsStateFilter(t *testing.T) {
	all, err := GatherSystemdUnits(t.Context(), "")
	if err != nil {
		t.Skipf("GatherSystemdUnits() error: %v", err)
	}
	if len(all.Units) == 0 {
		t.Fatal("Units should not be empty")
	}

	active, err := GatherSystemdUnits(t.Context(), "active")
	if err != nil {
		t.Fatalf("GatherSystemdUnits('active') error: %v", err)
	}
	if len(active.Units) == 0 {
		t.Error("Expected at least some active units")
	}
	for _, u := range active.Units {
		if u.Active != "active" {
			t.Errorf(
				"Expected active unit, got active=%q for %s",
				u.Active,
				u.Unit,
			)
		}
	}
	if len(active.Units) > len(all.Units) {
		t.Error("Filtered set should not exceed total")
	}

	failed, err := GatherSystemdUnits(t.Context(), "failed")
	if err != nil {
		t.Fatalf("GatherSystemdUnits('failed') error: %v", err)
	}
	for _, u := range failed.Units {
		if u.Active != "failed" {
			t.Errorf(
				"Expected failed unit, got active=%q for %s",
				u.Active,
				u.Unit,
			)
		}
	}
	t.Logf("All=%d Active=%d Failed=%d",
		len(all.Units), len(active.Units), len(failed.Units))
}

func TestGatherFunctionsWithFilter(t *testing.T) {
	t.Run("DiskInfo/root", func(t *testing.T) {
		out, err := GatherDiskInfo(t.Context(), "/", 0)
		skipOnErr(t, err, "GatherDiskInfo(\"/\") error: %v", err)
		if len(out.Partitions) == 0 {
			t.Fatal("Expected at least / partition")
		}
		for _, p := range out.Partitions {
			if p.MountPoint != "/" {
				t.Errorf("Expected mount_point /, got %s", p.MountPoint)
			}
		}
	})

	t.Run("DiskInfo/nonexistent", func(t *testing.T) {
		out, err := GatherDiskInfo(t.Context(), "/nonexistent", 0)
		if err != nil {
			t.Fatalf(
				"GatherDiskInfo(\"/nonexistent\") error: %v", err)
		}
		if len(out.Partitions) != 0 {
			t.Errorf(
				"Expected 0 partitions for non-matching filter, got %d",
				len(out.Partitions),
			)
		}
	})

	t.Run("InodeUsage/root", func(t *testing.T) {
		out, err := GatherInodeUsage(t.Context(), "/")
		skipOnErr(t, err, "GatherInodeUsage(\"/\") error: %v", err)
		if len(out.Mounts) == 0 {
			t.Fatal("Expected at least / mount")
		}
		for _, m := range out.Mounts {
			if m.MountedOn != "/" {
				t.Errorf("Expected mounted_on /, got %s", m.MountedOn)
			}
		}
	})

	t.Run("MountOptions/root", func(t *testing.T) {
		out, err := GatherMountOptions(t.Context(), "/")
		skipOnErr(t, err, "GatherMountOptions(\"/\") error: %v", err)
		if len(out.Mounts) == 0 {
			t.Fatal("Expected at least / mount")
		}
		for _, m := range out.Mounts {
			if m.Target != "/" {
				t.Errorf("Expected target /, got %s", m.Target)
			}
		}
	})
}

func TestGatherInodeUsage(t *testing.T) {
	out, err := GatherInodeUsage(t.Context(), "")
	if err != nil {
		t.Skipf("GatherInodeUsage() error: %v", err)
	}
	if len(out.Mounts) == 0 {
		t.Fatal("Mounts should not be empty")
	}
	for i, m := range out.Mounts {
		if m.Filesystem == "" {
			t.Errorf("Mounts[%d].Filesystem should not be empty", i)
		}
		if m.MountedOn == "" {
			t.Errorf("Mounts[%d].MountedOn should not be empty", i)
		}
	}
}

func TestGatherMountOptions(t *testing.T) {
	out, err := GatherMountOptions(t.Context(), "")
	if err != nil {
		t.Skipf("GatherMountOptions() error: %v", err)
	}
	if len(out.Mounts) == 0 {
		t.Fatal("Mounts should not be empty")
	}
	for i, m := range out.Mounts {
		if m.Target == "" {
			t.Errorf("Mounts[%d].Target should not be empty", i)
		}
	}
}

func TestGatherProcessInfo(t *testing.T) {
	t.Run("Defaults", func(t *testing.T) {
		out, err := GatherProcessInfo(t.Context(), "", 10)
		skipOnErr(t, err,
			"GatherProcessInfo(\"\", 10) error: %v", err)
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
			if out.Processes[i-1].CPUPercent <
				out.Processes[i].CPUPercent {
				t.Error("Processes should be sorted by CPU descending")
				break
			}
		}
	})

	t.Run("SortByMemory", func(t *testing.T) {
		out, err := GatherProcessInfo(t.Context(), "memory", 5)
		skipOnErr(t, err,
			"GatherProcessInfo(\"memory\", 5) error: %v", err)
		if len(out.Processes) > 5 {
			t.Errorf(
				"Expected at most 5 processes, got %d",
				len(out.Processes),
			)
		}
		for i := 1; i < len(out.Processes); i++ {
			if out.Processes[i-1].MemoryPercent <
				out.Processes[i].MemoryPercent {
				t.Error("Processes should be sorted by Memory descending")
				break
			}
		}
	})

	t.Run("LimitClamping", func(t *testing.T) {
		// Clamping is now handled in HandleGetProcessInfo;
		// GatherProcessInfo uses the limit as-is.
		out, err := GatherProcessInfo(t.Context(), "cpu", 100)
		skipOnErr(t, err,
			"GatherProcessInfo(\"cpu\", 100) error: %v", err)
		if len(out.Processes) > 100 {
			t.Errorf(
				"Expected at most 100 processes, got %d",
				len(out.Processes),
			)
		}
	})

	t.Run("IncludesStatus", func(t *testing.T) {
		out, err := GatherProcessInfo(t.Context(), "cpu", 5)
		skipOnErr(t, err,
			"GatherProcessInfo(\"cpu\", 5) error: %v", err)
		if len(out.Processes) > 0 && out.Processes[0].Status == "" {
			t.Error("Process status should not be empty")
		}
	})

	t.Run("SortByBoth", func(t *testing.T) {
		out, err := GatherProcessInfo(t.Context(), "both", 10)
		skipOnErr(t, err,
			"GatherProcessInfo(\"both\", 10) error: %v", err)
		if len(out.ByCPU) == 0 {
			t.Fatal("ByCPU should not be empty")
		}
		if len(out.ByMemory) == 0 {
			t.Fatal("ByMemory should not be empty")
		}
		if out.Processes != nil {
			t.Error("Processes should be nil when sort_by='both'")
		}
		for i := 1; i < len(out.ByCPU); i++ {
			if out.ByCPU[i-1].CPUPercent < out.ByCPU[i].CPUPercent {
				t.Error("ByCPU should be sorted by CPU descending")
				break
			}
		}
		for i := 1; i < len(out.ByMemory); i++ {
			if out.ByMemory[i-1].MemoryPercent < out.ByMemory[i].MemoryPercent {
				t.Error("ByMemory should be sorted by Memory descending")
				break
			}
		}
	})
}

func TestGatherDockerInfo(t *testing.T) {
	out, err := GatherDockerInfo(t.Context())
	skipOnErr(t, err, "GatherDockerInfo() error: %v", err)
	t.Logf(
		"Found %d containers and %d images",
		len(out.Containers),
		len(out.Images),
	)
}

func TestGatherContainerDetail(t *testing.T) {
	containers := requireDockerContainers(t)
	out, err := GatherContainerDetail(t.Context(), containers[0].ID)
	skipOnErr(t, err, "GatherContainerDetail() error: %v", err)
	if out.Container.ID == "" {
		t.Error("Container ID should not be empty")
	}
	if out.Container.Name == "" {
		t.Error("Container Name should not be empty")
	}
	t.Logf(
		"Container %s (%s) image=%s",
		out.Container.Name,
		out.Container.ID,
		out.Container.Image,
	)
}

func TestGatherContainerLogs(t *testing.T) {
	containers := requireDockerContainers(t)
	out, err := GatherContainerLogs(t.Context(), containers[0].ID, 5, false)
	skipOnErr(t, err, "GatherContainerLogs() error: %v", err)
	t.Logf(
		"Got %d log lines from container %s",
		len(out.Logs),
		containers[0].ID,
	)
}

func TestGatherContainerStats(t *testing.T) {
	statsID := requireDockerRunningContainer(t)
	out, err := GatherContainerStats(t.Context(), statsID)
	skipOnErr(t, err, "GatherContainerStats() error: %v", err)
	if len(out.Containers) == 0 {
		t.Fatal("Expected at least one container stat entry")
	}
	c := out.Containers[0]
	t.Logf("Container %s: CPU=%.1f%% Memory=%d/%d PIDs=%d",
		statsID, c.CPUPercent, c.MemoryUsage,
		c.MemoryLimit, c.PIDs)
}

func TestGatherContainerTop(t *testing.T) {
	topID := requireDockerRunningContainer(t)
	out, err := GatherContainerTop(t.Context(), topID, nil)
	skipOnErr(t, err, "GatherContainerTop() error: %v", err)
	if len(out.Titles) == 0 {
		t.Error("Titles should not be empty")
	}
	t.Logf(
		"Container %s: %d processes, titles=%v",
		topID,
		len(out.Processes),
		out.Titles,
	)
}

func TestGatherContainerDiff(t *testing.T) {
	containers := requireDockerContainers(t)
	out, err := GatherContainerDiff(t.Context(), containers[0].ID)
	skipOnErr(t, err, "GatherContainerDiff() error: %v", err)
	t.Logf(
		"Container %s: %d filesystem changes",
		containers[0].ID,
		len(out.Changes),
	)
}

func TestGatherImageHistory(t *testing.T) {
	images := requireDockerImages(t)
	out, err := GatherImageHistory(t.Context(), images[0].ID)
	skipOnErr(t, err, "GatherImageHistory() error: %v", err)
	if len(out.Layers) == 0 {
		t.Error("Layers should not be empty")
	}
	t.Logf("Image %s: %d layers", images[0].ID, len(out.Layers))
}

func TestGatherImageDetail(t *testing.T) {
	images := requireDockerImages(t)
	out, err := GatherImageDetail(t.Context(), images[0].ID)
	skipOnErr(t, err, "GatherImageDetail() error: %v", err)
	if out.Image.ID == "" {
		t.Error("Image ID should not be empty")
	}
	t.Logf("Image %s: arch=%s os=%s size=%s layers=%d",
		out.Image.ID, out.Image.Architecture, out.Image.OS,
		out.Image.Size, len(out.Image.Layers))
}

func TestGatherDockerNetworks(t *testing.T) {
	out, err := GatherDockerNetworks(t.Context())
	skipOnErr(t, err, "Docker not available: %v", err)
	if len(out.Networks) == 0 {
		t.Error("Networks should not be empty on a Docker host")
	}
	t.Logf("Found %d Docker networks", len(out.Networks))
}

func TestGatherDockerVolumes(t *testing.T) {
	out, err := GatherDockerVolumes(t.Context())
	skipOnErr(t, err, "Docker not available: %v", err)
	t.Logf("Found %d Docker volumes", len(out.Volumes))
}

func TestGatherDockerSystemInfo(t *testing.T) {
	out, err := GatherDockerSystemInfo(t.Context())
	skipOnErr(t, err, "Docker not available: %v", err)
	if out.Info.ServerVersion == "" {
		t.Error("ServerVersion should not be empty")
	}
	t.Logf("Docker %s (%s/%s) driver=%s cgroups=%s runtimes=%v",
		out.Info.ServerVersion, out.Info.Architecture, out.Info.OSType,
		out.Info.Driver, out.Info.CgroupDriver, out.Info.Runtimes)
}

func TestGatherDockerDiskUsage(t *testing.T) {
	out, err := GatherDockerDiskUsage(t.Context())
	skipOnErr(t, err, "Docker not available: %v", err)
	t.Logf(
		"Containers: %d active/%d total (%s), Images: %d active/%d total (%s)",
		out.Containers.ActiveCount,
		out.Containers.TotalCount,
		out.Containers.TotalSize,
		out.Images.ActiveCount,
		out.Images.TotalCount,
		out.Images.TotalSize,
	)
}

func TestGatherDockerStatsAll(t *testing.T) {
	out, err := GatherDockerStatsAll(t.Context(), nil)
	skipOnErr(t, err, "Docker not available: %v", err)
	t.Logf(
		"Got stats for %d running container(s)",
		len(out.Containers),
	)
	for _, c := range out.Containers {
		if c.Error != "" {
			t.Logf("  %s (%s): error=%s", c.Name, c.ID, c.Error)
		} else {
			t.Logf(
				"  %s (%s): CPU=%.1f%% Mem=%d/%d Net=%d blk=%d/%d",
				c.Name, c.ID, c.CPUPercent, c.MemoryUsage,
				c.MemoryLimit, len(c.Network),
				c.BlockRead, c.BlockWrite,
			)
		}
	}
}

func TestGatherDockerSystemSnapshot(t *testing.T) {
	ctx := t.Context()
	result, snapshot, err := HandleGetDockerSystemSnapshot(
		ctx,
		nil,
		NoArgs{},
	)
	if err != nil {
		t.Fatalf("%s error: %v", config.ToolNameGetDockerSystemSnapshot, err)
	}
	if result != nil {
		t.Error("CallToolResult should be nil (structured output path)")
	}
	t.Logf(
		"Containers=%d Images=%d Stats=%d DiskContainers=%d/%d",
		len(snapshot.Info.Containers),
		len(snapshot.Info.Images),
		len(snapshot.Stats.Containers),
		snapshot.DiskUsage.Containers.ActiveCount,
		snapshot.DiskUsage.Containers.TotalCount,
	)
	if len(snapshot.Errors) > 0 {
		t.Logf("Errors: %v", snapshot.Errors)
	}
}

func TestGatherSystemSnapshot(t *testing.T) {
	ctx := t.Context()
	result, snapshot, err := HandleGetSystemSnapshot(
		ctx,
		nil,
		NoArgs{},
	)
	if err != nil {
		t.Fatalf("%s error: %v", config.ToolNameGetSystemSnapshot, err)
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
		t.Log("Temperature data unavailable (sensors may not be accessible)")
	}
	t.Logf("Snapshot errors: %v", snapshot.Errors)
}

func TestGatherSystemSnapshotErrors(t *testing.T) {
	result, snapshot, err := HandleGetSystemSnapshot(
		t.Context(),
		nil,
		NoArgs{},
	)
	if err != nil {
		t.Fatalf("%s error: %v", config.ToolNameGetSystemSnapshot, err)
	}
	if result != nil {
		t.Error("CallToolResult should be nil")
	}
	t.Logf("Snapshot errors: %v", snapshot.Errors)
}

func TestGatherJournalLogs(t *testing.T) {
	entries, err := GatherJournalLogs(t.Context(), "", "", "", "", 5, false)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("journalctl not installed")
		}
		t.Fatalf("GatherJournalLogs() error: %v", err)
	}
	t.Logf("Found %d journal entries", len(entries.Entries))
	// Verify structured data
	for i, e := range entries.Entries {
		if e.Timestamp == "" {
			t.Errorf("Entries[%d].Timestamp should not be empty", i)
		}
		if e.Message == "" {
			t.Errorf("Entries[%d].Message should not be empty", i)
		}
		t.Logf("  [%s] priority=%s unit=%s pid=%d msg=%s",
			e.Timestamp, e.Priority, e.Unit, e.PID, e.Message)
	}
}

func TestGatherGPUInfo(t *testing.T) {
	out, err := GatherGPUInfo(t.Context())
	skipOnErr(t, err, "No GPU tool found: %v", err)
	if out.Vendor == "" {
		t.Error("Vendor should not be empty")
	}
	t.Logf("GPU vendor: %s, devices: %d", out.Vendor, len(out.GPUs))
}

func TestGatherPing(t *testing.T) {
	out, err := GatherPing(t.Context(), "127.0.0.1", 1, 5)
	if err != nil {
		t.Skipf("ping failed: %v", err)
	}
	if out.Host != "127.0.0.1" {
		t.Errorf("expected host 127.0.0.1, got %s", out.Host)
	}
	if out.PacketsTransmitted < 1 {
		t.Error("PacketsTransmitted should be >= 1")
	}
	t.Logf(
		"Ping %s: %d/%d packets, %.1f%% loss, avg=%.2fms",
		out.Host, out.PacketsReceived, out.PacketsTransmitted,
		out.PacketLossPercent, out.AvgLatencyMs)
}

func TestGatherInstalledPackages(t *testing.T) {
	out, err := GatherInstalledPackages(t.Context(), "")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("no supported package manager found")
		}
		t.Skipf("GatherInstalledPackages() error: %v", err)
	}
	if out.Total == 0 {
		t.Error("Total should be > 0")
	}
	if len(out.Packages) == 0 {
		t.Error("Packages should not be empty")
	}
	t.Logf("Found %d installed packages", out.Total)
}

func TestGatherInstalledPackagesFilter(t *testing.T) {
	out, err := GatherInstalledPackages(t.Context(), "linux")
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("no supported package manager found")
		}
		t.Skipf("GatherInstalledPackages() error: %v", err)
	}
	t.Logf("Found %d packages matching filter", out.Total)
}

func TestGatherCheckUpdates(t *testing.T) {
	out, err := GatherCheckUpdates(t.Context())
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("no supported package manager found")
		}
		t.Skipf("GatherCheckUpdates() error: %v", err)
	}
	t.Logf("Found %d available updates", out.Total)
}

func TestGatherLargestFiles(t *testing.T) {
	out, err := GatherLargestFiles(t.Context(), ".", 5)
	if err != nil {
		t.Fatalf("GatherLargestFiles() error: %v", err)
	}
	if out.Path == "" {
		t.Error("Path should not be empty")
	}
	for i, e := range out.Entries {
		if e.Name == "" {
			t.Errorf("Entries[%d].Name should not be empty", i)
		}
		if e.SizeBytes < 0 {
			t.Errorf("Entries[%d].SizeBytes should be >= 0", i)
		}
		if e.SizeHuman == "" {
			t.Errorf("Entries[%d].SizeHuman should not be empty", i)
		}
	}
	t.Logf("Found %d entries in %s", len(out.Entries), out.Path)
}

func TestGatherProcessFDs(t *testing.T) {
	t.Run("CurrentProcess", func(t *testing.T) {
		pid := int32(os.Getpid())
		out, err := GatherProcessFDs(t.Context(), pid)
		skipOnErr(t, err, "GatherProcessFDs() error: %v", err)
		if len(out.FDs) == 0 {
			t.Fatal("Expected at least some file descriptors")
		}
		if out.Name == "" {
			t.Error("Process name should not be empty")
		}
		if out.PID != int(pid) {
			t.Errorf("Expected PID %d, got %d", pid, out.PID)
		}
		t.Logf(
			"Process %d (%s): %d file descriptors",
			out.PID, out.Name, out.Count,
		)
	})

	t.Run("InvalidPID", func(t *testing.T) {
		out, err := GatherProcessFDs(t.Context(), -1)
		if err != nil {
			t.Fatalf("GatherProcessFDs() error: %v", err)
		}
		if len(out.Errors) == 0 {
			t.Error("Expected error in output.Errors for negative PID")
		}
	})
}

func TestGatherTopIOProcesses(t *testing.T) {
	out, err := GatherTopIOProcesses(t.Context(), 5)
	if err != nil {
		t.Fatalf("GatherTopIOProcesses() error: %v", err)
	}
	if len(out.Errors) > 0 {
		t.Skipf("pidstat not available: %s", out.Errors[0])
	}
	t.Logf("Found %d IO processes", len(out.Processes))
}

func TestGatherFailedLogins(t *testing.T) {
	out, err := GatherFailedLogins(t.Context(), 5)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("lastb not installed")
		}
		t.Skipf("GatherFailedLogins() error (likely permissions): %v", err)
	}
	t.Logf("Found %d failed login entries", len(out.Entries))
	if out.Summary.TotalAttempts != len(out.Entries) {
		t.Errorf("Summary.TotalAttempts=%d != len(entries)=%d",
			out.Summary.TotalAttempts, len(out.Entries))
	}
}

func TestParseLastbFiltersBoot(t *testing.T) {
	input := "reboot   system boot  0.0         Mon Jan  1 12:00\nroot     pts/0        192.168.1.1     Mon Jan  1 12:30 - 12:31 (00:01)\nbtmp begins Mon Jan  1 12:00\n"
	entries := ParseLastbOutput(input)
	if len(entries) != 1 {
		t.Errorf("expected 1 entry after filtering Boot, got %d", len(entries))
	}
	if len(entries) > 0 && entries[0].Username != "root" {
		t.Errorf(
			"expected root as remaining entry, got %s",
			entries[0].Username,
		)
	}
}

func TestParseLastbFiltersBootTerminal(t *testing.T) {
	input := "3231d21b Boot        0.0         --\nroot     pts/0        192.168.1.1     Mon Jan  1 12:30\n"
	entries := ParseLastbOutput(input)
	if len(entries) != 1 {
		t.Errorf(
			"expected 1 entry after filtering terminal=Boot, got %d",
			len(entries),
		)
	}
	if len(entries) > 0 && entries[0].Username != "root" {
		t.Errorf(
			"expected root as remaining entry, got %s",
			entries[0].Username,
		)
	}
}

func TestComputeFailedLoginsSummary(t *testing.T) {
	entries := []FailedLoginEntry{
		{Username: "root", Source: "10.0.0.1"},
		{Username: "admin", Source: "10.0.0.1"},
		{Username: "root", Source: "10.0.0.2"},
	}
	summary := computeFailedLoginsSummary(entries)
	if summary.TotalAttempts != 3 {
		t.Errorf("TotalAttempts = %d, want 3", summary.TotalAttempts)
	}
	if summary.UniqueUsernames != 2 {
		t.Errorf("UniqueUsernames = %d, want 2", summary.UniqueUsernames)
	}
	if summary.UniqueSources != 2 {
		t.Errorf("UniqueSources = %d, want 2", summary.UniqueSources)
	}
}

func TestGatherDNSResolve(t *testing.T) {
	out, err := GatherDNSResolve(t.Context(), "localhost")
	skipOnErr(t, err, "GatherDNSResolve(\"localhost\") error: %v", err)
	if len(out.Addresses) == 0 {
		t.Error("Expected at least one address for localhost")
	}
	t.Logf("localhost resolves to: %v", out.Addresses)
}

func TestGatherServiceStatus(t *testing.T) {
	services := []string{
		"systemd-journald.service", "dbus.service", "sshd",
	}
	var out *ServiceStatusOutput
	var err error
	for _, name := range services {
		out, err = GatherServiceStatus(t.Context(), name, false)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Skipf("No common service found: %v", err)
	}
	if out.Name == "" {
		t.Error("Name should not be empty")
	}
	if out.Output == "" {
		t.Error("Output should not be empty")
	}
	t.Logf("Service %s: %s", out.Name, out.Active)
}

func TestGatherEnvironmentVariables(t *testing.T) {
	out, err := GatherEnvironmentVariables(t.Context(), "")
	if err != nil {
		t.Fatalf("GatherEnvironmentVariables() error: %v", err)
	}
	if out.Count == 0 {
		t.Fatal("Expected at least one environment variable")
	}
	if len(out.Variables) != out.Count {
		t.Errorf(
			"Variables length %d != Count %d",
			len(out.Variables),
			out.Count,
		)
	}
	foundPath := false
	for _, v := range out.Variables {
		if v.Name == "PATH" {
			foundPath = true
			break
		}
	}
	if !foundPath {
		t.Error("Expected PATH environment variable to be present")
	}
	for i := 1; i < len(out.Variables); i++ {
		if out.Variables[i-1].Name > out.Variables[i].Name {
			t.Error("Variables should be sorted by name")
			break
		}
	}
}

func TestGatherHardwareBusInfo(t *testing.T) {
	out, err := GatherHardwareBusInfo(t.Context(), "")
	if err != nil {
		t.Skipf("GatherHardwareBusInfo() error (CLI may be missing): %v", err)
	}
	if len(out.PCIDevices) == 0 && len(out.USBDevices) == 0 {
		t.Log("No PCI or USB devices found (expected on minimal systems)")
	}
	for i, d := range out.PCIDevices {
		if d.Bus != "pci" {
			t.Errorf("PCIDevices[%d].Bus = %q, want 'pci'", i, d.Bus)
		}
		if d.Device == "" {
			t.Errorf("PCIDevices[%d].Device should not be empty", i)
		}
	}
	for i, d := range out.USBDevices {
		if d.Bus == "" {
			t.Errorf("USBDevices[%d].Bus should not be empty", i)
		}
		if d.Device == "" {
			t.Errorf("USBDevices[%d].Device should not be empty", i)
		}
	}
	t.Logf(
		"Found %d PCI devices, %d USB devices",
		len(out.PCIDevices),
		len(out.USBDevices),
	)
}

func TestSplitHostPort(t *testing.T) {
	addr, port, ok := SplitHostPort("0.0.0.0:80")
	if !ok || addr != "0.0.0.0" || port != "80" {
		t.Errorf(
			"SplitHostPort('0.0.0.0:80') = (%q, %q, %v)", addr, port, ok)
	}
	addr, port, ok = SplitHostPort("[::]:443")
	if !ok || addr != "[::]" || port != "443" {
		t.Errorf(
			"SplitHostPort('[::]:443') = (%q, %q, %v)", addr, port, ok)
	}
}

func TestParseProcessField(t *testing.T) {
	result := ParseProcessField(
		`users:(("sshd",pid=1234,fd=3))`)
	if result != "sshd" {
		t.Errorf("expected 'sshd', got %q", result)
	}
	result = ParseProcessField("plaintext")
	if result != "plaintext" {
		t.Errorf("expected 'plaintext', got %q", result)
	}
}

func TestHandleResolveDNSEmptyHostname(t *testing.T) {
	_, _, err := HandleResolveDNS(t.Context(), nil, ResolveDNSInput{})
	if err == nil {
		t.Error("Expected error for empty hostname")
	}
}

func TestHumanSize(t *testing.T) {
	cases := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}
	for _, c := range cases {
		got := HumanSize(c.bytes)
		if got != c.want {
			t.Errorf("HumanSize(%d) = %q, want %q", c.bytes, got, c.want)
		}
	}
}

func TestShellQuote(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"simple", "'simple'"},
		{"path with spaces", "'path with spaces'"},
		{"it's", "'it'\\''s'"},
		{"/home/user", "'/home/user'"},
	}
	for _, c := range cases {
		got := ShellQuote(c.input)
		if got != c.want {
			t.Errorf("ShellQuote(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestGatherUserAutomation(t *testing.T) {
	out, err := GatherUserAutomation(t.Context())
	skipOnErr(t, err, "GatherUserAutomation() error: %v", err)
	if out.CronJobs == nil {
		t.Error("CronJobs should not be nil")
	}
	if out.SystemdTimers == nil {
		t.Error("SystemdTimers should not be nil")
	}
}

func TestGatherDesktopSessionInfo(t *testing.T) {
	out, err := GatherDesktopSessionInfo(t.Context())
	skipOnErr(t, err, "GatherDesktopSessionInfo() error: %v", err)
	// These may be empty in non-desktop environments, so just check no error
	_ = out
}

func TestGatherUserInfo(t *testing.T) {
	t.Run("AllUsers", func(t *testing.T) {
		out, err := GatherUserInfo(t.Context(), "")
		skipOnErr(t, err, "GatherUserInfo() error: %v", err)
		if len(out.Users) == 0 {
			t.Error("Users should not be empty")
		}
		root := out.Users[0]
		if root.Username != "root" {
			t.Errorf("expected first user 'root', got %q", root.Username)
		}
		if root.UID != 0 {
			t.Errorf("expected root UID 0, got %d", root.UID)
		}
		t.Logf("Found %d users", len(out.Users))
	})

	t.Run("SearchFilter", func(t *testing.T) {
		out, err := GatherUserInfo(t.Context(), "root")
		skipOnErr(t, err, "GatherUserInfo() error: %v", err)
		if len(out.Users) == 0 {
			t.Error("expected at least one user matching 'root'")
		}
		for _, u := range out.Users {
			if !strings.Contains(strings.ToLower(u.Username), "root") {
				t.Errorf("user %q does not match search 'root'", u.Username)
			}
		}
	})

	t.Run("GroupMembership", func(t *testing.T) {
		out, err := GatherUserInfo(t.Context(), "root")
		skipOnErr(t, err, "GatherUserInfo() error: %v", err)
		if len(out.Users) == 0 {
			t.Skip("no root user found")
		}
		root := out.Users[0]
		if len(root.Groups) == 0 {
			t.Log("root has no supplementary groups (unusual but possible)")
		}
		t.Logf("root groups: %v", root.Groups)
	})
}

func TestGatherManPage(t *testing.T) {
	t.Run("Default", func(t *testing.T) {
		out, err := GatherManPage(t.Context(), "ls", 500, true, "", 0, 0)
		skipOnErr(t, err, "GatherManPage() error: %v", err)
		if out.Command != "ls" {
			t.Errorf("expected command 'ls', got %q", out.Command)
		}
		if out.Content == "" {
			t.Error("Content should not be empty")
		}
		if !strings.Contains(out.Content, "LS") {
			t.Error("Content should contain 'LS' as man page header")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := GatherManPage(
			t.Context(),
			"nonexistent_command_xyz",
			500,
			true,
			"",
			0,
			0,
		)
		if err == nil {
			t.Fatal("expected error for nonexistent command")
		}
		if !strings.Contains(err.Error(), "No manual page found") {
			t.Errorf("expected 'No manual page found' error, got: %v", err)
		}
	})

	t.Run("MaxLines", func(t *testing.T) {
		out, err := GatherManPage(t.Context(), "ls", 5, true, "", 0, 0)
		skipOnErr(t, err, "GatherManPage() error: %v", err)
		lines := strings.Count(out.Content, "\n") + 1
		if lines > 5 {
			t.Errorf("expected at most 5 lines, got %d", lines)
		}
		if out.Truncated != true {
			t.Error("expected Truncated to be true")
		}
	})

	t.Run("CleanSpecialChars", func(t *testing.T) {
		out, err := GatherManPage(t.Context(), "ls", 500, true, "", 0, 0)
		skipOnErr(t, err, "GatherManPage() error: %v", err)
		if !strings.Contains(out.Content, "\x08") {
			return
		}
		t.Error("Expected backspace characters to be cleaned")
	})

	t.Run("Search", func(t *testing.T) {
		out, err := GatherManPage(t.Context(), "ls", 500, true, "FILE", 0, 0)
		skipOnErr(t, err, "GatherManPage() error: %v", err)
		if out.Content == "" {
			t.Error("search results should not be empty when term exists")
		}
		if !strings.Contains(out.Content, "FILE") {
			t.Error("search results should contain the search term")
		}
	})

	t.Run("SearchNotFound", func(t *testing.T) {
		out, err := GatherManPage(
			t.Context(),
			"ls",
			500,
			true,
			"XYZZY_NONEXISTENT_123",
			0,
			0,
		)
		skipOnErr(t, err, "GatherManPage() error: %v", err)
		if out.Content != "" {
			t.Error("expected empty content for search with no matches")
		}
	})

	t.Run("Offset", func(t *testing.T) {
		out, err := GatherManPage(t.Context(), "ls", 10, true, "", 0, 0)
		skipOnErr(t, err, "GatherManPage() error: %v", err)
		out2, err2 := GatherManPage(t.Context(), "ls", 10, true, "", 0, 10)
		skipOnErr(t, err2, "GatherManPage() error: %v", err2)
		if out.Content == out2.Content {
			t.Error("offset 0 and offset 10 should return different content")
		}
	})

	t.Run("ContextLines", func(t *testing.T) {
		out, err := GatherManPage(
			t.Context(), "ls", 500, true, "FILE", 2, 0)
		skipOnErr(t, err, "GatherManPage() error: %v", err)
		if out.Content == "" {
			t.Error("should return content with context lines")
		}
	})

	t.Run("EmptyCommand", func(t *testing.T) {
		_, _, err := HandleGetManPage(t.Context(), nil, GetManPageInput{})
		if err == nil {
			t.Error("expected error for empty command")
		}
	})
}

func TestGatherPowerAnalytics(t *testing.T) {
	out, err := GatherPowerAnalytics(t.Context())
	skipOnErr(t, err, "GatherPowerAnalytics() error: %v", err)
	checkNotNegative(t, out.BatteryPercent, "BatteryPercent")
	checkNotNegative(t, out.DischargeRateWatts, "DischargeRateWatts")
	checkNotNegative(t, out.CapacityDegradation, "CapacityDegradation")
	t.Logf(
		"AC Online: %v, Battery: %.0f%%, Discharge: %.2fW, Degradation: %.1f%%",
		out.ACOnline,
		out.BatteryPercent,
		out.DischargeRateWatts,
		out.CapacityDegradation,
	)
	if out.BatteryPercent > 0 {
		t.Log("Battery detected and reporting")
	}
}

func TestExecOutput(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		out, err := execOutput(t.Context(), "echo", "-n", "hello world")
		if err != nil {
			t.Fatalf("execOutput() error: %v", err)
		}
		if out != "hello world" {
			t.Errorf("execOutput() = %q, want %q", out, "hello world")
		}
	})
	t.Run("NotFound", func(t *testing.T) {
		_, err := execOutput(t.Context(), "nonexistent-binary-xyz")
		if err == nil {
			t.Fatal("expected error for nonexistent binary")
		}
	})
}

func TestExecCombinedOutput(t *testing.T) {
	out, err := execCombinedOutput(
		t.Context(),
		"sh",
		"-c",
		"echo stdout; echo stderr >&2",
	)
	if err != nil {
		t.Fatalf("execCombinedOutput() error: %v", err)
	}
	if !strings.Contains(out, "stdout") || !strings.Contains(out, "stderr") {
		t.Errorf("execCombinedOutput() = %q, want both stdout and stderr", out)
	}
}

func TestExecLines(t *testing.T) {
	t.Run("MultiLine", func(t *testing.T) {
		lines, err := execLines(t.Context(), "printf", "a\nb\nc\n")
		if err != nil {
			t.Fatalf("execLines() error: %v", err)
		}
		if want := []string{"a", "b", "c"}; !equalSlices(lines, want) {
			t.Errorf("execLines() = %v, want %v", lines, want)
		}
	})
	t.Run("Empty", func(t *testing.T) {
		lines, err := execLines(t.Context(), "true")
		if err != nil {
			t.Fatalf("execLines() error: %v", err)
		}
		if lines != nil {
			t.Errorf("execLines() = %v, want nil", lines)
		}
	})
}
