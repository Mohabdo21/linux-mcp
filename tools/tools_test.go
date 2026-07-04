package tools

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestGatherSystemInfo(t *testing.T) {
	out, err := GatherSystemInfo(t.Context())
	if err != nil {
		t.Skipf("GatherSystemInfo() error: %v", err)
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
	out, err := GatherDiskInfo(t.Context(), "")
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

func TestGatherDiskInfoWithFilter(t *testing.T) {
	out, err := GatherDiskInfo(t.Context(), "/")
	if err != nil {
		t.Skipf("GatherDiskInfo(\"/\") error: %v", err)
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
	out, err := GatherDiskInfo(t.Context(), "/nonexistent")
	if err != nil {
		t.Fatalf("GatherDiskInfo(\"/nonexistent\") error: %v", err)
	}
	if len(out.Partitions) != 0 {
		t.Errorf(
			"Expected 0 partitions for non-matching filter, got %d",
			len(out.Partitions),
		)
	}
}

func TestGatherNetworkInfo(t *testing.T) {
	out, err := GatherNetworkInfo(t.Context())
	if err != nil {
		t.Skipf("GatherNetworkInfo() error: %v", err)
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
	out, err := GatherProcessInfo(t.Context(), "", 0)
	if err != nil {
		t.Skipf("GatherProcessInfo(\"\", 0) error: %v", err)
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
	out, err := GatherProcessInfo(t.Context(), "memory", 5)
	if err != nil {
		t.Skipf("GatherProcessInfo(\"memory\", 5) error: %v", err)
	}
	if len(out.Processes) > 5 {
		t.Errorf("Expected at most 5 processes, got %d", len(out.Processes))
	}
	for i := 1; i < len(out.Processes); i++ {
		if out.Processes[i-1].MemoryPercent <
			out.Processes[i].MemoryPercent {
			t.Error("Processes should be sorted by Memory descending")
			break
		}
	}
}

func TestGatherProcessInfoLimitClamping(t *testing.T) {
	out, err := GatherProcessInfo(t.Context(), "cpu", 200)
	if err != nil {
		t.Skipf("GatherProcessInfo(\"cpu\", 200) error: %v", err)
	}
	if len(out.Processes) > 100 {
		t.Errorf("Expected limit clamped to 100, got %d", len(out.Processes))
	}
}

func TestGatherProcessInfoIncludesStatus(t *testing.T) {
	out, err := GatherProcessInfo(t.Context(), "cpu", 5)
	if err != nil {
		t.Skipf("GatherProcessInfo(\"cpu\", 5) error: %v", err)
	}
	if len(out.Processes) > 0 && out.Processes[0].Status == "" {
		t.Error("Process status should not be empty")
	}
}

func TestGatherDockerInfo(t *testing.T) {
	out, err := GatherDockerInfo(t.Context())
	if err != nil {
		t.Skipf("GatherDockerInfo() error: %v", err)
	}
	t.Logf(
		"Found %d containers and %d images",
		len(out.Containers),
		len(out.Images),
	)
}

func TestGatherContainerDetail(t *testing.T) {
	containers, err := ListDockerContainers(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	if len(containers) == 0 {
		t.Skip("No containers available to inspect")
	}
	out, err := GatherContainerDetail(t.Context(), containers[0].ID)
	if err != nil {
		t.Skipf("GatherContainerDetail() error: %v", err)
	}
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
	containers, err := ListDockerContainers(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	if len(containers) == 0 {
		t.Skip("No containers available to fetch logs")
	}
	out, err := GatherContainerLogs(t.Context(), containers[0].ID, 5, false)
	if err != nil {
		t.Skipf("GatherContainerLogs() error: %v", err)
	}
	t.Logf(
		"Got %d log lines from container %s",
		len(out.Logs),
		containers[0].ID,
	)
}

func TestGatherContainerStats(t *testing.T) {
	containers, err := ListDockerContainers(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	var statsID string
	for _, c := range containers {
		if strings.Contains(c.Status, "Up") {
			statsID = c.ID
			break
		}
	}
	if statsID == "" {
		t.Skip("No running container available for stats")
	}
	out, err := GatherContainerStats(t.Context(), statsID)
	if err != nil {
		t.Skipf("GatherContainerStats() error: %v", err)
	}
	t.Logf("Container %s: CPU=%.1f%% Memory=%d/%d PIDs=%d",
		statsID, out.Stats.CPUPercent, out.Stats.MemoryUsage,
		out.Stats.MemoryLimit, out.Stats.PIDs)
}

func TestGatherContainerTop(t *testing.T) {
	containers, err := ListDockerContainers(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	var topID string
	for _, c := range containers {
		if strings.Contains(c.Status, "Up") {
			topID = c.ID
			break
		}
	}
	if topID == "" {
		t.Skip("No running container available for top")
	}
	out, err := GatherContainerTop(t.Context(), topID, nil)
	if err != nil {
		t.Skipf("GatherContainerTop() error: %v", err)
	}
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
	containers, err := ListDockerContainers(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	if len(containers) == 0 {
		t.Skip("No containers available for diff")
	}
	out, err := GatherContainerDiff(t.Context(), containers[0].ID)
	if err != nil {
		t.Skipf("GatherContainerDiff() error: %v", err)
	}
	t.Logf(
		"Container %s: %d filesystem changes",
		containers[0].ID,
		len(out.Changes),
	)
}

func TestGatherImageHistory(t *testing.T) {
	images, err := ListDockerImages(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	if len(images) == 0 {
		t.Skip("No images available")
	}
	out, err := GatherImageHistory(t.Context(), images[0].ID)
	if err != nil {
		t.Skipf("GatherImageHistory() error: %v", err)
	}
	if len(out.Layers) == 0 {
		t.Error("Layers should not be empty")
	}
	t.Logf("Image %s: %d layers", images[0].ID, len(out.Layers))
}

func TestGatherImageDetail(t *testing.T) {
	images, err := ListDockerImages(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	if len(images) == 0 {
		t.Skip("No images available")
	}
	out, err := GatherImageDetail(t.Context(), images[0].ID)
	if err != nil {
		t.Skipf("GatherImageDetail() error: %v", err)
	}
	if out.Image.ID == "" {
		t.Error("Image ID should not be empty")
	}
	t.Logf("Image %s: arch=%s os=%s size=%s layers=%d",
		out.Image.ID, out.Image.Architecture, out.Image.OS,
		out.Image.Size, len(out.Image.Layers))
}

func TestGatherDockerNetworks(t *testing.T) {
	out, err := GatherDockerNetworks(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	if len(out.Networks) == 0 {
		t.Error("Networks should not be empty on a Docker host")
	}
	t.Logf("Found %d Docker networks", len(out.Networks))
}

func TestGatherDockerVolumes(t *testing.T) {
	out, err := GatherDockerVolumes(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	t.Logf("Found %d Docker volumes", len(out.Volumes))
}

func TestGatherDockerSystemInfo(t *testing.T) {
	out, err := GatherDockerSystemInfo(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	if out.Info.ServerVersion == "" {
		t.Error("ServerVersion should not be empty")
	}
	t.Logf("Docker %s (%s/%s) driver=%s cgroups=%s runtimes=%v",
		out.Info.ServerVersion, out.Info.Architecture, out.Info.OSType,
		out.Info.Driver, out.Info.CgroupDriver, out.Info.Runtimes)
}

func TestGatherDockerDiskUsage(t *testing.T) {
	out, err := GatherDockerDiskUsage(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
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

func TestGatherSystemSnapshot(t *testing.T) {
	ctx := t.Context()
	result, snapshot, err := HandleGetSystemSnapshot(
		ctx,
		nil,
		GetSystemSnapshotInput{},
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
		t.Log("Temperature data unavailable (sensors may not be accessible)")
	}
	t.Logf("Snapshot errors: %v", snapshot.Errors)
}

func TestGatherSystemSnapshotErrors(t *testing.T) {
	result, snapshot, err := HandleGetSystemSnapshot(
		t.Context(),
		nil,
		GetSystemSnapshotInput{},
	)
	if err != nil {
		t.Fatalf("get_system_snapshot error: %v", err)
	}
	if result != nil {
		t.Error("CallToolResult should be nil")
	}
	t.Logf("Snapshot errors: %v", snapshot.Errors)
}

func TestGatherJournalLogs(t *testing.T) {
	out, err := GatherJournalLogs(t.Context(), "", "", "", "", 5, false)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("journalctl not installed")
		}
		t.Fatalf("GatherJournalLogs() error: %v", err)
	}
	t.Logf("Found %d journal entries", len(out.Entries))
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

func TestGatherInodeUsageWithFilter(t *testing.T) {
	out, err := GatherInodeUsage(t.Context(), "/")
	if err != nil {
		t.Skipf("GatherInodeUsage(\"/\") error: %v", err)
	}
	if len(out.Mounts) == 0 {
		t.Fatal("Expected at least / mount")
	}
	for _, m := range out.Mounts {
		if m.MountedOn != "/" {
			t.Errorf("Expected mounted_on /, got %s", m.MountedOn)
		}
	}
}

func TestGatherListeningPorts(t *testing.T) {
	out, err := GatherListeningPorts(t.Context(), "")
	if err != nil {
		t.Skipf("GatherListeningPorts() error: %v", err)
	}
	t.Logf("Found %d listening ports", len(out.Ports))
}

func TestGatherServiceStatus(t *testing.T) {
	services := []string{
		"systemd-journald.service", "dbus.service", "sshd",
	}
	var out ServiceStatusOutput
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

func TestGatherTopIOProcesses(t *testing.T) {
	out, err := GatherTopIOProcesses(t.Context(), 5)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			t.Skip("pidstat not installed")
		}
		t.Fatalf("GatherTopIOProcesses() error: %v", err)
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

func TestGatherGPUInfo(t *testing.T) {
	out, err := GatherGPUInfo(t.Context())
	if err != nil {
		t.Skipf("No GPU tool found: %v", err)
	}
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

func TestGatherLoadAverage(t *testing.T) {
	out, err := GatherLoadAverage(t.Context())
	if err != nil {
		t.Skipf("GatherLoadAverage() error: %v", err)
	}
	if out.Load1 < 0 {
		t.Errorf("Load1 should be >= 0, got %f", out.Load1)
	}
	if out.Load5 < 0 {
		t.Errorf("Load5 should be >= 0, got %f", out.Load5)
	}
	if out.Load15 < 0 {
		t.Errorf("Load15 should be >= 0, got %f", out.Load15)
	}
}

func TestGatherLoggedInUsers(t *testing.T) {
	out, err := GatherLoggedInUsers(t.Context())
	if err != nil {
		t.Skipf("GatherLoggedInUsers() error: %v", err)
	}
	t.Logf("Found %d logged-in users", len(out.Users))
}

func TestGatherDNSResolve(t *testing.T) {
	out, err := GatherDNSResolve(t.Context(), "localhost")
	if err != nil {
		t.Skipf("GatherDNSResolve(\"localhost\") error: %v", err)
	}
	if len(out.Addresses) == 0 {
		t.Error("Expected at least one address for localhost")
	}
	t.Logf("localhost resolves to: %v", out.Addresses)
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

func TestGatherMountOptionsWithFilter(t *testing.T) {
	out, err := GatherMountOptions(t.Context(), "/")
	if err != nil {
		t.Skipf("GatherMountOptions(\"/\") error: %v", err)
	}
	if len(out.Mounts) == 0 {
		t.Fatal("Expected at least / mount")
	}
	for _, m := range out.Mounts {
		if m.Target != "/" {
			t.Errorf("Expected target /, got %s", m.Target)
		}
	}
}

func TestGatherSystemdUnits(t *testing.T) {
	out, err := GatherSystemdUnits(t.Context())
	if err != nil {
		t.Skipf("GatherSystemdUnits() error: %v", err)
	}
	if len(out.Units) == 0 {
		t.Error("Units should not be empty")
	}
	t.Logf("Found %d systemd units", len(out.Units))
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

func TestGatherManPage(t *testing.T) {
	out, err := GatherManPage(t.Context(), "ls", 500, true, "", 0, 0)
	if err != nil {
		t.Skipf("GatherManPage() error: %v", err)
	}
	if out.Command != "ls" {
		t.Errorf("expected command 'ls', got %q", out.Command)
	}
	if out.Content == "" {
		t.Error("Content should not be empty")
	}
	if !strings.Contains(out.Content, "LS") {
		t.Error("Content should contain 'LS' as man page header")
	}
}

func TestGatherManPageNotFound(t *testing.T) {
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
}

func TestGatherManPageMaxLines(t *testing.T) {
	out, err := GatherManPage(t.Context(), "ls", 5, true, "", 0, 0)
	if err != nil {
		t.Skipf("GatherManPage() error: %v", err)
	}
	lines := strings.Count(out.Content, "\n") + 1
	if lines > 5 {
		t.Errorf("expected at most 5 lines, got %d", lines)
	}
	if out.Truncated != true {
		t.Error("expected Truncated to be true")
	}
}

func TestGatherManPageCleanSpecialChars(t *testing.T) {
	out, err := GatherManPage(t.Context(), "ls", 500, true, "", 0, 0)
	if err != nil {
		t.Skipf("GatherManPage() error: %v", err)
	}
	if !strings.Contains(out.Content, "\x08") {
		return
	}
	t.Error("Expected backspace characters to be cleaned")
}

func TestGatherManPageSearch(t *testing.T) {
	out, err := GatherManPage(t.Context(), "ls", 500, true, "FILE", 0, 0)
	if err != nil {
		t.Skipf("GatherManPage() error: %v", err)
	}
	if out.Content == "" {
		t.Error("search results should not be empty when term exists")
	}
	if !strings.Contains(out.Content, "FILE") {
		t.Error("search results should contain the search term")
	}
}

func TestGatherManPageSearchNotFound(t *testing.T) {
	out, err := GatherManPage(
		t.Context(),
		"ls",
		500,
		true,
		"XYZZY_NONEXISTENT_123",
		0,
		0,
	)
	if err != nil {
		t.Skipf("GatherManPage() error: %v", err)
	}
	if out.Content != "" {
		t.Error("expected empty content for search with no matches")
	}
}

func TestGatherManPageOffset(t *testing.T) {
	out, err := GatherManPage(t.Context(), "ls", 10, true, "", 0, 0)
	if err != nil {
		t.Skipf("GatherManPage() error: %v", err)
	}
	out2, err2 := GatherManPage(t.Context(), "ls", 10, true, "", 0, 10)
	if err2 != nil {
		t.Skipf("GatherManPage() error: %v", err2)
	}
	if out.Content == out2.Content {
		t.Error("offset 0 and offset 10 should return different content")
	}
}

func TestGatherManPageContextLines(t *testing.T) {
	out, err := GatherManPage(t.Context(), "ls", 500, true, "FILE", 2, 0)
	if err != nil {
		t.Skipf("GatherManPage() error: %v", err)
	}
	if out.Content == "" {
		t.Error("should return content with context lines")
	}
}

func TestHandleManPageEmptyCommand(t *testing.T) {
	_, _, err := HandleGetManPage(t.Context(), nil, GetManPageInput{})
	if err == nil {
		t.Error("expected error for empty command")
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
