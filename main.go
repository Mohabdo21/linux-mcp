package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "linux-mcp",
		Version: "1.0.0",
	}, nil)

	registerTools(server)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
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
}
