package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/Mohabdo21/linux-mcp/tools"
)

var Version = "dev"

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func setupLogging() {
	cfg := config.Get()
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLogLevel(cfg.LogLevel),
	})
	slog.SetDefault(slog.New(handler))
}

func main() {
	if err := config.Load(); err != nil {
		log.Printf("Config load error (using defaults): %v", err)
	}

	setupLogging()

	slog.Info("server starting",
		"version", Version,
		"log_level", config.Get().LogLevel,
	)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "linux-mcp",
		Version: Version,
	}, nil)

	tools.RegisterTools(server)
	tools.RegisterResources(server)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP)
	go func() {
		for range sigCh {
			if err := config.Reload(); err != nil {
				slog.Error("config reload failed", "error", err)
			} else {
				setupLogging()
				slog.Info("config reloaded")
			}
		}
	}()

	if err := server.Run(
		context.Background(),
		&mcp.StdioTransport{},
	); err != nil {
		slog.Error("server failed", "error", err)
	}

	slog.Info("server stopped")
}
