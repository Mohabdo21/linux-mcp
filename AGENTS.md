# AGENTS.md - linux-mcp (Linux MCP Server)

## Build & Test

- `make check` runs fmt, vet, and golangci-lint (enforces 80-char max line length via golines).
- `make test` depends on `make check` - tests will not run unless fmt/vet/lint all pass.
- Run a single test: `go test -race -v -run TestGatherCPUInfo ./...`
- Build: `make build` (outputs to `bin/linux-mcp`); `make build-static` for fully static binary.
- Uses Go 1.26.4 with iterator-based APIs like `strings.SplitSeq`.

## Architecture

- Entrypoint: `main.go` creates an MCP server named `"linux-mcp"` v1.0.0 on STDIO transport.
- All tool handlers live in `tools/` package (14 files, `package tools`). One category per file: `cpu.go`, `memory.go`, `disk.go`, `network.go`, `process.go`, `docker.go`, `system.go`, `journal.go`, `service.go`, `security.go`, `gpu.go`, `ping.go`, plus `register.go` and `util.go`.
- Registration via `tools.RegisterTools(server)`.
- Tests in `tools_test.go` (~498 lines, `package main`). Imports `tools.` prefix.
- Handler pattern: `func(ctx, *mcp.CallToolRequest, InputType) (*mcp.CallToolResult, OutputType, error)` - first return is always `nil`.
- Each tool has a `Gather*` exported function tested independently from the handler.
- Input structs use `jsonschema:` tags. Validation uses defaults and clamping (max limits enforced).

## Key Conventions

- `get_system_snapshot` uses graceful degradation: individual failures append to `Errors` slice, never fail the whole call.
- Many tools rely on external Linux CLIs (`docker`, `journalctl`, `df`, `ss`, `systemctl`, `pidstat`, `lastb`, `nvidia-smi`, `rocm-smi`, `intel_gpu_top`, `du`, `ping`). Tests skip gracefully if the binary is missing.
- `get_service_status` and `get_journal_logs` accept `user=true` for user-level systemd queries.
- No CI/CD, no Dockerfile, no .env.
