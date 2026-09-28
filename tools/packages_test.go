package tools

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
)

// Points PATH at a temp dir holding an executable stub.
func fakePkgManager(t *testing.T, name, script string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(dir, name),
		[]byte("#!/bin/sh\n"+script+"\n"),
		0o755,
	); err != nil {
		t.Fatalf("write %s stub: %v", name, err)
	}
	t.Setenv("PATH", dir)
}

func TestDetectPkgManagerRPM(t *testing.T) {
	fakePkgManager(t, "rpm", "exit 0")

	if got := detectPkgManager(); got != "rpm" {
		t.Errorf("detectPkgManager() = %q, want %q", got, "rpm")
	}
}

func TestDetectPkgManagerDNF(t *testing.T) {
	fakePkgManager(t, "dnf", "exit 0")

	if got := detectPkgManager(); got != "rpm" {
		t.Errorf(
			"detectPkgManager() = %q, want %q, dnf hosts are rpm based",
			got,
			"rpm",
		)
	}
}

func TestGatherInstalledPackagesRPM(t *testing.T) {
	fakePkgManager(
		t,
		"rpm",
		`printf 'bash 5.2.26-3.el9\nvim-enhanced 2:9.0.2120-1.el9\n'`,
	)

	out, err := GatherInstalledPackages(t.Context(), "")
	if err != nil {
		t.Fatalf("GatherInstalledPackages() error = %v, want nil", err)
	}
	if out.Total != 2 {
		t.Fatalf("Total = %d, want 2", out.Total)
	}
	if out.Packages[0].Name != "bash" ||
		out.Packages[0].Version != "5.2.26-3.el9" {
		t.Errorf("Packages[0] = %+v, want {bash 5.2.26-3.el9}", out.Packages[0])
	}
	if out.Packages[1].Name != "vim-enhanced" {
		t.Errorf(
			"Packages[1].Name = %q, want %q",
			out.Packages[1].Name,
			"vim-enhanced",
		)
	}
}

func TestGatherInstalledPackagesRPMNameFilter(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	stub := "printf '%s\\n' \"$@\" > \"$ARGS_FILE\"\nprintf 'bash 5.2.26-3.el9\\n'\n"
	if err := os.WriteFile(
		filepath.Join(dir, "rpm"),
		[]byte("#!/bin/sh\n"+stub),
		0o755,
	); err != nil {
		t.Fatalf("write rpm stub: %v", err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("ARGS_FILE", argsFile)

	out, err := GatherInstalledPackages(t.Context(), "bash")
	if err != nil {
		t.Fatalf("GatherInstalledPackages() error = %v, want nil", err)
	}
	if out.Total != 1 {
		t.Fatalf("Total = %d, want 1", out.Total)
	}

	raw, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read recorded args: %v", err)
	}
	got := string(raw)
	if !strings.Contains(got, "--qf") {
		t.Errorf("rpm args = %q, want a --qf query format", got)
	}
	if !strings.Contains(got, `name="bash*"`) {
		t.Errorf(
			"rpm args = %q, want the name filter passed as a name= selector",
			got,
		)
	}
}

// pacman -Qs follows each package line with an indented description.
func TestParsePacmanQOutputSearch(t *testing.T) {
	raw := "local/alsa-lib 1.2.16.1-1\n" +
		"    An alternative implementation of Linux sound support\n" +
		"local/alsa-utils 1.2.16.1-1\n" +
		"    Advanced Linux Sound Architecture - Utilities\n" +
		"local/apparmor 4.1.7-1\n" +
		"    Mandatory Access Control (MAC) using Linux Security Module (LSM)\n"

	out := parsePacmanQOutput(raw)

	want := []InstalledPackage{
		{Name: "alsa-lib", Version: "1.2.16.1-1"},
		{Name: "alsa-utils", Version: "1.2.16.1-1"},
		{Name: "apparmor", Version: "4.1.7-1"},
	}
	if !reflect.DeepEqual(out.Packages, want) {
		t.Errorf("parsePacmanQOutput() = %+v, want %+v", out.Packages, want)
	}
	if out.Total != len(want) {
		t.Errorf(
			"Total = %d, want %d: description lines must not count as packages",
			out.Total,
			len(want),
		)
	}
}

// pacman -Q emits no description lines.
func TestParsePacmanQOutputList(t *testing.T) {
	raw := "local/alsa-lib 1.2.16.1-1\nlocal/apparmor 4.1.7-1\n"

	out := parsePacmanQOutput(raw)

	want := []InstalledPackage{
		{Name: "alsa-lib", Version: "1.2.16.1-1"},
		{Name: "apparmor", Version: "4.1.7-1"},
	}
	if !reflect.DeepEqual(out.Packages, want) {
		t.Errorf("parsePacmanQOutput() = %+v, want %+v", out.Packages, want)
	}
	if out.Total != 2 {
		t.Errorf("Total = %d, want 2", out.Total)
	}
}

func TestParseRpmQOutput(t *testing.T) {
	out := parseRpmQOutput("bash 5.2.26-3.el9\nvim-enhanced 2:9.0.2120-1.el9\n")

	if out.Total != 2 {
		t.Fatalf("Total = %d, want 2", out.Total)
	}
	if out.Packages[0].Name != "bash" {
		t.Errorf("Packages[0].Name = %q, want %q", out.Packages[0].Name, "bash")
	}
	if out.Packages[1].Version != "2:9.0.2120-1.el9" {
		t.Errorf(
			"Packages[1].Version = %q, want %q",
			out.Packages[1].Version,
			"2:9.0.2120-1.el9",
		)
	}
}

func TestParseRpmQOutputEmpty(t *testing.T) {
	out := parseRpmQOutput("")

	if out.Total != 0 {
		t.Errorf("Total = %d, want 0", out.Total)
	}
	if out.Packages == nil {
		t.Error(
			"Packages = nil, want an empty slice so it marshals as [] not null",
		)
	}
}

func TestParseDnfCheckUpdateOutput(t *testing.T) {
	raw := "Last metadata expiration check: 0:00:12 ago on Tue 29 Sep 2026.\n" +
		"\n" +
		"bash.x86_64              5.2.26-3.el9            baseos\n" +
		"kernel.x86_64            6.6.5-200.fc40          updates\n" +
		"python3.11.x86_64        3.11.9-1.fc40           @System\n"

	out := parseDnfCheckUpdateOutput(raw)
	if out.Total != 3 {
		t.Fatalf(
			"Total = %d, want 3, the metadata line must not count as an update",
			out.Total,
		)
	}
	if out.Updates[0].Name != "bash" {
		t.Errorf(
			"Updates[0].Name = %q, want %q with the arch stripped",
			out.Updates[0].Name,
			"bash",
		)
	}
	if out.Updates[0].New != "5.2.26-3.el9" {
		t.Errorf(
			"Updates[0].New = %q, want %q",
			out.Updates[0].New,
			"5.2.26-3.el9",
		)
	}
	if out.Updates[1].Name != "kernel" {
		t.Errorf("Updates[1].Name = %q, want %q", out.Updates[1].Name, "kernel")
	}
	// The arch is the last dot-separated field, never the first: a package name
	// may itself contain a dot.
	if out.Updates[2].Name != "python3.11" {
		t.Errorf(
			"Updates[2].Name = %q, want %q",
			out.Updates[2].Name,
			"python3.11",
		)
	}
}

func TestGatherAvailableUpdatesDNFNoUpdates(t *testing.T) {
	fakePkgManager(t, "dnf", "exit 0")

	out, err := GatherAvailableUpdates(t.Context())
	if err != nil {
		t.Fatalf(
			"GatherAvailableUpdates() error = %v, want nil on dnf exit 0",
			err,
		)
	}
	if out.Total != 0 {
		t.Errorf("Total = %d, want 0", out.Total)
	}
	if out.Updates == nil {
		t.Error(
			"Updates = nil, want an empty slice so it marshals as [] not null",
		)
	}
}

func TestGatherAvailableUpdatesDNFUpdatesAvailable(t *testing.T) {
	fakePkgManager(
		t,
		"dnf",
		"printf 'bash.x86_64   5.2.26-3.el9   baseos\\n'\nexit 100",
	)

	out, err := GatherAvailableUpdates(t.Context())
	if err != nil {
		t.Fatalf(
			"GatherAvailableUpdates() error = %v, want nil: dnf exits 100 to signal updates",
			err,
		)
	}
	if out.Total != 1 {
		t.Fatalf("Total = %d, want 1", out.Total)
	}
	if out.Updates[0].Name != "bash" {
		t.Errorf("Updates[0].Name = %q, want %q", out.Updates[0].Name, "bash")
	}
}

func TestGatherAvailableUpdatesDNFError(t *testing.T) {
	fakePkgManager(t, "dnf", "exit 1")

	if _, err := GatherAvailableUpdates(t.Context()); err == nil {
		t.Fatal(
			"GatherAvailableUpdates() error = nil, want a failure for dnf exit 1",
		)
	}
}

// check_updates is a read-only query, so it must be served as
// get_available_updates and the old name must be gone.
func TestAvailableUpdatesToolName(t *testing.T) {
	var found bool
	for _, td := range toolRegistry {
		switch td.Name {
		case "check_updates":
			t.Error("toolRegistry still serves the old name check_updates")
		case "get_available_updates":
			found = true
		}
	}
	if !found {
		t.Error("toolRegistry does not serve get_available_updates")
	}
}

// Pin the value: ToolTimeout falls back to 30s for an unknown name, so a
// stale map key after the rename would pass silently.
func TestAvailableUpdatesToolTimeoutIsConfigured(t *testing.T) {
	if got := config.ToolTimeout(
		"get_available_updates",
		0,
	); got != 15*time.Second {
		t.Errorf("ToolTimeout(get_available_updates) = %v, want 15s", got)
	}
}
