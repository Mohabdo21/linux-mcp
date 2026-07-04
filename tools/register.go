package tools

import "github.com/modelcontextprotocol/go-sdk/mcp"

func RegisterTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_system_info",
		Description: "Returns system information including hostname, OS, " +
			"kernel version, architecture, and uptime",
	}, HandleGetSystemInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_cpu_info",
		Description: "Returns CPU information including usage percentage, " +
			"model, frequency, and core counts",
	}, HandleGetCPUInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_cpu_temperature",
		Description: "Returns current CPU temperature if sensor data " +
			"is available",
	}, HandleGetCPUTemperature)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_memory_info",
		Description: "Returns memory usage including RAM and swap " +
			"statistics",
	}, HandleGetMemoryInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_disk_info",
		Description: "Returns disk usage for mounted partitions, " +
			"optionally filtered by mount point",
	}, HandleGetDiskInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_network_info",
		Description: "Returns network I/O statistics per interface",
	}, HandleGetNetworkInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_process_info",
		Description: "Returns list of running processes, optionally " +
			"sorted by CPU or memory usage with configurable limit",
	}, HandleGetProcessInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_info",
		Description: "Returns Docker containers and images if Docker " +
			"is installed",
	}, HandleGetDockerInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_container_details",
		Description: "Returns detailed information about a Docker " +
			"container including state, config, env, mounts, and network settings",
	}, HandleGetContainerDetail)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_container_logs",
		Description: "Returns log lines from a Docker container " +
			"(stdout/stderr) with optional tail count and timestamps",
	}, HandleGetContainerLogs)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_container_stats",
		Description: "Returns live resource usage statistics for a " +
			"Docker container including CPU, memory, network I/O, and PIDs",
	}, HandleGetContainerStats)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_docker_container_top",
		Description: "Returns running processes inside a Docker container",
	}, HandleGetContainerTop)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_container_diff",
		Description: "Returns filesystem changes (added, modified, " +
			"deleted files) in a Docker container since it was started",
	}, HandleGetContainerDiff)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_image_history",
		Description: "Returns the layer history of a Docker image " +
			"including commands, sizes, and creation times",
	}, HandleGetImageHistory)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_image_details",
		Description: "Returns detailed information about a Docker " +
			"image including config, env, entrypoint, labels, and layers",
	}, HandleGetImageDetail)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_networks",
		Description: "Returns a list of Docker networks with driver, " +
			"scope, and configuration details",
	}, HandleGetDockerNetworks)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_volumes",
		Description: "Returns a list of Docker volumes with driver, " +
			"mountpoint, size, and label information",
	}, HandleGetDockerVolumes)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_system_info",
		Description: "Returns Docker daemon system information " +
			"including version, storage driver, runtimes, and resource counts",
	}, HandleGetDockerSystemInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_docker_disk_usage",
		Description: "Returns Docker disk usage breakdown for " +
			"containers, images, volumes, and build cache",
	}, HandleGetDockerDiskUsage)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_system_snapshot",
		Description: "Returns a comprehensive snapshot of system " +
			"status combining all tools",
	}, HandleGetSystemSnapshot)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_journal_logs",
		Description: "Reads systemd journal logs with optional " +
			"filtering by unit, priority, and time range. " +
			"Set user=true to query user-level journal.",
	}, HandleGetJournalLogs)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_inode_usage",
		Description: "Returns inode usage for mounted filesystems " +
			"to diagnose 'disk full' errors when df shows free space",
	}, HandleGetInodeUsage)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_listening_ports",
		Description: "Returns listening ports and their associated " +
			"processes for security auditing and port conflict resolution",
	}, HandleGetListeningPorts)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_service_status",
		Description: "Returns detailed status of a systemd service. " +
			"Set user=true to query user-level service.",
	}, HandleGetServiceStatus)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_top_io_processes",
		Description: "Returns processes with the highest disk I/O " +
			"activity to diagnose system lag",
	}, HandleGetTopIOProcesses)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_failed_logins",
		Description: "Returns recent failed login attempts to detect " +
			"brute-force attacks",
	}, HandleGetFailedLogins)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_gpu_info",
		Description: "Returns GPU information including usage, memory, " +
			"temperature, and power draw (supports NVIDIA, AMD, Intel)",
	}, HandleGetGPUInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_largest_files",
		Description: "Find the top N largest files/directories in " +
			"a given path (like du -sh | sort -hr | head)",
	}, HandleGetLargestFiles)
	mcp.AddTool(server, &mcp.Tool{
		Name: "ping_host",
		Description: "Send ICMP packets to a host and return latency, " +
			"packet loss, and response times",
	}, HandlePingHost)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_installed_packages",
		Description: "Query installed packages " +
			"(Arch: pacman -Q, Debian: dpkg -l, etc.), " +
			"optionally filtered by name",
	}, HandleGetInstalledPackages)
	mcp.AddTool(server, &mcp.Tool{
		Name: "check_updates",
		Description: "Count or list available package updates " +
			"without applying them " +
			"(e.g., pacman -Qu, apt list --upgradable)",
	}, HandleCheckUpdates)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_load_average",
		Description: "Returns 1-, 5-, and 15-minute load averages " +
			"as a universal system health check",
	}, HandleGetLoadAverage)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_logged_in_users",
		Description: "Returns active user sessions for security " +
			"and workload awareness",
	}, HandleGetLoggedInUsers)
	mcp.AddTool(server, &mcp.Tool{
		Name: "resolve_dns",
		Description: "Resolves a hostname to IP addresses to " +
			"distinguish DNS failures from network failures",
	}, HandleResolveDNS)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_mount_options",
		Description: "Returns mount point options (rw/ro, etc.) " +
			"for filesystem diagnostics",
	}, HandleGetMountOptions)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_systemd_units",
		Description: "Returns all systemd units and their states " +
			"for full service inventory",
	}, HandleGetSystemdUnits)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_man_page",
		Description: "Fetches the authoritative man page for any Linux command. " +
			"Use this when the user asks about flags, syntax, or edge cases. " +
			"Optional search helps pinpoint specific sections.",
	}, HandleGetManPage)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_environment_variables",
		Description: "Returns all active environment variables for the " +
			"current process as a sorted key-value map. " +
			"Useful for debugging PATH, API keys, locale settings, " +
			"and shell configuration in the MCP server runtime. " +
			"Supports an optional search parameter to filter by " +
			"name prefix or substring.",
	}, HandleGetEnvironmentVariables)
	mcp.AddTool(server, &mcp.Tool{
		Name: "get_hardware_bus_info",
		Description: "Lists detected PCI and USB devices on the system. " +
			"Useful for identifying attached hardware like network cards, " +
			"audio interfaces, and expansion cards " +
			"for driver troubleshooting and configuration verification. " +
			"Supports an optional search parameter to filter devices " +
			"by any field (bus, slot, class, vendor, device).",
	}, HandleGetHardwareBusInfo)
}
