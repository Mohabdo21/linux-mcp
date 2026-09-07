package tools

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Mohabdo21/linux-mcp/config"
)

type coverageOutput struct {
	Value string
	OutputErrors
}

func TestHandleToolCallDisabledTool(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(
		cfg,
		[]byte(`{"disabled":["coverage_disabled_tool"]}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINUX_MCP_CONFIG", cfg)
	if err := config.Load(); err != nil {
		t.Fatal(err)
	}

	called := false
	_, out, err := handleToolCall(
		context.Background(),
		"coverage_disabled_tool",
		0,
		func(context.Context) (*coverageOutput, error) {
			called = true
			return &coverageOutput{}, nil
		},
	)
	if err == nil || err.Error() != "tool disabled by configuration" {
		t.Fatalf("want disabled error, got %v", err)
	}
	if called {
		t.Fatal("gather must not run for a disabled tool")
	}
	if out != nil {
		t.Fatalf("want nil output, got %+v", out)
	}
}

func TestHandleToolCallSuccess(t *testing.T) {
	_, out, err := handleToolCall(context.Background(), "coverage_ok_tool", 0,
		func(context.Context) (*coverageOutput, error) {
			return &coverageOutput{Value: "data"}, nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == nil || out.Value != "data" {
		t.Fatalf("want output data, got %+v", out)
	}
	if out.ErrorCount() != 0 {
		t.Fatalf("want no errors, got %v", out.Errors)
	}
}

func TestHandleToolCallGatherErrorDegrades(t *testing.T) {
	_, out, err := handleToolCall(context.Background(), "coverage_ok_tool", 0,
		func(context.Context) (*coverageOutput, error) {
			return &coverageOutput{
				Value: "partial",
			}, errors.New(
				"gather failed",
			)
		})
	if err != nil {
		t.Fatalf("error must degrade into output, got %v", err)
	}
	if out == nil || out.Value != "partial" {
		t.Fatalf("want partial output, got %+v", out)
	}
	if out.ErrorCount() != 1 || out.Errors[0] != "gather failed" {
		t.Fatalf("want gather failed recorded, got %v", out.Errors)
	}
}

func TestHandleToolCallNilOutputWithError(t *testing.T) {
	_, out, err := handleToolCall(context.Background(), "coverage_ok_tool", 0,
		func(context.Context) (*coverageOutput, error) {
			return nil, errors.New("boom")
		})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("want gather error surfaced, got %v", err)
	}
	if out != nil {
		t.Fatalf("want nil output, got %+v", out)
	}
}

func TestHandleToolCallNilOutputNilError(t *testing.T) {
	_, out, err := handleToolCall(context.Background(), "coverage_ok_tool", 0,
		func(context.Context) (*coverageOutput, error) {
			return nil, nil
		})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != nil {
		t.Fatalf("want nil output, got %+v", out)
	}
}

func TestIsASCIIHostChar(t *testing.T) {
	for _, r := range "aZ9.-" {
		if !isASCIIHostChar(r) {
			t.Errorf("isASCIIHostChar(%q) = false, want true", r)
		}
	}
	for _, r := range "_ /é" {
		if isASCIIHostChar(r) {
			t.Errorf("isASCIIHostChar(%q) = true, want false", r)
		}
	}
}

func TestValidHost(t *testing.T) {
	valid := []string{
		"example.com",
		"a",
		"exa-mple.com",
		"8.8.8.8",
		"2001:db8::1",
	}
	for _, h := range valid {
		if !validHost(h) {
			t.Errorf("validHost(%q) = false, want true", h)
		}
	}
	invalid := []string{
		"",
		"-example.com",
		"example-.com",
		"example..com",
		"example_com",
		"exa mple.com",
		"example.cöm",
		strings.Repeat("a", 254),
		strings.Repeat("a", 64) + ".com",
	}
	for _, h := range invalid {
		if validHost(h) {
			t.Errorf("validHost(%q) = true, want false", h)
		}
	}
}

func TestIsValidUnitName(t *testing.T) {
	valid := []string{
		"",
		"sshd.service",
		"nginx@.service",
		"systemd-journald@.socket",
	}
	for _, u := range valid {
		if !isValidUnitName(u) {
			t.Errorf("isValidUnitName(%q) = false, want true", u)
		}
	}
	invalid := []string{"my unit", "foo{bar}", "foo?bar", "foo\rbar"}
	for _, u := range invalid {
		if isValidUnitName(u) {
			t.Errorf("isValidUnitName(%q) = true, want false", u)
		}
	}
}

func TestParseDpkgLOutput(t *testing.T) {
	output := `Desired=Unknown/Install/Remove/Purge/Hold
|/ Status?/Err?=...
||/ Name           Version     Architecture Description
+++-==============-===========-============-=================================
ii  adduser        3.134       all          add and remove users and groups
un  some-pkg       <none>      <none>       (no description available)
ii  bash           5.2.15-1    amd64        GNU Bourne Again SHell
`
	want := &InstalledPackagesOutput{
		Packages: []InstalledPackage{
			{Name: "adduser", Version: "3.134"},
			{Name: "bash", Version: "5.2.15-1"},
		},
		Total: 2,
	}
	if got := parseDpkgLOutput(output); !reflect.DeepEqual(got, want) {
		t.Fatalf("parseDpkgLOutput() = %+v, want %+v", got, want)
	}
}

func TestParseDpkgLOutputSkipsShortLines(t *testing.T) {
	if got := parseDpkgLOutput("ii  bash\nx\nii nopkg\n"); got.Total != 0 {
		t.Fatalf("want no packages, got %+v", got)
	}
}

func TestParsePacmanQuOutput(t *testing.T) {
	output := `glibc 2.39-8 -> 2.39-9
libfoo 1.1-2

  trailing-indented  2.0-1
`
	want := &CheckUpdatesOutput{
		Updates: []AvailableUpdate{
			{Name: "glibc", Current: "2.39-8", New: "2.39-9"},
			{Name: "libfoo", Current: "", New: "1.1-2"},
			{Name: "trailing-indented", Current: "", New: "2.0-1"},
		},
		Total: 3,
	}
	if got := parsePacmanQuOutput(output); !reflect.DeepEqual(got, want) {
		t.Fatalf("parsePacmanQuOutput() = %+v, want %+v", got, want)
	}
}

func TestParseASN(t *testing.T) {
	cases := map[string]string{
		"AS24940 Hetzner Online": "AS24940",
		"AS15169":                "AS15169",
		"":                       "",
		"NotAnAsn 1234":          "NotAnAsn",
	}
	for in, want := range cases {
		if got := parseASN(in); got != want {
			t.Errorf("parseASN(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDetectServiceTags(t *testing.T) {
	cases := []struct {
		isp, org, asn string
		want          []string
	}{
		{"Hetzner Online GmbH", "", "AS24940", []string{"Hetzner"}},
		{"Google LLC", "Google Fiber", "AS15169", []string{"Google Cloud"}},
		{"Amazon", "AWS", "AS16509", []string{"AWS"}},
		{"Oracle", "Oracle Cloud", "AS31898", []string{"Oracle Cloud"}},
		{"Local ISP", "AnyOrg", "AS12345", nil},
	}
	for _, c := range cases {
		if got := detectServiceTags(
			c.isp,
			c.org,
			c.asn,
		); !reflect.DeepEqual(
			got,
			c.want,
		) {
			t.Errorf(
				"detectServiceTags(%q,%q,%q) = %v, want %v",
				c.isp,
				c.org,
				c.asn,
				got,
				c.want,
			)
		}
	}
}

func TestTristateKind(t *testing.T) {
	cases := map[uint8]string{
		0: "modified",
		1: "added",
		2: "deleted",
		3: "unknown(3)",
	}
	for kind, want := range cases {
		if got := tristateKind(kind); got != want {
			t.Errorf("tristateKind(%d) = %q, want %q", kind, got, want)
		}
	}
}

func TestClassifyFD(t *testing.T) {
	cases := map[string]string{
		"socket:[12345]":     "socket",
		"pipe:[54321]":       "pipe",
		"anon_inode:inotify": "anon_inode",
		"/usr/lib/libc.so.6": "file",
		"":                   "file",
	}
	for target, want := range cases {
		if got := classifyFD(target); got != want {
			t.Errorf("classifyFD(%q) = %q, want %q", target, got, want)
		}
	}
}

func TestParseProcStatIncludesCurrentProcess(t *testing.T) {
	procs, err := parseProcStat()
	if err != nil {
		t.Skipf("cannot read /proc: %v", err)
	}
	if len(procs) == 0 {
		t.Fatal("parseProcStat() returned no processes")
	}
	if _, ok := procs[int32(os.Getpid())]; !ok {
		t.Error("parseProcStat() missing the current test process")
	}
}
