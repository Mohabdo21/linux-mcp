package tools

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDetectPkgManagerAPK(t *testing.T) {
	fakePkgManager(t, "apk", "exit 0")

	if got := detectPkgManager(); got != "apk" {
		t.Errorf("detectPkgManager() = %q, want %q", got, "apk")
	}
}

func TestParseApkListOutput(t *testing.T) {
	raw := "alpine-baselayout-3.7.2-r1 x86_64 {alpine-baselayout} (GPL-2.0-only) [installed]\n" +
		"busybox-1.36.1-r20 x86_64 {busybox} (GPL-2.0-only) [installed]\n" +
		"musl-1.2.4_git20230717-r5 x86_64 {musl} (MIT) [installed]\n"

	out := parseApkListOutput(raw)

	want := []InstalledPackage{
		{Name: "alpine-baselayout", Version: "3.7.2-r1"},
		{Name: "busybox", Version: "1.36.1-r20"},
		{Name: "musl", Version: "1.2.4_git20230717-r5"},
	}
	if !reflect.DeepEqual(out.Packages, want) {
		t.Errorf("parseApkListOutput() = %+v, want %+v", out.Packages, want)
	}
	if out.Total != len(want) {
		t.Errorf("Total = %d, want %d", out.Total, len(want))
	}
}

// Names may contain -<digits> themselves, so the version cannot be found by
// splitting on the last -rN.
func TestParseApkListOutputNameContainsDigits(t *testing.T) {
	raw := "xf86-video-r128-6.13.0-r0 x86_64 {xf86-video} (MIT) [installed]\n" +
		"liblsp-r3d-glx-lib-1.2.25-r0 x86_64 {liblsp} (GPL-2.0-only) [installed]\n"

	out := parseApkListOutput(raw)

	want := []InstalledPackage{
		{Name: "xf86-video-r128", Version: "6.13.0-r0"},
		{Name: "liblsp-r3d-glx-lib", Version: "1.2.25-r0"},
	}
	if !reflect.DeepEqual(out.Packages, want) {
		t.Errorf("parseApkListOutput() = %+v, want %+v", out.Packages, want)
	}
}

func TestParseApkListOutputEmpty(t *testing.T) {
	out := parseApkListOutput("")

	if out.Total != 0 {
		t.Errorf("Total = %d, want 0", out.Total)
	}
	if out.Packages == nil {
		t.Error(
			"Packages = nil, want an empty slice so it marshals as [] not null",
		)
	}
}

func TestGatherInstalledPackagesAPK(t *testing.T) {
	fakePkgManager(
		t,
		"apk",
		`printf 'busybox-1.36.1-r20 x86_64 {busybox} (GPL-2.0-only) [installed]\n'`,
	)

	out, err := GatherInstalledPackages(t.Context(), "")
	if err != nil {
		t.Fatalf("GatherInstalledPackages() error = %v, want nil", err)
	}
	if out.Total != 1 {
		t.Fatalf("Total = %d, want 1", out.Total)
	}
	if out.Packages[0].Name != "busybox" ||
		out.Packages[0].Version != "1.36.1-r20" {
		t.Errorf(
			"Packages[0] = %+v, want {busybox 1.36.1-r20}",
			out.Packages[0],
		)
	}
}

func TestGatherInstalledPackagesAPKNameFilter(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	stub := "printf '%s\\n' \"$@\" > \"$ARGS_FILE\"\n" +
		"printf 'php83-common-8.3.4-r0 x86_64 {php83} (PHP-8.3) [installed]\n'"
	if err := os.WriteFile(
		filepath.Join(dir, "apk"),
		[]byte("#!/bin/sh\n"+stub),
		0o755,
	); err != nil {
		t.Fatalf("write apk stub: %v", err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("ARGS_FILE", argsFile)

	out, err := GatherInstalledPackages(t.Context(), "php83")
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
	if !strings.Contains(got, "--installed") {
		t.Errorf("apk args = %q, want --installed", got)
	}
	// apk patterns are globs, so a bare name matches nothing.
	if !strings.Contains(got, "php83*") {
		t.Errorf(
			"apk args = %q, want the name filter as a glob so it is not literal",
			got,
		)
	}
}

func TestParseApkVersionOutput(t *testing.T) {
	raw := "Installed:                                Available:\n" +
		"busybox-1.36.1-r20                      < 1.36.1-r21 \n" +
		"busybox-binsh-1.36.1-r20                < 1.36.1-r21 \n" +
		"musl-utils-1.2.4_git20230717-r5         < 1.2.4_git20230717-r6 \n"

	out := parseApkVersionOutput(raw)

	want := []AvailableUpdate{
		{Name: "busybox", Current: "1.36.1-r20", New: "1.36.1-r21"},
		{Name: "busybox-binsh", Current: "1.36.1-r20", New: "1.36.1-r21"},
		{
			Name:    "musl-utils",
			Current: "1.2.4_git20230717-r5",
			New:     "1.2.4_git20230717-r6",
		},
	}
	if !reflect.DeepEqual(out.Updates, want) {
		t.Errorf("parseApkVersionOutput() = %+v, want %+v", out.Updates, want)
	}
	if out.Total != len(want) {
		t.Errorf("Total = %d, want %d", out.Total, len(want))
	}
}

// The header and the WARNING apk prints when its index cache is cold both
// appear on stdout and neither is an update.
func TestParseApkVersionOutputHeaderAndWarningsOnly(t *testing.T) {
	raw := "WARNING: opening from cache https://dl-cdn.alpinelinux.org/alpine/v3.19/main\n" +
		"Installed:                                Available:\n"

	out := parseApkVersionOutput(raw)

	if out.Total != 0 {
		t.Errorf("Total = %d, want 0", out.Total)
	}
	if out.Updates == nil {
		t.Error(
			"Updates = nil, want an empty slice so it marshals as [] not null",
		)
	}
}

func TestParseApkVersionOutputNameContainsDigits(t *testing.T) {
	raw := "Installed: Available:\n" +
		"xf86-video-r128-6.13.0-r0               < 6.13.1-r0 \n"

	out := parseApkVersionOutput(raw)

	want := []AvailableUpdate{
		{Name: "xf86-video-r128", Current: "6.13.0-r0", New: "6.13.1-r0"},
	}
	if !reflect.DeepEqual(out.Updates, want) {
		t.Errorf("parseApkVersionOutput() = %+v, want %+v", out.Updates, want)
	}
}

func TestGatherAvailableUpdatesAPK(t *testing.T) {
	fakePkgManager(
		t,
		"apk",
		"printf 'Installed: Available:\\n';\n"+
			"printf 'busybox-1.36.1-r20 < 1.36.1-r21 \\n'",
	)

	out, err := GatherAvailableUpdates(t.Context())
	if err != nil {
		t.Fatalf("GatherAvailableUpdates() error = %v, want nil", err)
	}
	if out.Total != 1 {
		t.Fatalf("Total = %d, want 1", out.Total)
	}
	if out.Updates[0].Name != "busybox" {
		t.Errorf(
			"Updates[0].Name = %q, want %q",
			out.Updates[0].Name,
			"busybox",
		)
	}
	if out.Updates[0].Current != "1.36.1-r20" {
		t.Errorf(
			"Updates[0].Current = %q, want %q",
			out.Updates[0].Current,
			"1.36.1-r20",
		)
	}
}

// apk version exits 0 whether or not updates exist, so a non-zero exit is a
// real failure.
func TestGatherAvailableUpdatesAPKError(t *testing.T) {
	fakePkgManager(t, "apk", "exit 1")

	if _, err := GatherAvailableUpdates(t.Context()); err == nil {
		t.Fatal(
			"GatherAvailableUpdates() error = nil, want a failure for apk exit 1",
		)
	}
}

func TestGatherAvailableUpdatesAPKNoUpdates(t *testing.T) {
	fakePkgManager(t, "apk", "printf 'Installed: Available:\\n'")

	out, err := GatherAvailableUpdates(t.Context())
	if err != nil {
		t.Fatalf("GatherAvailableUpdates() error = %v, want nil", err)
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
