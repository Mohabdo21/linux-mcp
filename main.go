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

	//registerTools(server)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Printf("Server failed: %v", err)
	}
}

func registerTools(server *mcp.Server) {
	// TODO: register all 9 tools here
}
