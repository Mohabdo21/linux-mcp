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
- **Disk** - per-partition usage, inodes, mount options, largest files, block devices
- **Network** - interface stats, active connections, listening ports, DNS, ping, IP geolocation & ASN lookup
- **Processes** - running processes sorted by CPU or memory, open file descriptors per process
- **Docker** - containers, images, networks, volumes, disk usage, system info, stats for all containers, system snapshot
- **Services & automation** - systemd units, service status, user timers, crontab, system cron jobs
- **Security** - active user sessions, failed login detection, SELinux/AppArmor status, firewall/SSH/SUID/world-writable audit with security score
- **Packages** - installed packages and available updates (pacman, dpkg)
- **Hardware** - GPU info, PCI/USB bus devices, power/battery analytics
- **Desktop session** - Wayland/X11 protocol, DE identifiers, runtime config
- **Storage health** - RAID status, logrotate configuration, time synchronization, SMART disk health, per-device I/O metrics
- **System health** - comprehensive health assessment with memory, disk, load, and systemd checks
- **/proc diagnostics** - deep /proc inspection: interrupts, softirqs, vmstat, diskstats, filesystems, kernel version, slabinfo
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

### Tools and resources

62 tools and 19 resources covering system, CPU, memory, disk, network, processes, Docker, security, packages, hardware, and more.

**[Full tool and resource reference](docs/tools.md)**

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
