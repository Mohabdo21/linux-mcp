package tools

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type PingHostInput struct {
	Host    string `json:"host"              jsonschema:"hostname or IP address to ping"`
	Count   int    `json:"count,omitempty"   jsonschema:"number of packets (default: 4)"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"timeout in seconds (default: 10)"`
}

type PingOutput struct {
	Host               string   `json:"host"`
	PacketsTransmitted int      `json:"packets_transmitted"`
	PacketsReceived    int      `json:"packets_received"`
	PacketLossPercent  float64  `json:"packet_loss_percent"`
	MinLatencyMs       float64  `json:"min_latency_ms"`
	AvgLatencyMs       float64  `json:"avg_latency_ms"`
	MaxLatencyMs       float64  `json:"max_latency_ms"`
	Errors             []string `json:"errors,omitempty"`
}

func GatherPing(host string, count, timeout int) (PingOutput, error) {
	if count <= 0 {
		count = 4
	}
	if timeout <= 0 {
		timeout = 10
	}
	cmd := exec.Command(
		"ping",
		"-c",
		fmt.Sprintf("%d", count),
		"-w",
		fmt.Sprintf("%d", timeout),
		host,
	)
	out, err := cmd.Output()
	output := string(out)
	result := PingOutput{Host: host}
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
		return PingOutput{}, err
	}
	return result, nil
}

func HandlePingHost(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input PingHostInput,
) (*mcp.CallToolResult, PingOutput, error) {
	if input.Host == "" {
		return nil, PingOutput{}, errors.New("host is required")
	}
	out, err := GatherPing(input.Host, input.Count, input.Timeout)
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
