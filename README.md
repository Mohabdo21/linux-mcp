# linux-mcp - Linux MCP Server

A Linux system monitoring server built on the [Model Context Protocol (MCP)](https://modelcontextprotocol.io). Provides real-time system information - CPU, memory, disk, network, processes, Docker, and more - via MCP tools over STDIO transport.

## Features

- **System info** - hostname, OS, kernel version, architecture, uptime, load averages
- **CPU** - usage percentage, model, frequency, physical/virtual core counts
- **CPU temperature** - per-sensor temperature readings (when available)
- **Memory** - RAM and swap usage, used/free/total with percentages
- **Disk** - per-partition usage, mount options, inode usage, and largest files
- **Network** - per-interface I/O statistics, listening ports, and DNS resolution
- **Processes** - running processes sorted by CPU or memory, with configurable limits
- **Docker** - container and image listing (via Docker CLI)
- **Services** - systemd service status and full unit inventory
- **Security** - active user sessions and failed login detection
- **Packages** - query installed packages and check for available updates (supports pacman and dpkg)
- **System snapshot** - all of the above in a single call, with graceful degradation on individual failures

## Prerequisites

- **Go 1.26+** (to build from source)
- **Linux** (the server targets Linux; some tools use Linux-specific paths)
- **Docker** (optional - only needed for Docker info tools)

## Installation

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

| Tool                     | Description                                                                                                                       |
| ------------------------ | --------------------------------------------------------------------------------------------------------------------------------- |
| `get_system_info`        | Returns hostname, OS, kernel version, architecture, and uptime                                                                    |
| `get_cpu_info`           | Returns CPU usage percentage, model, frequency, and core counts                                                                   |
| `get_cpu_temperature`    | Returns current CPU temperature if sensor data is available                                                                       |
| `get_memory_info`        | Returns memory usage including RAM and swap statistics                                                                            |
| `get_disk_info`          | Returns disk usage for mounted partitions, optionally filtered by mount point                                                     |
| `get_network_info`       | Returns network I/O statistics per interface                                                                                      |
| `get_process_info`       | Returns list of running processes, sortable by CPU or memory, with configurable limit                                             |
| `get_docker_info`        | Returns Docker containers and images if Docker is installed                                                                       |
| `get_system_snapshot`    | Returns a comprehensive snapshot combining all tools                                                                              |
| `get_journal_logs`       | Reads systemd journal logs with optional filtering by unit, priority, and time range; set `user=true` to query user-level journal |
| `get_inode_usage`        | Returns inode usage for mounted filesystems to diagnose "disk full" errors when df shows free space                               |
| `get_listening_ports`    | Returns listening ports and their associated processes for security auditing and port conflict resolution                         |
| `get_service_status`     | Returns detailed status of a systemd service; set `user=true` to query user-level service                                         |
| `get_top_io_processes`   | Returns processes with the highest disk I/O activity to diagnose system lag                                                       |
| `get_failed_logins`      | Returns recent failed login attempts to detect brute-force attacks                                                                |
| `get_gpu_info`           | Returns GPU information including usage, memory, temperature, and power draw (supports NVIDIA, AMD, Intel)                        |
| `get_largest_files`      | Find the top N largest files/directories in a given path (like du -sh \| sort -hr \| head)                                        |
| `ping_host`              | Send ICMP packets to a host and return latency, packet loss, and response times                                                   |
| `get_installed_packages` | Query installed packages (pacman -Q or dpkg -l), optionally filtered by name                                                      |
| `check_updates`          | Count or list available package updates without applying them (pacman -Qu or apt list --upgradable)                               |
| `get_load_average`       | Returns 1-, 5-, and 15-minute load averages as a universal system health check                                                    |
| `get_logged_in_users`    | Returns active user sessions for security and workload awareness                                                                  |
| `resolve_dns`            | Resolves a hostname to IP addresses to distinguish DNS failures from network failures                                             |
| `get_mount_options`      | Returns mount point options (rw/ro, etc.) for filesystem diagnostics                                                              |
| `get_systemd_units`      | Returns all systemd units and their states for full service inventory                                                             |

## License

- [MIT](./LICENSE)

## Dependencies

- [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) - MCP SDK for Go
- [shirou/gopsutil](https://github.com/shirou/gopsutil) - System metrics (CPU, memory, disk, network, processes, sensors)
