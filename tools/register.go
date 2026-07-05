package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

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
		"get_system_info",
		"Returns system information including hostname, OS, kernel version, architecture, and uptime",
		HandleGetSystemInfo,
	)
	registerTool(
		server,
		"get_cpu_info",
		"Returns CPU information including usage percentage, model, frequency, and core counts",
		HandleGetCPUInfo,
	)
	registerTool(
		server,
		"get_cpu_temperature",
		"Returns current CPU temperature if sensor data is available",
		HandleGetCPUTemperature,
	)
	registerTool(
		server,
		"get_memory_info",
		"Returns memory usage including RAM and swap statistics",
		HandleGetMemoryInfo,
	)
	registerTool(
		server,
		"get_disk_info",
		"Returns disk usage for mounted partitions, optionally filtered by mount point",
		HandleGetDiskInfo,
	)
	registerTool(
		server,
		"get_network_info",
		"Returns network I/O statistics per interface",
		HandleGetNetworkInfo,
	)
	registerTool(
		server,
		"get_process_info",
		"Returns list of running processes, optionally sorted by CPU or memory usage with configurable limit",
		HandleGetProcessInfo,
	)
	registerTool(
		server,
		"get_docker_info",
		"Returns Docker containers and images if Docker is installed",
		HandleGetDockerInfo,
	)
	registerTool(
		server,
		"get_docker_container_details",
		"Returns detailed information about a Docker container including state, config, env, mounts, and network settings",
		HandleGetContainerDetail,
	)
	registerTool(
		server,
		"get_docker_container_logs",
		"Returns log lines from a Docker container (stdout/stderr) with optional tail count and timestamps",
		HandleGetContainerLogs,
	)
	registerTool(
		server,
		"get_docker_container_stats",
		"Returns live resource usage statistics for a Docker container including CPU, memory, network I/O, and PIDs",
		HandleGetContainerStats,
	)
	registerTool(
		server,
		"get_docker_container_top",
		"Returns running processes inside a Docker container",
		HandleGetContainerTop,
	)
	registerTool(
		server,
		"get_docker_container_diff",
		"Returns filesystem changes (added, modified, deleted files) in a Docker container since it was started",
		HandleGetContainerDiff,
	)
	registerTool(
		server,
		"get_docker_image_history",
		"Returns the layer history of a Docker image including commands, sizes, and creation times",
		HandleGetImageHistory,
	)
	registerTool(
		server,
		"get_docker_image_details",
		"Returns detailed information about a Docker image including config, env, entrypoint, labels, and layers",
		HandleGetImageDetail,
	)
	registerTool(
		server,
		"get_docker_networks",
		"Returns a list of Docker networks with driver, scope, and configuration details",
		HandleGetDockerNetworks,
	)
	registerTool(
		server,
		"get_docker_volumes",
		"Returns a list of Docker volumes with driver, mountpoint, size, and label information",
		HandleGetDockerVolumes,
	)
	registerTool(
		server,
		"get_docker_system_info",
		"Returns Docker daemon system information including version, storage driver, runtimes, and resource counts",
		HandleGetDockerSystemInfo,
	)
	registerTool(
		server,
		"get_docker_disk_usage",
		"Returns Docker disk usage breakdown for containers, images, volumes, and build cache",
		HandleGetDockerDiskUsage,
	)
	registerTool(
		server,
		"get_docker_stats_all",
		"Returns CPU, memory, network I/O, and block I/O for all running containers in a single call. Accepts an optional list of container names or IDs to filter.",
		HandleGetDockerStatsAll,
	)
	registerTool(
		server,
		"get_docker_system_snapshot",
		"Returns a comprehensive Docker health snapshot combining containers, images, running stats, disk usage, and networks in a single call.",
		HandleGetDockerSystemSnapshot,
	)
	registerTool(
		server,
		"get_system_snapshot",
		"Returns a comprehensive snapshot of system status combining all tools",
		HandleGetSystemSnapshot,
	)
	registerTool(
		server,
		"get_journal_logs",
		"Reads systemd journal logs with optional filtering by unit, priority, and time range. Set user=true to query user-level journal.",
		HandleGetJournalLogs,
	)
	registerTool(
		server,
		"get_inode_usage",
		"Returns inode usage for mounted filesystems to diagnose 'disk full' errors when df shows free space",
		HandleGetInodeUsage,
	)
	registerTool(
		server,
		"get_listening_ports",
		"Returns listening ports and their associated processes for security auditing and port conflict resolution",
		HandleGetListeningPorts,
	)
	registerTool(
		server,
		"get_service_status",
		"Returns detailed status of a systemd service. Set user=true to query user-level service.",
		HandleGetServiceStatus,
	)
	registerTool(
		server,
		"get_top_io_processes",
		"Returns processes with the highest disk I/O activity to diagnose system lag",
		HandleGetTopIOProcesses,
	)
	registerTool(
		server,
		"get_failed_logins",
		"Returns recent failed login attempts to detect brute-force attacks",
		HandleGetFailedLogins,
	)
	registerTool(
		server,
		"get_gpu_info",
		"Returns GPU information including usage, memory, temperature, and power draw (supports NVIDIA, AMD, Intel)",
		HandleGetGPUInfo,
	)
	registerTool(
		server,
		"get_largest_files",
		"Find the top N largest files/directories in a given path (like du -sh | sort -hr | head)",
		HandleGetLargestFiles,
	)
	registerTool(
		server,
		"ping_host",
		"Send ICMP packets to a host and return latency, packet loss, and response times",
		HandlePingHost,
	)
	registerTool(
		server,
		"get_installed_packages",
		"Query installed packages (Arch: pacman -Q, Debian: dpkg -l, etc.), optionally filtered by name",
		HandleGetInstalledPackages,
	)
	registerTool(
		server,
		"check_updates",
		"Count or list available package updates without applying them (e.g., pacman -Qu, apt list --upgradable)",
		HandleCheckUpdates,
	)
	registerTool(
		server,
		"get_load_average",
		"Returns 1-, 5-, and 15-minute load averages as a universal system health check",
		HandleGetLoadAverage,
	)
	registerTool(
		server,
		"get_logged_in_users",
		"Returns active user sessions for security and workload awareness",
		HandleGetLoggedInUsers,
	)
	registerTool(
		server,
		"resolve_dns",
		"Resolves a hostname to IP addresses to distinguish DNS failures from network failures",
		HandleResolveDNS,
	)
	registerTool(
		server,
		"get_mount_options",
		"Returns mount point options (rw/ro, etc.) for filesystem diagnostics",
		HandleGetMountOptions,
	)
	registerTool(
		server,
		"get_systemd_units",
		"Returns all systemd units and their states for full service inventory",
		HandleGetSystemdUnits,
	)
	registerTool(
		server,
		"get_man_page",
		"Fetches the authoritative man page for any Linux command. Use this when the user asks about flags, syntax, or edge cases. Optional search helps pinpoint specific sections.",
		HandleGetManPage,
	)
	registerTool(
		server,
		"get_environment_variables",
		"Returns all active environment variables for the current process as a sorted key-value map. Useful for debugging PATH, API keys, locale settings, and shell configuration in the MCP server runtime. Supports an optional search parameter to filter by name prefix or substring.",
		HandleGetEnvironmentVariables,
	)
	registerTool(
		server,
		"get_hardware_bus_info",
		"Lists detected PCI and USB devices on the system. Useful for identifying attached hardware like network cards, audio interfaces, and expansion cards for driver troubleshooting and configuration verification. Supports an optional search parameter to filter devices by any field (bus, slot, class, vendor, device).",
		HandleGetHardwareBusInfo,
	)
	registerTool(
		server,
		"get_user_automation",
		"Aggregates and lists all scheduled background scripts or automation tasks running specifically under the current user account. Combines crontab entries and systemd user timers.",
		HandleGetUserAutomation,
	)
	registerTool(
		server,
		"get_desktop_session_info",
		"Returns metadata regarding the active graphic display protocol (Wayland/X11), desktop session identifiers, and related environment configuration.",
		HandleGetDesktopSessionInfo,
	)
}
