package tools

import (
	"testing"
)

func TestParseProcSoftIRQs(t *testing.T) {
	input := `                    CPU0       CPU1       CPU2
          HI:          0          2          1
       TIMER:    1320007    1261876    1807666
      NET_TX:         34          8         56
`
	result := parseProcSoftIRQs(input)
	if len(result) == 0 {
		t.Fatal("Expected non-empty result")
	}
	// Verify types are parsed correctly (not CPU names)
	types := make(map[string]bool)
	for _, r := range result {
		types[r.Type] = true
		if r.Total == 0 && r.Type != "NET_TX" {
			t.Errorf("Type %s has zero total", r.Type)
		}
	}
	for _, want := range []string{"HI", "TIMER", "NET_TX"} {
		if !types[want] {
			t.Errorf("Missing expected type %s", want)
		}
	}
	// Verify TIMER total is sum across CPUs
	for _, r := range result {
		if r.Type == "TIMER" {
			want := uint64(1320007 + 1261876 + 1807666)
			if r.Total != want {
				t.Errorf("TIMER total = %d, want %d", r.Total, want)
			}
		}
	}
}

func TestParseProcSoftIRQsSkipsHeader(t *testing.T) {
	input := `                    CPU0       CPU1
          HI:          1          2
`
	result := parseProcSoftIRQs(input)
	for _, r := range result {
		if r.Type == "CPU0" || r.Type == "CPU1" {
			t.Errorf("Should not parse CPU header as IRQ type: %s", r.Type)
		}
	}
}

func TestParseProcVMStat(t *testing.T) {
	input := `nr_free_pages 787502
nr_active_anon 679667
pgfault 526989650
`
	result := parseProcVMStat(input)
	if result["nr_free_pages"] != 787502 {
		t.Errorf("nr_free_pages = %d, want 787502", result["nr_free_pages"])
	}
	if result["nr_active_anon"] != 679667 {
		t.Errorf("nr_active_anon = %d, want 679667", result["nr_active_anon"])
	}
	if result["pgfault"] != 526989650 {
		t.Errorf("pgfault = %d, want 526989650", result["pgfault"])
	}
}

func TestParseProcDiskStats(t *testing.T) {
	input := ` 259    0 nvme0n1 1291768 31271 98565555 414833977 2792332 92594 112448603 1104171482 0 309992 1289763515
 259    1 nvme0n1p1 414 1138 11742 53 12 0 10 1 0 23 54
`
	result := parseProcDiskStats(input)
	if len(result) != 2 {
		t.Fatalf("Expected 2 devices, got %d", len(result))
	}
	if result[0].Device != "nvme0n1" {
		t.Errorf("Device = %q, want nvme0n1", result[0].Device)
	}
	if result[0].ReadsCompleted != 1291768 {
		t.Errorf("ReadsCompleted = %d, want 1291768", result[0].ReadsCompleted)
	}
	if result[0].WritesCompleted != 2792332 {
		t.Errorf(
			"WritesCompleted = %d, want 2792332",
			result[0].WritesCompleted,
		)
	}
}

func TestParseProcDiskStatsSkipsLoop(t *testing.T) {
	input := `   7    0 loop0 100 0 800 10 50 0 400 5 0 15 20
 259    0 nvme0n1 1291768 31271 98565555 414833977 2792332 92594 112448603 1104171482 0 309992 1289763515
`
	result := parseProcDiskStats(input)
	if len(result) != 1 {
		t.Fatalf("Expected 1 device (loop skipped), got %d", len(result))
	}
	if result[0].Device != "nvme0n1" {
		t.Errorf("Device = %q, want nvme0n1", result[0].Device)
	}
}

func TestParseProcFilesystems(t *testing.T) {
	input := `nodev	sysfs
nodev	tmpfs
nodev	proc
	ext4
	vfat
`
	result := parseProcFilesystems(input)
	if len(result) != 5 {
		t.Fatalf("Expected 5 filesystems, got %d", len(result))
	}
	if !result[0].Nodev {
		t.Error("sysfs should be nodev")
	}
	if result[3].Nodev {
		t.Error("ext4 should not be nodev")
	}
	if result[3].Name != "ext4" {
		t.Errorf("Name = %q, want ext4", result[3].Name)
	}
}

func TestParseProcSlabinfo(t *testing.T) {
	input := `# name            <active_objs> <num_objs> <objsize> <objperslab> <pagesperslab>
ext4_groupinfo_4k     1024  1024   144   28    1
dentry                 512   512   192   21    1
`
	result := parseProcSlabinfo(input)
	if len(result) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(result))
	}
	if result[0].Name != "ext4_groupinfo_4k" {
		t.Errorf("Name = %q, want ext4_groupinfo_4k", result[0].Name)
	}
	if result[0].ActiveObjs != 1024 {
		t.Errorf("ActiveObjs = %d, want 1024", result[0].ActiveObjs)
	}
	// Sorted by active_objs descending
	if result[0].ActiveObjs < result[1].ActiveObjs {
		t.Error("Results should be sorted by active_objs descending")
	}
}

func TestParseProcInterrupts(t *testing.T) {
	input := `           CPU0       CPU1
  18:      1000      2000   IO-APIC-edge      timer
 159:        10         5   PCI-MSI-edge      nvme0q0
`
	result := parseProcInterrupts(input)
	if len(result) == 0 {
		t.Fatal("Expected non-empty result")
	}
	// Should be sorted by total, highest first
	foundTimer := false
	for _, r := range result {
		if r == "18 timer" {
			foundTimer = true
		}
	}
	if !foundTimer {
		t.Error("Expected to find IRQ 18 timer")
	}
}

func TestParseSSHDConfig(t *testing.T) {
	// Test with actual sshd_config
	vals := parseSSHDConfig("/etc/ssh/sshd_config")
	if vals == nil {
		t.Skip("Cannot read /etc/ssh/sshd_config")
	}
	// Should have Include directive processed and drop-in files merged
	if len(vals) == 0 {
		t.Error("Expected non-empty config values")
	}
}

func TestParseLoginDefs(t *testing.T) {
	vals := parseLoginDefs("/etc/login.defs")
	if vals == nil {
		t.Skip("Cannot read /etc/login.defs")
	}
	if _, ok := vals["PASS_MAX_DAYS"]; !ok {
		t.Error("Expected PASS_MAX_DAYS in login.defs")
	}
}

func TestComputeSecurityScore(t *testing.T) {
	// Perfect config should score high
	perfect := &SecurityAuditOutput{
		Firewall: FirewallInfo{Active: true},
		SSHHardening: SSHHardeningInfo{
			ConfigPresent:          true,
			PermitRootLogin:        "no",
			PasswordAuthentication: "no",
		},
		SUIDBinaries:  []string{},
		WorldWritable: []string{},
		Umask:         "027",
		PasswordPolicy: PasswordPolicyInfo{
			PassMaxDays: "90",
		},
	}
	score := computeSecurityScore(perfect)
	if score < 80 {
		t.Errorf("Perfect config score = %d, want >= 80", score)
	}

	// Bad config should score low
	bad := &SecurityAuditOutput{
		Firewall: FirewallInfo{Active: false},
		SSHHardening: SSHHardeningInfo{
			ConfigPresent:          true,
			PermitRootLogin:        "yes",
			PasswordAuthentication: "yes",
		},
		SUIDBinaries: []string{
			"a",
			"b",
			"c",
			"d",
			"e",
			"f",
			"g",
			"h",
			"i",
			"j",
			"k",
		},
		WorldWritable: []string{"bad.conf"},
		Umask:         "0022",
		PasswordPolicy: PasswordPolicyInfo{
			PassMaxDays: "99999",
		},
	}
	badScore := computeSecurityScore(bad)
	if badScore >= 50 {
		t.Errorf("Bad config score = %d, want < 50", badScore)
	}
}

func TestGatherProcDiagnostics(t *testing.T) {
	t.Run("AllSections", func(t *testing.T) {
		out, err := GatherProcDiagnostics(t.Context(), "")
		if err != nil {
			t.Skipf("GatherProcDiagnostics() error: %v", err)
		}
		if len(out.Interrupts) == 0 {
			t.Error("Interrupts should not be empty")
		}
		if len(out.SoftIRQs) == 0 {
			t.Error("SoftIRQs should not be empty")
		}
		if out.VMStat == nil || len(out.VMStat.Metrics) == 0 {
			t.Error("VMStat should not be empty")
		}
		if len(out.DiskStats) == 0 {
			t.Error("DiskStats should not be empty")
		}
		if len(out.Filesystems) == 0 {
			t.Error("Filesystems should not be empty")
		}
		if out.Version == "" {
			t.Error("Version should not be empty")
		}
		t.Logf(
			"Sections: interrupts=%d softirqs=%d vmstat=%d diskstats=%d filesystems=%d",
			len(out.Interrupts),
			len(out.SoftIRQs),
			len(out.VMStat.Metrics),
			len(out.DiskStats),
			len(out.Filesystems),
		)
	})

	t.Run("FilteredSections", func(t *testing.T) {
		out, err := GatherProcDiagnostics(t.Context(), "version,vmstat")
		if err != nil {
			t.Skipf("GatherProcDiagnostics() error: %v", err)
		}
		if out.Version == "" {
			t.Error("Version should not be empty")
		}
		if out.VMStat == nil || len(out.VMStat.Metrics) == 0 {
			t.Error("VMStat should not be empty")
		}
		if len(out.Interrupts) != 0 {
			t.Error("Interrupts should be empty when not requested")
		}
	})
}

func TestGatherSecurityAudit(t *testing.T) {
	out, err := GatherSecurityAudit(t.Context())
	if err != nil {
		t.Skipf("GatherSecurityAudit() error: %v", err)
	}
	if out.Score < 0 || out.Score > 100 {
		t.Errorf("Score = %d, want 0-100", out.Score)
	}
	if out.Umask == "" {
		t.Error("Umask should not be empty")
	}
	t.Logf("Security score: %d, Firewall: %v, Umask: %s",
		out.Score, out.Firewall.Active, out.Umask)
}

func TestGatherDiskIOMetrics(t *testing.T) {
	out, err := GatherDiskIOMetrics(t.Context())
	if err != nil {
		t.Skipf("GatherDiskIOMetrics() error: %v", err)
	}
	if len(out.Metrics) == 0 {
		t.Error("Metrics should not be empty")
	}
	for i, m := range out.Metrics {
		if m.Device == "" {
			t.Errorf("Metrics[%d].Device should not be empty", i)
		}
		if m.ReadsCompleted == 0 && m.WritesCompleted == 0 {
			t.Errorf("Metrics[%d] has no I/O activity", i)
		}
	}
	t.Logf("Found %d devices with I/O metrics", len(out.Metrics))
}
