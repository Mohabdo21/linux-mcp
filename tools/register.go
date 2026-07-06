package tools

import (
	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTool[In, Out any](
	s *mcp.Server,
	name, description string,
	h mcp.ToolHandlerFor[In, Out],
) {
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: description}, h)
}

func RegisterTools(server *mcp.Server) {
	registerTool(
		server,
		config.ToolNameGetSystemInfo,
		"Returns system information including hostname, OS, kernel, architecture, uptime, platform details, process count, boot time, virtualization info, host UUID, hardware (DMI) product info, BIOS version, and TPM version",
		HandleGetSystemInfo,
	)
	registerTool(
		server,
		config.ToolNameGetCPUInfo,
		"Returns CPU information including usage percentage, model, frequency, and core counts",
		HandleGetCPUInfo,
	)
	registerTool(
		server,
		config.ToolNameGetCPUTemperature,
		"Returns current CPU temperature if sensor data is available",
		HandleGetCPUTemperature,
	)
	registerTool(
		server,
		config.ToolNameGetMemoryInfo,
		"Returns memory usage including RAM and swap statistics",
		HandleGetMemoryInfo,
	)
	registerTool(
		server,
		config.ToolNameGetDiskInfo,
		"Returns disk usage for mounted partitions, optionally filtered by mount point",
		HandleGetDiskInfo,
	)
	registerTool(
		server,
		config.ToolNameGetNetworkInfo,
		"Returns network I/O statistics per interface",
		HandleGetNetworkInfo,
	)
	registerTool(
		server,
		config.ToolNameGetProcessInfo,
		"Returns list of running processes, optionally sorted by CPU or memory usage with configurable limit",
		HandleGetProcessInfo,
	)
	registerTool(
		server,
		config.ToolNameGetDockerInfo,
		"Returns Docker containers and images if Docker is installed",
		HandleGetDockerInfo,
	)
	registerTool(
		server,
		config.ToolNameGetDockerContainerDetails,
		"Returns detailed information about a Docker container including state, config, env, mounts, and network settings",
		HandleGetContainerDetail,
	)
	registerTool(
		server,
		config.ToolNameGetDockerContainerLogs,
		"Returns log lines from a Docker container (stdout/stderr) with optional tail count and timestamps",
		HandleGetContainerLogs,
	)
	registerTool(
		server,
		config.ToolNameGetDockerContainerStats,
		"Returns live resource usage statistics for a Docker container including CPU, memory, network I/O, and PIDs",
		HandleGetContainerStats,
	)
	registerTool(
		server,
		config.ToolNameGetDockerContainerTop,
		"Returns running processes inside a Docker container",
		HandleGetContainerTop,
	)
	registerTool(
		server,
		config.ToolNameGetDockerContainerDiff,
		"Returns filesystem changes (added, modified, deleted files) in a Docker container since it was started",
		HandleGetContainerDiff,
	)
	registerTool(
		server,
		config.ToolNameGetDockerImageHistory,
		"Returns the layer history of a Docker image including commands, sizes, and creation times",
		HandleGetImageHistory,
	)
	registerTool(
		server,
		config.ToolNameGetDockerImageDetails,
		"Returns detailed information about a Docker image including config, env, entrypoint, labels, and layers",
		HandleGetImageDetail,
	)
	registerTool(
		server,
		config.ToolNameGetDockerNetworks,
		"Returns a list of Docker networks with driver, scope, and configuration details",
		HandleGetDockerNetworks,
	)
	registerTool(
		server,
		config.ToolNameGetDockerVolumes,
		"Returns a list of Docker volumes with driver, mountpoint, size, and label information",
		HandleGetDockerVolumes,
	)
	registerTool(
		server,
		config.ToolNameGetDockerSystemInfo,
		"Returns Docker daemon system information including version, storage driver, runtimes, and resource counts",
		HandleGetDockerSystemInfo,
	)
	registerTool(
		server,
		config.ToolNameGetDockerDiskUsage,
		"Returns Docker disk usage breakdown for containers, images, volumes, and build cache",
		HandleGetDockerDiskUsage,
	)
	registerTool(
		server,
		config.ToolNameGetDockerStatsAll,
		"Returns CPU, memory, network I/O, and block I/O for all running containers in a single call. Accepts an optional list of container names or IDs to filter.",
		HandleGetDockerStatsAll,
	)
	registerTool(
		server,
		config.ToolNameGetDockerSystemSnapshot,
		"Returns a comprehensive Docker health snapshot combining containers, images, running stats, disk usage, and networks in a single call.",
		HandleGetDockerSystemSnapshot,
	)
	registerTool(
		server,
		config.ToolNameGetSystemSnapshot,
		"Returns a comprehensive snapshot of system status combining all tools",
		HandleGetSystemSnapshot,
	)
	registerTool(
		server,
		config.ToolNameGetJournalLogs,
		"Reads systemd journal logs with optional filtering by unit, priority, and time range. Set user=true to query user-level journal.",
		HandleGetJournalLogs,
	)
	registerTool(
		server,
		config.ToolNameGetInodeUsage,
		"Returns inode usage for mounted filesystems to diagnose 'disk full' errors when df shows free space",
		HandleGetInodeUsage,
	)
	registerTool(
		server,
		config.ToolNameGetNetworkConnections,
		"Returns all active network connections (TCP and UDP) including state, local/remote addresses, process info, and optional reverse DNS hostnames. Supports filtering by status (e.g. ESTABLISHED, LISTEN, TIME_WAIT) and type (tcp, udp). Optionally resolve_hostnames for remote addresses, group by PID, and limit results with max_connections.",
		HandleGetNetworkConnections,
	)
	registerTool(
		server,
		config.ToolNameGetListeningPorts,
		"Returns listening ports and their associated processes for security auditing and port conflict resolution",
		HandleGetListeningPorts,
	)
	registerTool(
		server,
		config.ToolNameGetServiceStatus,
		"Returns detailed status of a systemd service. Set user=true to query user-level service.",
		HandleGetServiceStatus,
	)
	registerTool(
		server,
		config.ToolNameGetProcessFDs,
		"Lists the open file descriptors (files, sockets, pipes) and total count for a specific process ID",
		HandleGetProcessFDs,
	)
	registerTool(
		server,
		config.ToolNameGetTopIOProcesses,
		"Returns processes with the highest disk I/O activity to diagnose system lag",
		HandleGetTopIOProcesses,
	)
	registerTool(
		server,
		config.ToolNameGetFailedLogins,
		"Returns recent failed login attempts to detect brute-force attacks",
		HandleGetFailedLogins,
	)
	registerTool(
		server,
		config.ToolNameGetGPUInfo,
		"Returns GPU information including usage, memory, temperature, and power draw (supports NVIDIA, AMD, Intel)",
		HandleGetGPUInfo,
	)
	registerTool(
		server,
		config.ToolNameGetLargestFiles,
		"Find the top N largest files/directories in a given path (like du -sh | sort -hr | head)",
		HandleGetLargestFiles,
	)
	registerTool(
		server,
		config.ToolNamePingHost,
		"Send ICMP packets to a host and return latency, packet loss, and response times",
		HandlePingHost,
	)
	registerTool(
		server,
		config.ToolNameGetInstalledPackages,
		"Query installed packages (Arch: pacman -Q, Debian: dpkg -l, etc.), optionally filtered by name",
		HandleGetInstalledPackages,
	)
	registerTool(
		server,
		config.ToolNameCheckUpdates,
		"Count or list available package updates without applying them (e.g., pacman -Qu, apt list --upgradable)",
		HandleCheckUpdates,
	)
	registerTool(
		server,
		config.ToolNameGetLoadAverage,
		"Returns 1-, 5-, and 15-minute load averages as a universal system health check",
		HandleGetLoadAverage,
	)
	registerTool(
		server,
		config.ToolNameGetLoggedInUsers,
		"Returns active user sessions for security and workload awareness",
		HandleGetLoggedInUsers,
	)
	registerTool(
		server,
		config.ToolNameResolveDNS,
		"Resolves a hostname to IP addresses to distinguish DNS failures from network failures",
		HandleResolveDNS,
	)
	registerTool(
		server,
		config.ToolNameGetMountOptions,
		"Returns mount point options (rw/ro, etc.) for filesystem diagnostics",
		HandleGetMountOptions,
	)
	registerTool(
		server,
		config.ToolNameGetSystemdUnits,
		"Returns all systemd units and their states for full service inventory",
		HandleGetSystemdUnits,
	)
	registerTool(
		server,
		config.ToolNameGetManPage,
		"Fetches the authoritative man page for any Linux command. Use this when the user asks about flags, syntax, or edge cases. Optional search helps pinpoint specific sections.",
		HandleGetManPage,
	)
	registerTool(
		server,
		config.ToolNameGetEnvironmentVariables,
		"Returns all active environment variables for the current process as a sorted key-value map. Useful for debugging PATH, API keys, locale settings, and shell configuration in the MCP server runtime. Supports an optional search parameter to filter by name prefix or substring.",
		HandleGetEnvironmentVariables,
	)
	registerTool(
		server,
		config.ToolNameGetHardwareBusInfo,
		"Lists detected PCI and USB devices on the system. Useful for identifying attached hardware like network cards, audio interfaces, and expansion cards for driver troubleshooting and configuration verification. Supports an optional search parameter to filter devices by any field (bus, slot, class, vendor, device).",
		HandleGetHardwareBusInfo,
	)
	registerTool(
		server,
		config.ToolNameGetUserAutomation,
		"Aggregates and lists all scheduled background scripts or automation tasks running specifically under the current user account. Combines crontab entries and systemd user timers.",
		HandleGetUserAutomation,
	)
	registerTool(
		server,
		config.ToolNameGetDesktopSessionInfo,
		"Returns metadata regarding the active graphic display protocol (Wayland/X11), desktop session identifiers, and related environment configuration.",
		HandleGetDesktopSessionInfo,
	)
	registerTool(
		server,
		config.ToolNameGetPowerAnalytics,
		"Returns the active power state (AC vs Battery), current discharge rate in watts, current battery percentage, and overall capacity degradation",
		HandleGetPowerAnalytics,
	)
	registerTool(
		server,
		config.ToolNameGetUserInfo,
		"Lists system users parsed from /etc/passwd and /etc/group including username, UID, GID, home directory, shell, and supplementary group memberships. Supports optional username filtering.",
		HandleGetUserInfo,
	)
	registerTool(
		server,
		config.ToolNameGetIPInfo,
		"Returns IP geolocation data, ASN/organization information, and known service provider tags (e.g. \"AWS\", \"Cloudflare\", \"GitHub\") for a given IP address or your own public IP. Uses the ip-api.com free geolocation service.",
		HandleGetIPInfo,
	)
}
