package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakeSystemctl puts a systemctl stub first on PATH and returns the file the
// stub creates when it is run.
func fakeSystemctl(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	marker := filepath.Join(dir, "called")
	if err := os.WriteFile(
		filepath.Join(dir, "systemctl"),
		[]byte(`#!/bin/sh
printf '%s\n' "$@" > "$SERVICE_TEST_ARGS"
echo 'Active: active (running)'
exit "${SYSTEMCTL_TEST_EXIT:-0}"
`),
		0o755,
	); err != nil {
		t.Fatalf("write systemctl stub: %v", err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("SERVICE_TEST_ARGS", marker)
	t.Setenv("SYSTEMCTL_TEST_EXIT", "0")
	return marker
}

func TestGatherServiceStatusRejectsInvalidName(t *testing.T) {
	for _, user := range []bool{false, true} {
		for _, name := range []string{
			"", "-x", "ssh d", "a/b", "a\x00b", "a\nb", "café.service",
		} {
			t.Run(fmt.Sprintf("%s/user=%t", name, user), func(t *testing.T) {
				marker := fakeSystemctl(t)
				out, err := GatherServiceStatus(t.Context(), name, user)
				if out != nil || err == nil ||
					err.Error() != fmt.Sprintf(
						"invalid service name: %q",
						name,
					) {
					t.Errorf("GatherServiceStatus(%q) = %+v, %v, "+
						"want an invalid service name error", name, out, err)
				}
				if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
					t.Errorf(
						"expected no systemctl invocation, stat error: %v",
						statErr,
					)
				}
			})
		}
	}
}

func TestGetServiceStatusAcceptsUnitNames(t *testing.T) {
	policyConfig(t)
	for _, name := range []string{
		"sshd", "nginx.service", "getty@tty1.service", "foo@bar@baz.service",
		`foo\x2dbar.service`, `\x2dfoo.service`, "_foo.service", ".foo.service", "foo:bar.service",
	} {
		t.Run(name, func(t *testing.T) {
			fakeSystemctl(t)
			_, out, err := HandleGetServiceStatus(
				t.Context(),
				nil,
				GetServiceStatusInput{Name: name},
			)
			if err != nil {
				t.Fatalf(
					"HandleGetServiceStatus() error = %v, want nil",
					err,
				)
			}
			if out == nil || out.Name != name ||
				out.Active != "active (running)" {
				t.Fatalf("unexpected service output: %+v", out)
			}
		})
	}
}

func TestGatherServiceStatusArguments(t *testing.T) {
	for _, user := range []bool{false, true} {
		t.Run(fmt.Sprintf("user=%t", user), func(t *testing.T) {
			marker := fakeSystemctl(t)
			if _, err := GatherServiceStatus(
				t.Context(),
				"nginx.service",
				user,
			); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(marker)
			if err != nil {
				t.Fatal(err)
			}
			want := "status\n--no-pager\n-l\n--\nnginx.service\n"
			if user {
				want = "--user\n" + want
			}
			if string(raw) != want {
				t.Errorf("systemctl args = %q, want %q", raw, want)
			}
		})
	}
}

func TestServiceResourceRejectsOptionLikeName(t *testing.T) {
	policyConfig(t)
	for _, path := range []string{"-x", "%2Dx"} {
		t.Run(path, func(t *testing.T) {
			marker := fakeSystemctl(t)
			req := &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{URI: scheme("service/" + path)},
			}
			result, err := handleReadResource(t.Context(), req)
			if result != nil || err == nil ||
				!strings.Contains(err.Error(), `invalid service name: "-x"`) {
				t.Fatalf(
					"expected invalid service name error, got %+v, %v",
					result,
					err,
				)
			}
			if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
				t.Fatalf(
					"expected no systemctl invocation, stat error: %v",
					statErr,
				)
			}
		})
	}
}

func TestServiceResourceDecodesUnitName(t *testing.T) {
	policyConfig(t)
	marker := fakeSystemctl(t)
	req := &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: scheme("service/foo%40bar%5Cx2dbaz.service"),
		},
	}
	result, err := handleReadResource(t.Context(), req)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || len(result.Contents) != 1 ||
		result.Contents[0].Text == "null" {
		t.Fatalf("unexpected resource result: %+v", result)
	}
	raw, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(raw), "--\nfoo@bar\\x2dbaz.service\n") {
		t.Fatalf("systemctl args = %q", raw)
	}
}

func TestServiceStatusPreservesCommandError(t *testing.T) {
	policyConfig(t)
	fakeSystemctl(t)
	t.Setenv("SYSTEMCTL_TEST_EXIT", "3")
	out, err := GatherServiceStatus(t.Context(), "nginx.service", false)
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 3 ||
		out == nil ||
		out.Output == "" {
		t.Fatalf("expected partial output and exit 3, got %+v, %v", out, err)
	}
	_, out, err = HandleGetServiceStatus(
		t.Context(),
		nil,
		GetServiceStatusInput{Name: "nginx.service"},
	)
	if err != nil || out == nil || len(out.Errors) != 1 ||
		out.Errors[0] != "exit status 3" {
		t.Fatalf("expected command error in tool output, got %+v, %v", out, err)
	}
}
