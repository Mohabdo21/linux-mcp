package tools

import (
	"testing"
)

func TestParseTimespan(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"6.455s", 6.455},
		{"973ms", 0.973},
		{"886us", 0.000886},
		{"88ms", 0.088},
		{"2.521s", 2.521},
		{"1min 9.608s", 69.608},
		{"32.119s", 32.119},
	}
	for _, c := range cases {
		got, err := parseTimespan(c.in)
		if err != nil {
			t.Errorf("parseTimespan(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("parseTimespan(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseTimespanInvalid(t *testing.T) {
	for _, in := range []string{"", "abc", "5", "5x", "1min 9"} {
		if _, err := parseTimespan(in); err == nil {
			t.Errorf("parseTimespan(%q) expected error", in)
		}
	}
}

func TestParseBootBlame(t *testing.T) {
	output := `6.455s dev-tpmrm0.device
2.685s systemd-tpm2-setup-early.service
 973ms systemd-userdbd.service
 886us sshd-unix-local.socket
  57us gpg-agent@etc-pacman.d-gnupg.socket
`
	entries := parseBootBlame(output)
	if len(entries) != 5 {
		t.Fatalf("Expected 5 entries, got %d", len(entries))
	}
	if entries[0].Unit != "dev-tpmrm0.device" {
		t.Errorf("Unit = %q, want dev-tpmrm0.device", entries[0].Unit)
	}
	if entries[0].TimeSeconds != 6.455 {
		t.Errorf("TimeSeconds = %v, want 6.455", entries[0].TimeSeconds)
	}
	if entries[1].Unit != "systemd-tpm2-setup-early.service" {
		t.Errorf(
			"Unit = %q, want systemd-tpm2-setup-early.service",
			entries[1].Unit,
		)
	}
	if entries[2].TimeSeconds != 0.973 {
		t.Errorf("TimeSeconds = %v, want 0.973", entries[2].TimeSeconds)
	}
	if entries[4].Unit != "gpg-agent@etc-pacman.d-gnupg.socket" {
		t.Errorf(
			"Unit = %q, want gpg-agent@etc-pacman.d-gnupg.socket",
			entries[4].Unit,
		)
	}
}

func TestParseBootBlameSkipsNonParsableLines(t *testing.T) {
	output := `6.455s dev-tpmrm0.device
not a parseable line
`
	entries := parseBootBlame(output)
	if len(entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(entries))
	}
}

func TestParseBootCriticalChain(t *testing.T) {
	output := `The time when unit became active or started is printed after the "@" character.
The time the unit took to start is printed after the "+" character.

graphical.target @7.484s
└─power-profiles-daemon.service @7.395s +88ms
  └─multi-user.target @7.392s
    └─docker.service @6.737s +654ms
      └─containerd.service @6.367s +368ms
        └─dev-tpmrm0.device
`
	chain := parseBootCriticalChain(output)
	if len(chain) != 6 {
		t.Fatalf("Expected 6 links, got %d", len(chain))
	}
	if chain[0].Unit != "graphical.target" || chain[0].Depth != 0 {
		t.Errorf("Root = %+v, want graphical.target at depth 0", chain[0])
	}
	if chain[0].ActiveSeconds != 7.484 {
		t.Errorf("Root ActiveSeconds = %v, want 7.484", chain[0].ActiveSeconds)
	}
	if chain[1].Unit != "power-profiles-daemon.service" || chain[1].Depth != 1 {
		t.Errorf(
			"Link[1] = %+v, want power-profiles-daemon.service at depth 1",
			chain[1],
		)
	}
	if chain[1].StartSeconds != 0.088 {
		t.Errorf("Link[1] StartSeconds = %v, want 0.088", chain[1].StartSeconds)
	}
	if chain[3].Unit != "docker.service" || chain[3].Depth != 3 {
		t.Errorf(
			"Link[3] = %+v, want docker.service at depth 3",
			chain[3],
		)
	}
	if chain[5].Unit != "dev-tpmrm0.device" {
		t.Errorf("Link[5] = %+v, want dev-tpmrm0.device", chain[5])
	}
	if chain[5].ActiveTime != "" || chain[5].StartTime != "" {
		t.Error("Leaf should have no timestamps")
	}
}

func TestParseBootTime(t *testing.T) {
	output := `Startup finished in 12.067s (firmware) + 7.428s (loader) + 1.338s (kernel) + 3.800s (initrd) + 7.484s (userspace) = 32.119s
graphical.target reached after 7.484s in userspace.
`
	result := parseBootTime(output)
	if len(result.Phases) != 5 {
		t.Fatalf("Expected 5 phases, got %d", len(result.Phases))
	}
	if result.Phases[0].Name != "firmware" ||
		result.Phases[0].Seconds != 12.067 {
		t.Errorf("Phase[0] = %+v, want firmware 12.067s", result.Phases[0])
	}
	if result.Phases[4].Name != "userspace" ||
		result.Phases[4].Seconds != 7.484 {
		t.Errorf("Phase[4] = %+v, want userspace 7.484s", result.Phases[4])
	}
	if result.Total != "32.119s" || result.TotalSeconds != 32.119 {
		t.Errorf(
			"Total = %q %v, want 32.119s 32.119",
			result.Total,
			result.TotalSeconds,
		)
	}
	if result.Target != "graphical.target" {
		t.Errorf("Target = %q, want graphical.target", result.Target)
	}
	if result.TargetReachedSeconds != 7.484 {
		t.Errorf(
			"TargetReachedSeconds = %v, want 7.484",
			result.TargetReachedSeconds,
		)
	}
}

func TestParseBootTimeContainerVariant(t *testing.T) {
	output := `Startup finished in 296ms (userspace)
multi-user.target reached after 275ms in userspace
`
	result := parseBootTime(output)
	if len(result.Phases) != 1 {
		t.Fatalf("Expected 1 phase, got %d", len(result.Phases))
	}
	if result.Phases[0].Name != "userspace" ||
		result.Phases[0].Seconds != 0.296 {
		t.Errorf("Phase[0] = %+v, want userspace 0.296s", result.Phases[0])
	}
	if result.Total != "" {
		t.Errorf("Total = %q, want empty for single phase", result.Total)
	}
	if result.Target != "multi-user.target" {
		t.Errorf("Target = %q, want multi-user.target", result.Target)
	}
}

func TestGatherBootTime(t *testing.T) {
	out, err := GatherBootTime(t.Context())
	skipOnErr(t, err, "GatherBootTime() error: %v", err)
	if len(out.Phases) == 0 {
		t.Error("Phases should not be empty")
	}
	t.Logf("Phases: %d, total: %s, target: %s",
		len(out.Phases), out.Total, out.Target)
}

func TestGatherBootBlame(t *testing.T) {
	out, err := GatherBootBlame(t.Context())
	skipOnErr(t, err, "GatherBootBlame() error: %v", err)
	if len(out.Entries) == 0 {
		t.Error("Entries should not be empty")
	}
	t.Logf("Found %d blame entries", len(out.Entries))
}

func TestGatherBootCriticalChain(t *testing.T) {
	out, err := GatherBootCriticalChain(t.Context(), "")
	skipOnErr(t, err, "GatherBootCriticalChain() error: %v", err)
	if len(out.Chain) == 0 {
		t.Error("Chain should not be empty")
	}
	t.Logf("Chain: %d links, target: %s", len(out.Chain), out.Target)
}
