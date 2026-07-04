# linux-mcp - Linux MCP Server

[![MCP Registry](https://img.shields.io/badge/MCP%20Registry-linux--mcp-000?style=flat-square&logo=github)](https://registry.modelcontextprotocol.io)

A Linux system monitoring server built on the [Model Context Protocol (MCP)](https://modelcontextprotocol.io). Provides real-time system information - CPU, memory, disk, network, processes, Docker, and more - via MCP tools over STDIO transport.

## Features

- **System info** - hostname, OS, kernel version, architecture, uptime, load averages
- **CPU** - usage percentage, model, frequency, physical/virtual core counts
- **CPU temperature** - per-sensor temperature readings (when available)
- **Memory** - RAM and swap usage, used/free/total with percentages
- **Disk** - per-partition usage, mount options, inode usage, and largest files
- **Network** - per-interface I/O statistics, listening ports, and DNS resolution
- **Processes** - running processes sorted by CPU or memory, with configurable limits
- **Docker** - full container lifecycle, image inspection, networks, volumes, disk usage, and system info (via Docker SDK)
- **Services** - systemd service status and full unit inventory
- **Security** - active user sessions and failed login detection
- **Packages** - query installed packages and check for available updates (supports pacman and dpkg)
- **System snapshot** - all of the above in a single call, with graceful degradation on individual failures
- **Man pages** - retrieve system manual pages for any installed command
- **Resources** - system data also available as readable MCP resources (`system:///info`, `system:///cpu`, etc.)

## Prerequisites

- **Go 1.26+** (to build from source)
- **Linux** (the server targets Linux; some tools use Linux-specific paths)
- **Docker** (optional - only needed for Docker tools)

## Installation

### Via MCP Registry (recommended)

Discover and install from the [MCP Registry](https://registry.modelcontextprotocol.io):

```bash
mcp registry install io.github.Mohabdo21/linux-mcp
```

### Build from source

```bash
git clone https://github.com/Mohabdo21/linux-mcp.git
cd linux-mcp
make build
```

The binary is placed at `bin/linux-mcp`.

For a fully static binary (no libc dependency):

```bash
make build-static
```

## Usage

The server communicates over STDIO transport, following the MCP standard. It is designed to be launched by an MCP client (e.g., Claude Desktop, OpenCode, or any MCP host).

### Running directly

```bash
./bin/linux-mcp
```

This starts the server and listens for MCP requests on STDIN/STDOUT.

### Integration with OpenCode / Claude Desktop

Add the following to your MCP client configuration:

```json
{
  "mcpServers": {
    "linux-mcp": {
      "type": "local",
      "command": "/path/to/linux-mcp",
      "enabled": true
    }
  }
}
```

### Available tools

| Tool                           | Description                                                                                                                       |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------- |
| `get_system_info`              | Returns hostname, OS, kernel version, architecture, and uptime                                                                    |
| `get_cpu_info`                 | Returns CPU usage percentage, model, frequency, and core counts                                                                   |
| `get_cpu_temperature`          | Returns current CPU temperature if sensor data is available                                                                       |
| `get_memory_info`              | Returns memory usage including RAM and swap statistics                                                                            |
| `get_disk_info`                | Returns disk usage for mounted partitions, optionally filtered by mount point                                                     |
| `get_network_info`             | Returns network I/O statistics per interface                                                                                      |
| `get_process_info`             | Returns list of running processes, sortable by CPU or memory, with configurable limit                                             |
| `get_docker_info`              | Returns Docker containers and images if Docker is installed                                                                       |
| `get_docker_container_details` | Returns detailed container state, config, env, mounts, and network settings                                                       |
| `get_docker_container_logs`    | Returns log lines from a container with optional tail count and timestamps                                                        |
| `get_docker_container_stats`   | Returns live CPU, memory, network I/O, and PIDs for a container                                                                   |
| `get_docker_container_top`     | Returns running processes inside a Docker container                                                                               |
| `get_docker_container_diff`    | Returns filesystem changes in a container since it was started                                                                    |
| `get_docker_image_history`     | Returns layer history of a Docker image including commands, sizes, and creation times                                             |
| `get_docker_image_details`     | Returns detailed image config, env, entrypoint, labels, and layers                                                                |
| `get_docker_networks`          | Returns Docker networks with driver, scope, and configuration details                                                             |
| `get_docker_volumes`           | Returns Docker volumes with driver, mountpoint, size, and label information                                                       |
| `get_docker_system_info`       | Returns Docker daemon version, storage driver, runtimes, and resource counts                                                      |
| `get_docker_disk_usage`        | Returns Docker disk usage for containers, images, volumes, and build cache                                                        |
| `get_environment_variables`    | Returns all active environment variables as a sorted key-value map; useful for debugging PATH, API keys, and locale settings      |
| `get_system_snapshot`          | Returns a comprehensive snapshot combining all tools                                                                              |
| `get_journal_logs`             | Reads systemd journal logs with optional filtering by unit, priority, and time range; set `user=true` to query user-level journal |
| `get_inode_usage`              | Returns inode usage for mounted filesystems to diagnose "disk full" errors when df shows free space                               |
| `get_listening_ports`          | Returns listening ports and their associated processes for security auditing and port conflict resolution                         |
| `get_service_status`           | Returns detailed status of a systemd service; set `user=true` to query user-level service                                         |
| `get_top_io_processes`         | Returns processes with the highest disk I/O activity to diagnose system lag                                                       |
| `get_failed_logins`            | Returns recent failed login attempts to detect brute-force attacks                                                                |
| `get_gpu_info`                 | Returns GPU information including usage, memory, temperature, and power draw (supports NVIDIA, AMD, Intel)                        |
| `get_hardware_bus_info`        | Lists detected PCI and USB devices for driver troubleshooting and hardware identification                                         |
| `get_largest_files`            | Find the top N largest files/directories in a given path (like du -sh \| sort -hr \| head)                                        |
| `ping_host`                    | Send ICMP packets to a host and return latency, packet loss, and response times                                                   |
| `get_installed_packages`       | Query installed packages (pacman -Q or dpkg -l), optionally filtered by name                                                      |
| `check_updates`                | Count or list available package updates without applying them (pacman -Qu or apt list --upgradable)                               |
| `get_load_average`             | Returns 1-, 5-, and 15-minute load averages as a universal system health check                                                    |
| `get_logged_in_users`          | Returns active user sessions for security and workload awareness                                                                  |
| `get_man_page`                 | Returns the full system manual page for a given command as plain text with optional line limit                                    |
| `resolve_dns`                  | Resolves a hostname to IP addresses to distinguish DNS failures from network failures                                             |
| `get_mount_options`            | Returns mount point options (rw/ro, etc.) for filesystem diagnostics                                                              |
| `get_systemd_units`            | Returns all systemd units and their states for full service inventory                                                             |

### Available resources

In addition to tools, the server exposes system data as MCP resources that clients can read and subscribe to:

| Resource                                  | Description                                                          |
| ----------------------------------------- | -------------------------------------------------------------------- |
| `system:///info`                          | Hostname, OS, kernel version, architecture, and uptime               |
| `system:///cpu`                           | CPU usage, model, frequency, and core counts                         |
| `system:///memory`                        | RAM and swap usage statistics                                        |
| `system:///disk`                          | Disk usage for all mounted partitions                                |
| `system:///disk/{mount_point}` (template) | Disk usage for a specific mount point (e.g. `system:///disk//`)      |
| `system:///network`                       | Network I/O statistics per interface                                 |
| `system:///load`                          | 1-, 5-, and 15-minute load averages                                  |
| `system:///temperature`                   | Current CPU temperature from available sensors                       |
| `system:///gpu`                           | GPU usage, memory, temperature, and power (NVIDIA/AMD/Intel)         |
| `system:///logged_in_users`               | Active user sessions                                                 |
| `system:///listening_ports`               | Listening ports and associated processes                             |
| `system:///failed_logins`                 | Recent failed login attempts                                         |
| `system:///service/{name}` (template)     | Detailed status of a systemd service (e.g. `system:///service/sshd`) |

## Configuration

The server can be configured via a JSON file loaded from `~/.config/linux-mcp/config.json` or the `LINUX_MCP_CONFIG` environment variable.

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

| Field       | Description                                               |
| ----------- | --------------------------------------------------------- |
| `log_level` | One of `debug`, `info`, `warn`, `error` (default: `info`) |
| `timeouts`  | Per-tool timeout overrides as Go duration strings         |
| `disabled`  | List of tool names to disable at startup                  |

The server also handles **SIGHUP** to reload the configuration file at runtime without restarting.

## MCP Registry

Published on the [MCP Registry](https://registry.modelcontextprotocol.io) as `io.github.Mohabdo21/linux-mcp`.

## License

- [MIT](./LICENSE)

## Dependencies

- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) - MCP SDK for Go
- [shirou/gopsutil](https://github.com/shirou/gopsutil) - System metrics (CPU, memory, disk, network, processes, sensors)
- [docker/go-sdk](https://github.com/docker/go-sdk) - Docker Engine API client
