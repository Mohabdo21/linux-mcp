package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "system-status",
		Version: "1.0.0",
	}, nil)

	registerTools(server)

	if err := server.Run(
		context.Background(),
		&mcp.StdioTransport{},
	); err != nil {
		log.Printf("Server failed: %v", err)
	}
}

func registerTools(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_system_info",
		Description: "Returns system information including hostname, OS, kernel version, architecture, and uptime",
	}, handleGetSystemInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_cpu_info",
		Description: "Returns CPU information including usage percentage, model, frequency, and core counts",
	}, handleGetCPUInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_cpu_temperature",
		Description: "Returns current CPU temperature if sensor data is available",
	}, handleGetCPUTemperature)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_memory_info",
		Description: "Returns memory usage including RAM and swap statistics",
	}, handleGetMemoryInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_disk_info",
		Description: "Returns disk usage for mounted partitions, optionally filtered by mount point",
	}, handleGetDiskInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_network_info",
		Description: "Returns network I/O statistics per interface",
	}, handleGetNetworkInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_process_info",
		Description: "Returns list of running processes, optionally sorted by CPU or memory usage with configurable limit",
	}, handleGetProcessInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_docker_info",
		Description: "Returns Docker containers and images if Docker is installed",
	}, handleGetDockerInfo)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_system_snapshot",
		Description: "Returns a comprehensive snapshot of system status combining all tools",
	}, handleGetSystemSnapshot)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_journal_logs",
		Description: "Reads systemd journal logs with optional filtering by unit, priority, and time range",
	}, handleGetJournalLogs)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_inode_usage",
		Description: "Returns inode usage for mounted filesystems to diagnose 'disk full' errors when df shows free space",
	}, handleGetInodeUsage)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_listening_ports",
		Description: "Returns listening ports and their associated processes for security auditing and port conflict resolution",
	}, handleGetListeningPorts)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_service_status",
		Description: "Returns detailed status of a systemd service",
	}, handleGetServiceStatus)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_top_io_processes",
		Description: "Returns processes with the highest disk I/O activity to diagnose system lag",
	}, handleGetTopIOProcesses)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_failed_logins",
		Description: "Returns recent failed login attempts to detect brute-force attacks",
	}, handleGetFailedLogins)
}
