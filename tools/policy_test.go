package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func policyConfig(t *testing.T, disabled ...string) func(...string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("LINUX_MCP_CONFIG", path)
	t.Cleanup(func() {
		if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
			t.Error(err)
		}
		if err := config.Load(); err != nil {
			t.Error(err)
		}
	})
	write := func(names ...string) {
		t.Helper()
		data, err := json.Marshal(config.Config{Disabled: names})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(disabled...)
	if err := config.Load(); err != nil {
		t.Fatal(err)
	}
	return func(names ...string) {
		write(names...)
		if err := config.Reload(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPolicyResourcesAtRequestTime(t *testing.T) {
	reload := policyConfig(t)
	original := append([]resourceDef(nil), resourceRegistry...)
	t.Cleanup(func() { resourceRegistry = original })
	calls := 0
	for i := range resourceRegistry {
		resourceRegistry[i].resolver = func(context.Context, string) (any, error) {
			calls++
			return map[string]string{"value": "enabled"}, nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	server := mcp.NewServer(
		&mcp.Implementation{Name: "policy", Version: "test"},
		nil,
	)
	RegisterResources(server)
	mcp.AddTool(
		server,
		&mcp.Tool{Name: config.ToolNameGetSystemInfo},
		HandleGetSystemInfo,
	)
	st, ct := mcp.NewInMemoryTransports()
	ss, err := server.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := ss.Close(); err != nil {
			t.Error(err)
		}
	})
	client := mcp.NewClient(
		&mcp.Implementation{Name: "policy", Version: "test"},
		nil,
	)
	cs, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := cs.Close(); err != nil {
			t.Error(err)
		}
	})
	cases := []struct{ path, tool string }{
		{"info", config.ToolNameGetSystemInfo},
		{"cpu", config.ToolNameGetCPUInfo},
		{"memory", config.ToolNameGetMemoryInfo},
		{"disk", config.ToolNameGetDiskInfo},
		{"network", config.ToolNameGetNetworkInfo},
		{"load", config.ToolNameGetLoadAverage},
		{"temperature", config.ToolNameGetCPUTemperature},
		{"gpu", config.ToolNameGetGPUInfo},
		{"logged_in_users", config.ToolNameGetLoggedInUsers},
		{"listening_ports", config.ToolNameGetNetworkConnections},
		{"failed_logins", config.ToolNameGetFailedLogins},
		{"block_devices", config.ToolNameGetBlockDevices},
		{"raid", config.ToolNameGetRAIDStatus},
		{"time_sync", config.ToolNameGetTimeSyncStatus},
		{"selinux_apparmor", config.ToolNameGetSELinuxAppArmorStatus},
		{"logrotate", config.ToolNameGetLogrotateStatus},
		{"health", config.ToolNameGetSystemHealthCheck},
		{"disk/boot", config.ToolNameGetDiskInfo},
		{"service/sshd", config.ToolNameGetServiceStatus},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			reload(tc.tool)
			before := calls
			result, err := cs.ReadResource(
				ctx,
				&mcp.ReadResourceParams{URI: scheme(tc.path)},
			)
			if err == nil ||
				!strings.Contains(err.Error(), "disabled by configuration") {
				t.Errorf(
					"disabled resource returned result=%+v err=%v",
					result,
					err,
				)
			}
			if calls != before {
				t.Error("disabled resource invoked its collector")
			}
			reload()
			if _, err := cs.ReadResource(
				ctx,
				&mcp.ReadResourceParams{URI: scheme(tc.path)},
			); err != nil {
				t.Fatal(err)
			}
			if calls != before+1 {
				t.Error(
					"enabled resource did not invoke its collector exactly once",
				)
			}
		})
	}
	reload(config.ToolNameGetSystemInfo)
	result, err := cs.CallTool(
		ctx,
		&mcp.CallToolParams{Name: config.ToolNameGetSystemInfo},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Fatalf("disabled tool did not report an error: %+v", result)
	}
}

func TestPolicyCollectOrFallbackSkipsDisabled(t *testing.T) {
	policyConfig(t, config.ToolNameGetSystemInfo)
	called := false
	var errs []string
	out := collectOrFallback(context.Background(), config.ToolNameGetSystemInfo,
		func(context.Context) (*SystemInfoOutput, error) {
			called = true
			return &SystemInfoOutput{Hostname: "secret"}, nil
		}, SystemInfoOutput{}, &errs)
	if called || out.Hostname != "" {
		t.Fatal("disabled collector ran or leaked output")
	}
	if len(errs) != 1 ||
		!strings.Contains(
			errs[0],
			config.ToolNameGetSystemInfo+": tool disabled by configuration",
		) {
		t.Fatalf("missing omission report: %v", errs)
	}
	out = collectOrFallback(context.Background(), config.ToolNameGetMemoryInfo,
		func(context.Context) (*SystemInfoOutput, error) {
			return &SystemInfoOutput{Hostname: "enabled"}, nil
		}, SystemInfoOutput{}, &errs)
	if out.Hostname != "enabled" || len(errs) != 1 {
		t.Fatalf("enabled constituent not preserved: %+v errors=%v", out, errs)
	}
}

func TestPolicySnapshotSkipsDisabled(t *testing.T) {
	disabled := []string{
		config.ToolNameGetSystemInfo,
		config.ToolNameGetCPUInfo,
		config.ToolNameGetCPUTemperature,
		config.ToolNameGetMemoryInfo,
		config.ToolNameGetDiskInfo,
		config.ToolNameGetNetworkInfo,
		config.ToolNameGetProcessInfo,
		config.ToolNameGetDockerInfo,
		config.ToolNameGetLoadAverage,
	}
	policyConfig(t, disabled...)
	_, out, err := HandleGetSystemSnapshot(context.Background(), nil, NoArgs{})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range disabled {
		if !strings.Contains(
			strings.Join(out.Errors, ";"),
			name+": tool disabled by configuration",
		) {
			t.Errorf("missing omission report for %s: %v", name, out.Errors)
		}
	}
	if out.System.Hostname != "" || out.CPU.PhysicalCoreCount != 0 ||
		len(out.Processes.Processes) != 0 {
		t.Error("snapshot exposes disabled data")
	}
}

func TestPolicyHealthSkipsDisabled(t *testing.T) {
	dir := t.TempDir()
	calls := filepath.Join(dir, "systemctl.calls")
	script := "#!/bin/sh\nprintf 'called\\n' >> \"$POLICY_SYSTEMCTL_CALLS\"\n" +
		"printf '%s\\n' 'example.service loaded failed failed Example'\n"
	if err := os.WriteFile(
		filepath.Join(dir, "systemctl"),
		[]byte(script),
		0o700,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("POLICY_SYSTEMCTL_CALLS", calls)
	disabled := []string{
		config.ToolNameGetMemoryInfo,
		config.ToolNameGetDiskInfo,
		config.ToolNameGetLoadAverage,
		config.ToolNameGetCPUInfo,
		config.ToolNameGetSystemdUnits,
	}
	reload := policyConfig(t, disabled...)
	out, err := GatherSystemHealthCheck(context.Background())
	if err == nil {
		t.Error("omissions must be reported")
	}
	for _, name := range disabled {
		if !strings.Contains(
			strings.Join(out.Errors, ";"),
			name+": tool disabled by configuration",
		) {
			t.Errorf("missing omission report for %s: %v", name, out.Errors)
		}
	}
	if len(out.Checks) != 0 || out.Overall != "UNKNOWN" {
		t.Errorf("omitted checks must report UNKNOWN: %+v", out)
	}
	if _, err := os.Stat(calls); !os.IsNotExist(err) {
		t.Fatalf("disabled systemctl collector executed: stat error=%v", err)
	}
	reload(disabled[:len(disabled)-1]...)
	out, err = GatherSystemHealthCheck(context.Background())
	if err == nil || len(out.Errors) != 4 {
		t.Fatalf("expected four omissions: out=%+v err=%v", out, err)
	}
	if out.Overall != "CRITICAL" || len(out.Checks) != 1 ||
		out.Checks[0].Component != "systemd_failed" ||
		out.Checks[0].Detail != "example.service" {
		t.Fatalf("enabled systemd collector was not preserved: %+v", out)
	}
	invocations, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	if string(invocations) != "called\n" {
		t.Fatalf("want exactly one systemctl invocation, got %q", invocations)
	}
}
