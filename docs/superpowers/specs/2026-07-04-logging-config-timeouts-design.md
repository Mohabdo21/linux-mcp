# Logging, Configuration, and Timeouts for linux-mcp

## Overview

Add structured logging (stderr), JSON configuration file, and context-based
timeouts to the linux-mcp MCP server. Zero new dependencies beyond the Go
standard library.

## Approach

**Approach 1: Lightweight stdlib-only.** Config loading + slog + timeout
wrapping in a few focused files. Package-level state via atomic.Value for
SIGHUP reload support. No dependency injection refactor.

## New Package: `config`

```
config/
  config.go        -- Config struct, load/validation, ToolTimeout(), IsDisabled()
  config_schema.json -- JSON Schema for IDE completion
  config_test.go   -- Unit tests
```

### Config struct

```go
type Config struct {
    LogLevel string            `json:"log_level"` // "debug", "info", "warn", "error"
    Timeouts map[string]string `json:"timeouts"`  // tool name -> duration string
    Disabled []string          `json:"disabled"`   // list of disabled tool names
}
```

### Load path

1. `$LINUX_MCP_CONFIG` env var (if set, must be a valid path)
2. `~/.config/linux-mcp/config.json`
3. No file found -> use compiled-in defaults

### Validation

- Every entry in `Timeouts` must parse via `time.ParseDuration`
- `LogLevel` must match slog levels (debug/info/warn/error)
- `Disabled` entries are checked against known tool names at startup
- On validation failure: log the error and continue with defaults (do not crash)

### Thread safety

Config is stored in `atomic.Value` (a `*Config`). Load/Reload replaces the
pointer atomically. `Get()` returns a shallow copy so callers never mutate
shared state.

### Key functions

- `Load() error` -- loads from default path, falls back to env override
- `Reload() error` -- re-reads the same file path, re-validates, swaps atomically
- `Get() Config` -- returns a copy of the current config
- `ToolTimeout(name string, fallback time.Duration) time.Duration` -- reads from
  config, falls back to the provided default
- `IsDisabled(name string) bool`

### Default timeouts

| Tool                   | Default |
| ---------------------- | ------: |
| get_cpu_info           |      5s |
| get_cpu_temperature    |      5s |
| get_memory_info        |      5s |
| get_disk_info          |     10s |
| get_inode_usage        |     10s |
| get_network_info       |      5s |
| get_process_info       |     10s |
| get_docker_info        |     10s |
| get_system_snapshot    |    120s |
| get_journal_logs       |     20s |
| get_listening_ports    |     10s |
| get_service_status     |     10s |
| get_top_io_processes   |     15s |
| get_failed_logins      |     10s |
| get_gpu_info           |      5s |
| get_largest_files      |     30s |
| ping_host              |     10s |
| get_installed_packages |     15s |
| check_updates          |     15s |
| get_load_average       |      5s |
| get_logged_in_users    |      5s |
| resolve_dns            |     10s |
| get_mount_options      |     10s |
| get_systemd_units      |     10s |

## Logging (`slog` on stderr)

### Setup in `main.go`

```go
func setupLogging(level slog.Level) {
    handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
    slog.SetDefault(slog.New(handler))
}
```

### Logged events

- **Server start**: version, config path, log level
- **Tool call**: tool name, duration, error count, result summary
- **Config reload**: success or validation failure
- **Shutdown**: server exit

### Tool handler logging pattern

A helper in `tools/util.go`:

```go
func LogToolCall(ctx context.Context, tool string, dur time.Duration, errs int)
```

Each handler adds a defer:

```go
func HandleX(ctx context.Context, req *mcp.CallToolRequest, input XInput) (...) {
    start := time.Now()
    defer func() { LogToolCall(ctx, "tool_name", time.Since(start), len(out.Errors)) }()
    // ...
}
```

### No sensitive data

Logs include tool name, duration, error count -- never IPs, hostnames,
command output, or PII from arguments.

## Timeouts

### Wrapper in `tools/util.go`

```go
func WithToolTimeout(ctx context.Context, name string, fallback time.Duration) (context.Context, context.CancelFunc)
```

- Reads configured timeout from `config.ToolTimeout(name, fallback)`
- Calls `context.WithTimeout(ctx, timeout)`
- Each handler calls this as its first line

### Context propagation

All `Gather*` functions receive `ctx context.Context` even if they don't
use it directly (consistency). Handlers pass their timeout-wrapped context
through to `Gather*` and any `exec.CommandContext` calls.

Currently missing context on `Gather*`:

- GatherSystemInfo
- GatherCPUInfo
- GatherCPUTemperature
- GatherMemoryInfo
- GatherDiskInfo
- GatherNetworkInfo
- GatherProcessInfo
- GatherLoadAverage
- GatherDNSResolve

These get `ctx` added to their signatures. The `Handle*` functions pass the
timeout-wrapped context through. No behavioral change for gopsutil calls
(they ignore context), but the signature is consistent and timeout truncation
is clear.

### Disabled tool handling

In each `Handle*` function:

```go
if config.IsDisabled("tool_name") {
    return nil, OutputType{}, errors.New("tool disabled by configuration")
}
```

This returns a clear error so MCP clients (including AI) know why the tool
is unavailable.

## Files changed

| File                              | Change                                                 |
| --------------------------------- | ------------------------------------------------------ |
| `main.go`                         | Add config loading, slog setup, SIGHUP handler         |
| `config/config.go` (new)          | Config struct, load, validate, ToolTimeout, IsDisabled |
| `config/config_schema.json` (new) | JSON Schema                                            |
| `config/config_test.go` (new)     | Unit tests for config load/validate                    |
| `tools/util.go`                   | Add LogToolCall, WithToolTimeout                       |
| `tools/ping.go`                   | Add timeout wrapper + config.IsDisabled check          |
| `tools/process.go`                | Add timeout wrapper, ctx to GatherProcessInfo          |
| `tools/cpu.go`                    | Add ctx to GatherCPUInfo, GatherCPUTemperature         |
| `tools/memory.go`                 | Add ctx to GatherMemoryInfo                            |
| `tools/disk.go`                   | Add ctx to GatherDiskInfo, timeouts to handlers        |
| `tools/network.go`                | Add ctx to GatherNetworkInfo, timeouts to handlers     |
| `tools/system.go`                 | Add ctx to GatherSystemInfo, GatherLoadAverage         |
| `tools/docker.go`                 | Add timeout wrapper                                    |
| `tools/security.go`               | Add timeout wrappers                                   |
| `tools/journal.go`                | Add timeout wrapper                                    |
| `tools/service.go`                | Add timeout wrappers                                   |
| `tools/gpu.go`                    | Add timeout wrapper                                    |
| `tools/packages.go`               | Add timeout wrappers                                   |
| `tools/tools_test.go`             | Update Gather* calls with context arguments            |

## Tests

- `config/config_test.go`: load valid config, missing config, invalid timeout,
  invalid log level, disabled tool lookup, ToolTimeout with fallback
- Existing tests updated to pass `context.Background()` (or `t.Context()`) to
  Gather* functions that now require context

## Future (not in scope)

- Hot reload acknowledgement log line
- Per-tool disabled reason messages
- More granular timeout overrides (e.g. per-call timeout from the tool input
  itself -- user can already specify count/limit)
