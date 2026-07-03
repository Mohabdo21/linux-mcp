package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mohabdo21/linux-mcp/tools"
)

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "linux-mcp",
		Version: "1.0.0",
	}, nil)

	tools.RegisterTools(server)

	if err := server.Run(
		context.Background(),
		&mcp.StdioTransport{},
	); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
