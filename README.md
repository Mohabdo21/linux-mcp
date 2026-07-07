# linux-mcp - Linux MCP Server

[![MCP Registry](https://img.shields.io/badge/MCP%20Registry-linux--mcp-000?style=flat-square&logo=github)](https://registry.modelcontextprotocol.io)

A Linux system monitoring server built on the [Model Context Protocol (MCP)](https://modelcontextprotocol.io). Provides real-time system information - CPU, memory, disk, network, processes, Docker, and more - via MCP tools over STDIO transport.

<p align="center">
  <img src="assets/demo.gif" alt="linux-mcp demo" width="100%">
</p>

## Features

- **System** - hostname, OS, kernel, architecture, uptime, load averages
- **CPU** - usage, model, frequency, core counts, temperature sensors
- **Memory** - RAM and swap usage with percentages
- **Disk** - per-partition usage, inodes, mount options, largest files
- **Network** - interface stats, active connections, listening ports, DNS, ping, IP geolocation & ASN lookup
- **Processes** - running processes sorted by CPU or memory, open file descriptors per process
- **Docker** - containers, images, networks, volumes, disk usage, system info, stats for all containers, system snapshot
- **Services & automation** - systemd units, service status, user timers, crontab
- **Security** - active user sessions, failed login detection
- **Packages** - installed packages and available updates (pacman, dpkg)
- **Hardware** - GPU info, PCI/USB bus devices, power/battery analytics
- **Desktop session** - Wayland/X11 protocol, DE identifiers, runtime config
- **Man pages** - system manual pages for any installed command
- **Snapshot** - comprehensive system overview in a single call
- **MCP Resources** - system data also accessible as readable resources

## Prerequisites

- **Go 1.26+** (to build from source)
- **Linux** (the server targets Linux; some tools use Linux-specific paths)
- **Docker** (optional - only needed for Docker tools)

## Installation

### Via MCP Registry

Discover the server on the [MCP Registry](https://registry.modelcontextprotocol.io) and install via your MCP client (VS Code one-click install, or manual config).

### Download pre-built binary

Download the latest binary from the [GitHub Releases](https://github.com/Mohabdo21/linux-mcp/releases) page:

```bash
curl -LO https://github.com/Mohabdo21/linux-mcp/releases/latest/download/linux-mcp
chmod +x linux-mcp
```

A fully static build (no libc dependency) is also available as `linux-mcp_static`.

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

### Integration with OpenCode

Add the following to your OpenCode configuration:

```json
{
  "mcp": {
    "linux-mcp": {
      "type": "local",
      "command": ["/path/to/linux-mcp"],
      "enabled": true
    }
  }
}
```

### Available tools

| Tool                           | Description                                                                                                                                                                                 |
| ------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `get_system_info`              | Returns hostname, OS, kernel, architecture, uptime, platform details, process count, boot time, virtualization info, host UUID, hardware (DMI) product info, BIOS version, and TPM version  |
| `get_cpu_info`                 | Returns CPU usage percentage, model, frequency, and core counts                                                                                                                             |
| `get_cpu_temperature`          | Returns current CPU temperature if sensor data is available                                                                                                                                 |
| `get_desktop_session_info`     | Returns display protocol (Wayland/X11), desktop environment identifiers, and runtime configuration                                                                                          |
| `get_memory_info`              | Returns memory usage including RAM and swap statistics                                                                                                                                      |
| `get_disk_info`                | Returns disk usage for mounted partitions, optionally filtered by mount point                                                                                                               |
| `get_network_info`             | Returns network I/O statistics per interface                                                                                                                                                |
| `get_process_info`             | Returns list of running processes, sortable by CPU or memory, with configurable limit                                                                                                       |
| `get_process_fds`              | Lists open file descriptors (files, sockets, pipes) and total count for a specific process ID                                                                                               |
| `get_docker_info`              | Returns Docker containers and images if Docker is installed                                                                                                                                 |
| `get_docker_container_details` | Returns detailed container state, config, env, mounts, and network settings                                                                                                                 |
| `get_docker_container_logs`    | Returns log lines from a container with optional tail count and timestamps                                                                                                                  |
| `get_docker_container_stats`   | Returns live CPU, memory, network I/O, and PIDs for one or more containers (comma-separated names/IDs, or `all`)                                                                            |
| `get_docker_container_top`     | Returns running processes inside a Docker container                                                                                                                                         |
| `get_docker_container_diff`    | Returns filesystem changes in a container since it was started                                                                                                                              |
| `get_docker_image_history`     | Returns layer history of a Docker image including commands, sizes, and creation times                                                                                                       |
| `get_docker_image_details`     | Returns detailed image config, env, entrypoint, labels, and layers                                                                                                                          |
| `get_docker_networks`          | Returns Docker networks with driver, scope, and configuration details                                                                                                                       |
| `get_docker_stats_all`         | Returns CPU, memory, network I/O, and block I/O for all running containers; accepts optional container name/ID filter                                                                       |
| `get_docker_system_info`       | Returns Docker daemon version, storage driver, runtimes, and resource counts                                                                                                                |
| `get_docker_system_snapshot`   | Returns a comprehensive Docker health snapshot combining containers, images, running stats, disk usage, and networks                                                                        |
| `get_docker_volumes`           | Returns a list of Docker volumes with driver, mountpoint, size, and label information                                                                                                       |
| `get_docker_disk_usage`        | Returns Docker disk usage for containers, images, volumes, and build cache                                                                                                                  |
| `get_environment_variables`    | Returns all active environment variables as a sorted key-value map; useful for debugging PATH, API keys, and locale settings                                                                |
| `get_system_snapshot`          | Returns a comprehensive snapshot combining all tools                                                                                                                                        |
| `get_journal_logs`             | Reads systemd journal logs with optional filtering by unit, priority, and time range; set `user=true` to query user-level journal                                                           |
| `get_inode_usage`              | Returns inode usage for mounted filesystems to diagnose "disk full" errors when df shows free space                                                                                         |
| `get_network_connections`      | Returns all active TCP/UDP connections with state, addresses, process info, and optional reverse DNS hostnames; supports filtering by status and type, grouping by PID, and result limiting |
| `get_listening_ports`          | Returns listening ports and their associated processes for security auditing and port conflict resolution                                                                                   |
| `get_service_status`           | Returns detailed status of a systemd service; set `user=true` to query user-level service                                                                                                   |
| `get_top_io_processes`         | Returns processes with the highest disk I/O activity to diagnose system lag                                                                                                                 |
| `get_user_automation`          | Aggregates crontab entries and systemd user timers for a complete view of user-level scheduled tasks                                                                                        |
| `get_failed_logins`            | Returns recent failed login attempts to detect brute-force attacks                                                                                                                          |
| `get_gpu_info`                 | Returns GPU information including usage, memory, temperature, and power draw (supports NVIDIA, AMD, Intel)                                                                                  |
| `get_power_analytics`          | Returns battery status, charge percentage, discharge rate, capacity degradation, and AC power state                                                                                         |
| `get_hardware_bus_info`        | Lists detected PCI and USB devices for driver troubleshooting and hardware identification                                                                                                   |
| `get_largest_files`            | Find the top N largest files/directories in a given path (like du -sh \| sort -hr \| head)                                                                                                  |
| `ping_host`                    | Send ICMP packets to a host and return latency, packet loss, and response times                                                                                                             |
| `get_installed_packages`       | Query installed packages (pacman -Q or dpkg -l), optionally filtered by name                                                                                                                |
| `check_updates`                | Count or list available package updates without applying them (pacman -Qu or apt list --upgradable)                                                                                         |
| `get_load_average`             | Returns 1-, 5-, and 15-minute load averages as a universal system health check                                                                                                              |
| `get_logged_in_users`          | Returns active user sessions for security and workload awareness                                                                                                                            |
| `get_man_page`                 | Returns the full system manual page for a given command as plain text with optional line limit                                                                                              |
| `resolve_dns`                  | Resolves a hostname to IP addresses to distinguish DNS failures from network failures                                                                                                       |
| `get_ip_info`                  | Returns IP geolocation (country, city, region), ASN/organization, and detected service provider tags (e.g. "AWS", "Cloudflare", "GitHub"). Uses ip-api.com free geolocation API             |
| `get_user_info`                | Lists system users parsed from /etc/passwd and /etc/group including username, UID, GID, home directory, shell, and supplementary group memberships. Supports optional username filtering.   |
| `get_mount_options`            | Returns mount point options (rw/ro, etc.) for filesystem diagnostics                                                                                                                        |
| `get_systemd_units`            | Returns all systemd units and their states for full service inventory                                                                                                                       |

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
- [shirou/gopsutil/v4](https://github.com/shirou/gopsutil) - System metrics (CPU, memory, disk, network, processes, sensors)
- [docker/go-sdk](https://github.com/docker/go-sdk) - Docker Engine API client
- [moby/moby/client](https://github.com/moby/moby) - Docker client library
- [ip-api.com](https://ip-api.com) - Free IP geolocation API (used by `get_ip_info`)
