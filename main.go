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
}
