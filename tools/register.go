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
}
