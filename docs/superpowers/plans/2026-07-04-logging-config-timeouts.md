# Logging, Config, and Timeouts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add structured logging (slog to stderr), JSON config file with validation, and context-based timeouts to linux-mcp with zero new dependencies.

**Architecture:** New `config` package for config loading/validation/thread-safe access. Shared helpers in `tools/util.go`. Each handler wraps context with configured timeout, checks disabled list, logs duration.

**Tech Stack:** Go 1.26.4, stdlib `log/slog`, `sync/atomic`, `encoding/json`

## Global Constraints

- Zero new dependencies beyond Go stdlib
- Config at `~/.config/linux-mcp/config.json`, overridable by `LINUX_MCP_CONFIG` env
- SIGHUP reloads config without restart
- All timeouts are `time.Duration` strings parsed via `time.ParseDuration`
- Logging to stderr only (stdout reserved for MCP JSON-RPC)
- Disabled tools return `errors.New("tool disabled by configuration")`
- All `Gather*` functions accept `ctx context.Context` for signature consistency
- Follow existing code style (no comments, 80-char line limit, `strings.SplitSeq`)

---

### Task 1: Config Package

**Files:**

- Create: `config/config.go`
- Create: `config/config_schema.json`
- Create: `config/config_test.go`

**Interfaces:**

- Consumes: Nothing (first task)
- Produces: `config.Load()` / `config.Reload()` / `config.Get() Config` / `config.ToolTimeout(name string, fallback time.Duration) time.Duration` / `config.IsDisabled(name string) bool`

- [ ] **Step 1: Create config/config.go**

```go
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

type Config struct {
	LogLevel string            `json:"log_level"`
	Timeouts map[string]string `json:"timeouts"`
	Disabled []string          `json:"disabled"`
}

var (
	current  atomic.Value
	filePath string
)

func envPath() string {
	if p, ok := os.LookupEnv("LINUX_MCP_CONFIG"); ok {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "linux-mcp", "config.json")
}

func defaultConfig() *Config {
	return &Config{
		LogLevel: "info",
		Timeouts: map[string]string{
			"get_cpu_info":           "5s",
			"get_cpu_temperature":    "5s",
			"get_memory_info":        "5s",
			"get_disk_info":          "10s",
			"get_inode_usage":        "10s",
			"get_network_info":       "5s",
			"get_process_info":       "10s",
			"get_docker_info":        "10s",
			"get_system_snapshot":    "120s",
			"get_journal_logs":       "20s",
			"get_listening_ports":    "10s",
			"get_service_status":     "10s",
			"get_top_io_processes":   "15s",
			"get_failed_logins":      "10s",
			"get_gpu_info":           "5s",
			"get_largest_files":      "30s",
			"ping_host":              "10s",
			"get_installed_packages": "15s",
			"check_updates":          "15s",
			"get_load_average":       "5s",
			"get_logged_in_users":    "5s",
			"resolve_dns":            "10s",
			"get_mount_options":      "10s",
			"get_systemd_units":      "10s",
		},
		Disabled: []string{},
	}
}

func validate(c *Config) error {
	levels := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	if _, ok := levels[c.LogLevel]; !ok {
		return fmt.Errorf("invalid log_level %q", c.LogLevel)
	}
	for name, d := range c.Timeouts {
		if _, err := time.ParseDuration(d); err != nil {
			return fmt.Errorf("invalid timeout for %q: %w", name, err)
		}
	}
	return nil
}

func loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	if cfg.Timeouts == nil {
		cfg.Timeouts = map[string]string{}
	}
	if cfg.Disabled == nil {
		cfg.Disabled = []string{}
	}
	return &cfg, nil
}

func Load() error {
	return loadAtPath(envPath())
}

func loadAtPath(path string) error {
	if path == "" {
		current.Store(defaultConfig())
		return nil
	}
	filePath = path
	cfg, err := loadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			current.Store(defaultConfig())
			return nil
		}
		return err
	}
	current.Store(cfg)
	return nil
}

func Reload() error {
	if filePath == "" {
		return nil
	}
	cfg, err := loadFile(filePath)
	if err != nil {
		return err
	}
	current.Store(cfg)
	return nil
}

func Get() Config {
	if cfg, ok := current.Load().(*Config); ok {
		return *cfg
	}
	return *defaultConfig()
}

func ToolTimeout(name string, fallback time.Duration) time.Duration {
	cfg := Get()
	if s, ok := cfg.Timeouts[name]; ok {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return fallback
}

func IsDisabled(name string) bool {
	cfg := Get()
	for _, n := range cfg.Disabled {
		if n == name {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Create config/config_schema.json**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "linux-mcp Configuration",
  "type": "object",
  "properties": {
    "log_level": {
      "type": "string",
      "enum": ["debug", "info", "warn", "error"],
      "description": "Structured log level (stderr only)"
    },
    "timeouts": {
      "type": "object",
      "description": "Per-tool timeout overrides as Go duration strings",
      "additionalProperties": {
        "type": "string",
        "pattern": "^[0-9]+(ns|us|ms|s|m)$"
      }
    },
    "disabled": {
      "type": "array",
      "description": "Tool names to disable at startup",
      "items": { "type": "string" }
    }
  }
}
```

- [ ] **Step 3: Create config/config_test.go**

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	Load()
	cfg := Get()
	if cfg.LogLevel != "info" {
		t.Errorf("expected info, got %s", cfg.LogLevel)
	}
	if cfg.Timeouts["get_cpu_info"] != "5s" {
		t.Errorf("expected 5s, got %s", cfg.Timeouts["get_cpu_info"])
	}
	if len(cfg.Disabled) != 0 {
		t.Errorf("expected empty disabled, got %v", cfg.Disabled)
	}
}

func TestToolTimeout(t *testing.T) {
	Load()
	d := ToolTimeout("get_cpu_info", 10*time.Second)
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
	d = ToolTimeout("nonexistent", 10*time.Second)
	if d != 10*time.Second {
		t.Errorf("expected 10s, got %v", d)
	}
}

func TestIsDisabled(t *testing.T) {
	// start with a custom config
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	os.WriteFile(p, []byte(`{"disabled":["ping_host"]}`), 0644)
	if err := loadAtPath(p); err != nil {
		t.Fatal(err)
	}
	if !IsDisabled("ping_host") {
		t.Error("expected ping_host disabled")
	}
	if IsDisabled("get_cpu_info") {
		t.Error("expected get_cpu_info enabled")
	}
}

func TestInvalidLogLevel(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	os.WriteFile(p, []byte(`{"log_level":"trace"}`), 0644)
	err := loadAtPath(p)
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestInvalidTimeout(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	os.WriteFile(p, []byte(`{"timeouts":{"get_cpu_info":"not-a-duration"}}`), 0644)
	err := loadAtPath(p)
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestEnvOverride(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	os.WriteFile(p, []byte(`{"log_level":"debug"}`), 0644)
	t.Setenv("LINUX_MCP_CONFIG", p)
	Load()
	cfg := Get()
	if cfg.LogLevel != "debug" {
		t.Errorf("expected debug, got %s", cfg.LogLevel)
	}
}

func TestMissingConfigIsOk(t *testing.T) {
	loadAtPath("/nonexistent/path/config.json")
	cfg := Get()
	if cfg.LogLevel != "info" {
		t.Errorf("expected info, got %s", cfg.LogLevel)
	}
}

func TestReload(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	os.WriteFile(p, []byte(`{"log_level":"debug"}`), 0644)
	loadAtPath(p)
	if Get().LogLevel != "debug" {
		t.Fatal("expected debug after load")
	}
	os.WriteFile(p, []byte(`{"log_level":"warn"}`), 0644)
	if err := Reload(); err != nil {
		t.Fatal(err)
	}
	if Get().LogLevel != "warn" {
		t.Errorf("expected warn after reload, got %s", Get().LogLevel)
	}
}

func TestToolTimeoutFallback(t *testing.T) {
	Load()
	d := ToolTimeout("get_cpu_info", 10*time.Second)
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
	d = ToolTimeout("nonexistent_tool", 10*time.Second)
	if d != 10*time.Second {
		t.Errorf("expected 10s, got %v", d)
	}
}
```

- [ ] **Step 4: Create config/go.mod** (no - files are in root module; verify `config/config.go` compiles as package config within the `github.com/Mohabdo21/linux-mcp` module)

Run: `go build ./config/` (should succeed)

Note: `config/` is a subdirectory of the existing module `github.com/Mohabdo21/linux-mcp`; no separate go.mod needed.

- [ ] **Step 5: Run config tests**

Run: `go test -v ./config/`
Expected: all tests pass

- [ ] **Step 6: Commit**

```bash
git add config/config.go config/config_schema.json config/config_test.go
git commit -m "feat: add config package with JSON loading, validation, and thread-safe access"
```

---

### Task 2: Logging Setup and Shared Helpers

**Files:**

- Modify: `main.go`
- Modify: `tools/util.go`

**Interfaces:**

- Consumes: `config.Load()` / `config.Get()` / `config.Reload()` from Task 1
- Produces: `tools.WithToolTimeout(ctx, name, fallback)` / `tools.LogToolCall(ctx, tool, dur, errs)` used by all handlers

- [ ] **Step 1: Update main.go**

Replace the entire file:

```go
package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/Mohabdo21/linux-mcp/tools"
)

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func setupLogging() {
	cfg := config.Get()
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	})
	slog.SetDefault(slog.New(handler))
}

func main() {
	if err := config.Load(); err != nil {
		log.Printf("Config load error (using defaults): %v", err)
	}

	setupLogging()

	slog.Info("server starting",
		"version", "1.0.0",
		"log_level", config.Get().LogLevel,
	)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "linux-mcp",
		Version: "1.0.0",
	}, nil)

	tools.RegisterTools(server)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)
	go func() {
		for range sigCh {
			if err := config.Reload(); err != nil {
				slog.Error("config reload failed", "error", err)
			} else {
				setupLogging()
				slog.Info("config reloaded")
			}
		}
	}()

	if err := server.Run(
		context.Background(),
		&mcp.StdioTransport{},
	); err != nil {
		slog.Error("server failed", "error", err)
	}

	slog.Info("server stopped")
}
```

- [ ] **Step 2: Add helpers to tools/util.go**

Append to `tools/util.go` (after existing code, before the closing of the package):

```go
import (
	"log/slog"
	"time"
)

func WithToolTimeout(
	ctx context.Context,
	name string,
	fallback time.Duration,
) (context.Context, context.CancelFunc) {
	return context.WithTimeout(
		ctx,
		config.ToolTimeout(name, fallback),
	)
}

func LogToolCall(
	ctx context.Context,
	tool string,
	dur time.Duration,
	errs int,
) {
	slog.LogAttrs(ctx, slog.LevelInfo, "tool call",
		slog.String("tool", tool),
		slog.Duration("duration", dur),
		slog.Int("errors", errs),
	)
}
```

Update the imports in `tools/util.go` to include:

- `"context"`
- `"github.com/Mohabdo21/linux-mcp/config"`
- `"log/slog"`
- `"time"`

- [ ] **Step 3: Verify it compiles**

Run: `go build ./...`
Expected: builds without errors

- [ ] **Step 4: Commit**

```bash
git add main.go tools/util.go
git commit -m "feat: add slog logging (stderr) and shared timeout helper"
```

---

### Task 3: Context Propagation and Handler Wrappers

**Files:**

- Modify: `tools/cpu.go`
- Modify: `tools/memory.go`
- Modify: `tools/system.go`
- Modify: `tools/disk.go`
- Modify: `tools/network.go`
- Modify: `tools/process.go`
- Modify: `tools/ping.go`
- Modify: `tools/docker.go`
- Modify: `tools/gpu.go`
- Modify: `tools/journal.go`
- Modify: `tools/security.go`
- Modify: `tools/service.go`
- Modify: `tools/packages.go`

**Interfaces:**

- Consumes: `config.IsDisabled(name)` / `config.ToolTimeout(name, fallback)` from Task 1, `WithToolTimeout` / `LogToolCall` from Task 2
- Produces: All handlers now have timeout, logging, and disabled-check

Each handler follows this pattern:

```go
func HandleX(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input XInput,
) (*mcp.CallToolResult, XOutput, error) {
	if config.IsDisabled("tool_name") {
		return nil, XOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "tool_name", defaultTimeout)
	defer cancel()

	start := time.Now()
	out, err := GatherX(ctx, ...)
	LogToolCall(ctx, "tool_name",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Gather* functions that lack `ctx` get it added to their signature even if they don't use it (for interface consistency), and their callers pass it through.

- [ ] **Step 1: Update tools/system.go**

Add `ctx` to GatherSystemInfo and GatherLoadAverage signatures.

GatherSystemInfo change:

```go
// Before:
func GatherSystemInfo() (SystemInfoOutput, error) {
// After:
func GatherSystemInfo(ctx context.Context) (SystemInfoOutput, error) {
```

GatherLoadAverage change:

```go
// Before:
func GatherLoadAverage() (LoadAverageOutput, error) {
// After:
func GatherLoadAverage(ctx context.Context) (LoadAverageOutput, error) {
```

HandleGetSystemInfo:

```go
func HandleGetSystemInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetSystemInfoInput,
) (*mcp.CallToolResult, SystemInfoOutput, error) {
	if config.IsDisabled("get_system_info") {
		return nil, SystemInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_system_info", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherSystemInfo(ctx)
	LogToolCall(ctx, "get_system_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleGetLoadAverage:

```go
func HandleGetLoadAverage(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetLoadAverageInput,
) (*mcp.CallToolResult, LoadAverageOutput, error) {
	if config.IsDisabled("get_load_average") {
		return nil, LoadAverageOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_load_average", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherLoadAverage(ctx)
	LogToolCall(ctx, "get_load_average",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleGetSystemSnapshot - add disabled check, timeout wrapper, and logging:

```go
func HandleGetSystemSnapshot(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetSystemSnapshotInput,
) (*mcp.CallToolResult, SystemSnapshotOutput, error) {
	if config.IsDisabled("get_system_snapshot") {
		return nil, SystemSnapshotOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_system_snapshot", 120*time.Second)
	defer cancel()

	start := time.Now()

	var snapshot SystemSnapshotOutput
	var errs ErrList

	if out, err := GatherSystemInfo(ctx); err == nil {
		snapshot.System = out
	} else {
		errs.Add("system", err)
	}

	// ... (remaining Gather* calls unchanged, but now pass ctx)

	if out, err := GatherCPUInfo(ctx); err == nil {
		snapshot.CPU = out
	} else {
		errs.Add("cpu", err)
	}

	// ... etc for all Gather* calls -> pass ctx

	if out, err := GatherProcessInfo(ctx, "cpu", 10); err == nil {
		snapshot.Processes = out
	} else {
		errs.Add("processes", err)
	}

	if out, err := GatherDockerInfo(ctx); err == nil {
		snapshot.Docker = out
	} else {
		errs.Add("docker", err)
		snapshot.Docker = DockerInfoOutput{}
	}

	snapshot.Errors = errs
	LogToolCall(ctx, "get_system_snapshot",
		time.Since(start), len(errs))
	return nil, snapshot, nil
}
```

Add imports: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 2: Update tools/cpu.go**

Add ctx to GatherCPUInfo and GatherCPUTemperature signatures.

HandleGetCPUInfo:

```go
func HandleGetCPUInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetCPUInfoInput,
) (*mcp.CallToolResult, CPUInfoOutput, error) {
	if config.IsDisabled("get_cpu_info") {
		return nil, CPUInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_cpu_info", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherCPUInfo(ctx)
	LogToolCall(ctx, "get_cpu_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleGetCPUTemperature:

```go
func HandleGetCPUTemperature(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetCPUTemperatureInput,
) (*mcp.CallToolResult, CPUTemperatureOutput, error) {
	if config.IsDisabled("get_cpu_temperature") {
		return nil, CPUTemperatureOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_cpu_temperature", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherCPUTemperature(ctx)
	LogToolCall(ctx, "get_cpu_temperature",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 3: Update tools/memory.go**

Add ctx to GatherMemoryInfo signature.

HandleGetMemoryInfo:

```go
func HandleGetMemoryInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetMemoryInfoInput,
) (*mcp.CallToolResult, MemoryInfoOutput, error) {
	if config.IsDisabled("get_memory_info") {
		return nil, MemoryInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_memory_info", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherMemoryInfo(ctx)
	LogToolCall(ctx, "get_memory_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 4: Update tools/disk.go**

Add ctx to GatherDiskInfo signature.

HandleGetDiskInfo:

```go
func HandleGetDiskInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetDiskInfoInput,
) (*mcp.CallToolResult, DiskInfoOutput, error) {
	if config.IsDisabled("get_disk_info") {
		return nil, DiskInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_disk_info", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherDiskInfo(ctx, input.MountPoint)
	LogToolCall(ctx, "get_disk_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleGetInodeUsage:

```go
func HandleGetInodeUsage(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetInodeUsageInput,
) (*mcp.CallToolResult, InodeUsageOutput, error) {
	if config.IsDisabled("get_inode_usage") {
		return nil, InodeUsageOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_inode_usage", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherInodeUsage(ctx, input.MountPoint)
	LogToolCall(ctx, "get_inode_usage",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleGetLargestFiles:

```go
func HandleGetLargestFiles(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetLargestFilesInput,
) (*mcp.CallToolResult, LargestFilesOutput, error) {
	if config.IsDisabled("get_largest_files") {
		return nil, LargestFilesOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_largest_files", 30*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherLargestFiles(ctx, input.Path, input.Limit)
	LogToolCall(ctx, "get_largest_files",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleGetMountOptions:

```go
func HandleGetMountOptions(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetMountOptionsInput,
) (*mcp.CallToolResult, MountOptionsOutput, error) {
	if config.IsDisabled("get_mount_options") {
		return nil, MountOptionsOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_mount_options", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherMountOptions(ctx, input.MountPoint)
	LogToolCall(ctx, "get_mount_options",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports to disk.go: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 5: Update tools/network.go**

Add ctx to GatherNetworkInfo signature.

HandleGetNetworkInfo:

```go
func HandleGetNetworkInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetNetworkInfoInput,
) (*mcp.CallToolResult, NetworkInfoOutput, error) {
	if config.IsDisabled("get_network_info") {
		return nil, NetworkInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_network_info", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherNetworkInfo(ctx)
	LogToolCall(ctx, "get_network_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleGetListeningPorts:

```go
func HandleGetListeningPorts(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetListeningPortsInput,
) (*mcp.CallToolResult, ListeningPortsOutput, error) {
	if config.IsDisabled("get_listening_ports") {
		return nil, ListeningPortsOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_listening_ports", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherListeningPorts(ctx, input.Protocol)
	LogToolCall(ctx, "get_listening_ports",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleResolveDNS - Add ctx to GatherDNSResolve signature:

```go
func HandleResolveDNS(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ResolveDNSInput,
) (*mcp.CallToolResult, ResolveDNSOutput, error) {
	if config.IsDisabled("resolve_dns") {
		return nil, ResolveDNSOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "resolve_dns", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherDNSResolve(ctx, input.Hostname)
	LogToolCall(ctx, "resolve_dns",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

GatherDNSResolve signature change:

```go
func GatherDNSResolve(ctx context.Context, hostname string) (ResolveDNSOutput, error) {
```

Add imports to network.go: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 6: Update tools/process.go**

Add ctx to GatherProcessInfo signature.

HandleGetProcessInfo:

```go
func HandleGetProcessInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetProcessInfoInput,
) (*mcp.CallToolResult, ProcessInfoOutput, error) {
	if config.IsDisabled("get_process_info") {
		return nil, ProcessInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_process_info", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherProcessInfo(ctx, input.SortBy, input.Limit)
	LogToolCall(ctx, "get_process_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleGetTopIOProcesses:

```go
func HandleGetTopIOProcesses(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetTopIOProcessesInput,
) (*mcp.CallToolResult, TopIOProcessesOutput, error) {
	if config.IsDisabled("get_top_io_processes") {
		return nil, TopIOProcessesOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_top_io_processes", 15*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherTopIOProcesses(ctx, input.Limit)
	LogToolCall(ctx, "get_top_io_processes",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports to process.go: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 7: Update tools/ping.go**

HandlePingHost:

```go
func HandlePingHost(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input PingHostInput,
) (*mcp.CallToolResult, PingOutput, error) {
	if config.IsDisabled("ping_host") {
		return nil, PingOutput{},
			errors.New("tool disabled by configuration")
	}
	if input.Host == "" {
		return nil, PingOutput{}, errors.New("host is required")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "ping_host", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherPing(ctx, input.Host, input.Count, input.Timeout)
	LogToolCall(ctx, "ping_host",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports: `"errors"` (already present), `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 8: Update tools/docker.go**

HandleGetDockerInfo:

```go
func HandleGetDockerInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetDockerInfoInput,
) (*mcp.CallToolResult, DockerInfoOutput, error) {
	if config.IsDisabled("get_docker_info") {
		return nil, DockerInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_docker_info", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherDockerInfo(ctx)
	LogToolCall(ctx, "get_docker_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 9: Update tools/gpu.go**

HandleGetGPUInfo:

```go
func HandleGetGPUInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetGPUInfoInput,
) (*mcp.CallToolResult, GPUInfoOutput, error) {
	if config.IsDisabled("get_gpu_info") {
		return nil, GPUInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_gpu_info", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherGPUInfo(ctx)
	LogToolCall(ctx, "get_gpu_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 10: Update tools/journal.go**

HandleGetJournalLogs:

```go
func HandleGetJournalLogs(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetJournalLogsInput,
) (*mcp.CallToolResult, JournalLogsOutput, error) {
	if config.IsDisabled("get_journal_logs") {
		return nil, JournalLogsOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_journal_logs", 20*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherJournalLogs(
		ctx, input.Unit, input.Priority,
		input.Since, input.Until,
		input.Lines, input.User,
	)
	LogToolCall(ctx, "get_journal_logs",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 11: Update tools/security.go**

HandleGetLoggedInUsers:

```go
func HandleGetLoggedInUsers(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetLoggedInUsersInput,
) (*mcp.CallToolResult, LoggedInUsersOutput, error) {
	if config.IsDisabled("get_logged_in_users") {
		return nil, LoggedInUsersOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_logged_in_users", 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherLoggedInUsers(ctx)
	LogToolCall(ctx, "get_logged_in_users",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleGetFailedLogins:

```go
func HandleGetFailedLogins(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetFailedLoginsInput,
) (*mcp.CallToolResult, FailedLoginsOutput, error) {
	if config.IsDisabled("get_failed_logins") {
		return nil, FailedLoginsOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_failed_logins", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherFailedLogins(ctx, input.Lines)
	LogToolCall(ctx, "get_failed_logins",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports to security.go: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 12: Update tools/service.go**

HandleGetServiceStatus:

```go
func HandleGetServiceStatus(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetServiceStatusInput,
) (*mcp.CallToolResult, ServiceStatusOutput, error) {
	if config.IsDisabled("get_service_status") {
		return nil, ServiceStatusOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_service_status", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherServiceStatus(ctx, input.Name, input.User)
	LogToolCall(ctx, "get_service_status",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleGetSystemdUnits:

```go
func HandleGetSystemdUnits(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetSystemdUnitsInput,
) (*mcp.CallToolResult, SystemdUnitsOutput, error) {
	if config.IsDisabled("get_systemd_units") {
		return nil, SystemdUnitsOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_systemd_units", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherSystemdUnits(ctx)
	LogToolCall(ctx, "get_systemd_units",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports to service.go: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

- [ ] **Step 13: Update tools/packages.go**

HandleGetInstalledPackages:

```go
func HandleGetInstalledPackages(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetInstalledPackagesInput,
) (*mcp.CallToolResult, InstalledPackagesOutput, error) {
	if config.IsDisabled("get_installed_packages") {
		return nil, InstalledPackagesOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_installed_packages", 15*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherInstalledPackages(ctx, input.Name)
	LogToolCall(ctx, "get_installed_packages",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

HandleCheckUpdates:

```go
func HandleCheckUpdates(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ CheckUpdatesInput,
) (*mcp.CallToolResult, CheckUpdatesOutput, error) {
	if config.IsDisabled("check_updates") {
		return nil, CheckUpdatesOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "check_updates", 15*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherCheckUpdates(ctx)
	LogToolCall(ctx, "check_updates",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
```

Add imports to packages.go: `"errors"`, `"time"`, `"github.com/Mohabdo21/linux-mcp/config"`

Note: `"errors"` is already in the import list of some files - only add if missing.

- [ ] **Step 14: Verify compilation**

Run: `go build ./...`
Expected: builds without errors. If there are import issues, fix them (each file needs `errors`, `time`, and `github.com/Mohabdo21/linux-mcp/config` in imports).

- [ ] **Step 15: Commit**

```bash
git add tools/
git commit -m "feat: add context propagation, timeouts, and disabled-tool checks to all handlers"
```

---

### Task 4: Update Tests and Verify

**Files:**

- Modify: `tools/tools_test.go`
- Modify: `tools_test.go` (if it exists at root, check the import path)
- Run: full test suite and lint

- [ ] _*Step 1: Update test calls that use changed Gather* signatures_*

For all Gather* calls in `tools/tools_test.go` (package `tools`), add `t.Context()` as the first argument:

- `TestGatherSystemInfo`: `GatherSystemInfo()` -> `GatherSystemInfo(t.Context())`
- `TestGatherCPUInfo`: `GatherCPUInfo()` -> `GatherCPUInfo(t.Context())`
- `TestGatherCPUTemperature`: `GatherCPUTemperature()` -> `GatherCPUTemperature(t.Context())`
- `TestGatherMemoryInfo`: `GatherMemoryInfo()` -> `GatherMemoryInfo(t.Context())`
- `TestGatherDiskInfo` / `TestGatherDiskInfoWithFilter` / `TestGatherDiskInfoWithNoMatch`: `GatherDiskInfo(...)` -> `GatherDiskInfo(t.Context(), ...)`
- `TestGatherNetworkInfo`: `GatherNetworkInfo()` -> `GatherNetworkInfo(t.Context())`
- `TestGatherProcessInfo*`: `GatherProcessInfo(...)` -> `GatherProcessInfo(t.Context(), ...)`
- `TestGatherDockerInfo`: already has `t.Context()` -> no change needed
- `TestGatherSystemSnapshot`: already calls `HandleGetSystemSnapshot(t.Context(), ...)` -> no change needed
- `TestGatherLoadAverage`: `GatherLoadAverage()` -> `GatherLoadAverage(t.Context())`
- `TestGatherDNSResolve`: `GatherDNSResolve("localhost")` -> `GatherDNSResolve(t.Context(), "localhost")`
- `TestGatherInstalledPackages` / `TestGatherInstalledPackagesFilter`: already have context -> no change needed

Also verify the test file's import has `"context"` or uses `t.Context()` (Go 1.26 provides `testing.T.Context()`).

- [ ] **Step 2: Run all config tests**

Run: `go test -v ./config/`
Expected: all pass

- [ ] **Step 3: Run tool tests**

Run: `go test -v -race ./tools/`
Expected: tests that don't require external binaries should pass; tests needing `ping`, `pidstat`, `docker`, etc. should skip gracefully

- [ ] **Step 4: Run full build and lint**

Run: `make check && go build ./...`
Expected: gofmt, go vet, golangci-lint all pass; binaries build

- [ ] **Step 5: Run tests**

Run: `go test -race -v ./...`
Expected: all tests pass

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "test: update tests for new context signatures"
```

---

### Task 5: Create Example Config and Verify End-to-End

**Files:**

- Create: `~/.config/linux-mcp/config.json` (example, documented in README)

- [ ] **Step 1: Create an example config**

Create a commented reference file at the project root:

```json
{
  "log_level": "info",
  "timeouts": {
    "get_system_snapshot": "60s",
    "ping_host": "5s"
  },
  "disabled": []
}
```

Place it as `config.example.json` in the project root for users to reference.

- [ ] **Step 2: Quick smoke test**

Build the binary: `make build`
Run: `echo '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' | timeout 2 ./bin/linux-mcp || true`
Expected: JSON-RPC response with tool list (logging goes to stderr)

- [ ] **Step 3: Verify tests pass with final code**

Run: `make test`
Expected: all checks and tests pass

- [ ] **Step 4: Commit**

```bash
git add config.example.json
git commit -m "docs: add example config file"
```
