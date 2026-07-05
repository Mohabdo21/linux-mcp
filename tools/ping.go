package tools

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var errInvalidHost = errors.New(
	"invalid host: must be a valid hostname or IP address")

func validHost(host string) bool {
	if host == "" || len(host) > 253 {
		return false
	}
	if net.ParseIP(host) != nil {
		return true
	}
	for _, r := range host {
		if r != '.' && r != '-' && !unicode.IsLetter(r) &&
			!unicode.IsDigit(r) {
			return false
		}
	}
	for label := range strings.SplitSeq(host, ".") {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
	}
	return true
}

type PingHostInput struct {
	Host    string `json:"host"              jsonschema:"hostname or IP address to ping"`
	Count   int    `json:"count,omitempty"   jsonschema:"number of packets (default: 4)"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"timeout in seconds (default: 10)"`
}

type PingOutput struct {
	Host               string  `json:"host"`
	PacketsTransmitted int     `json:"packets_transmitted"`
	PacketsReceived    int     `json:"packets_received"`
	PacketLossPercent  float64 `json:"packet_loss_percent"`
	MinLatencyMs       float64 `json:"min_latency_ms"`
	AvgLatencyMs       float64 `json:"avg_latency_ms"`
	MaxLatencyMs       float64 `json:"max_latency_ms"`
	OutputErrors
}

func GatherPing(
	ctx context.Context,
	host string,
	count, timeout int,
) (*PingOutput, error) {
	if count <= 0 {
		count = 4
	}
	if timeout <= 0 {
		timeout = 10
	}
	cmd := exec.CommandContext(ctx,
		"ping",
		"-c",
		fmt.Sprintf("%d", count),
		"-w",
		fmt.Sprintf("%d", timeout),
		host,
	)
	out, err := cmd.Output()
	output := string(out)
	result := &PingOutput{Host: host}
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, "packets transmitted") {
			_, _ = fmt.Sscanf(
				line,
				"%d packets transmitted, %d received, %f%% packet loss",
				&result.PacketsTransmitted,
				&result.PacketsReceived,
				&result.PacketLossPercent,
			)
		}
		if strings.Contains(line, "rtt min/avg/max/mdev") {
			if _, after, ok := strings.Cut(line, "= "); ok {
				stats := strings.TrimSpace(after)
				parts := strings.Split(stats, "/")
				if len(parts) >= 3 {
					_, _ = fmt.Sscanf(
						parts[0], "%f", &result.MinLatencyMs)
					_, _ = fmt.Sscanf(
						parts[1], "%f", &result.AvgLatencyMs)
					maxPart := strings.Fields(parts[2])
					if len(maxPart) > 0 {
						_, _ = fmt.Sscanf(
							maxPart[0],
							"%f",
							&result.MaxLatencyMs,
						)
					}
				}
			}
		}
	}
	if err != nil && output == "" {
		return nil, err
	}
	return result, nil
}

func HandlePingHost(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input PingHostInput,
) (*mcp.CallToolResult, *PingOutput, error) {
	if !validHost(input.Host) {
		return nil, nil, errInvalidHost
	}
	return handleToolCall(
		ctx,
		"ping_host",
		10*time.Second,
		func(ctx context.Context) (*PingOutput, error) {
			return GatherPing(ctx, input.Host, input.Count, input.Timeout)
		},
	)
}
