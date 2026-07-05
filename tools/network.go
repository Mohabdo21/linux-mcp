package tools

import (
	"context"
	"net"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	gnet "github.com/shirou/gopsutil/v4/net"
)

type GetNetworkInfoInput struct{}

type InterfaceStats struct {
	Name        string `json:"name"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	ErrorsIn    uint64 `json:"errors_in"`
	ErrorsOut   uint64 `json:"errors_out"`
	DropsIn     uint64 `json:"drops_in"`
	DropsOut    uint64 `json:"drops_out"`
}

type NetworkInfoOutput struct {
	Interfaces []InterfaceStats `json:"interfaces"`
	OutputErrors
}

func GatherNetworkInfo(ctx context.Context) (*NetworkInfoOutput, error) {
	counters, err := gnet.IOCounters(true)
	if err != nil {
		return nil, err
	}
	var result []InterfaceStats
	for _, c := range counters {
		result = append(result, InterfaceStats{
			Name:        c.Name,
			BytesSent:   c.BytesSent,
			BytesRecv:   c.BytesRecv,
			PacketsSent: c.PacketsSent,
			PacketsRecv: c.PacketsRecv,
			ErrorsIn:    c.Errin,
			ErrorsOut:   c.Errout,
			DropsIn:     c.Dropin,
			DropsOut:    c.Dropout,
		})
	}
	return &NetworkInfoOutput{Interfaces: result}, nil
}

func HandleGetNetworkInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ GetNetworkInfoInput,
) (*mcp.CallToolResult, *NetworkInfoOutput, error) {
	return handleToolCall(
		ctx,
		"get_network_info",
		0,
		GatherNetworkInfo,
	)
}

type GetListeningPortsInput struct {
	Protocol string `json:"protocol,omitempty" jsonschema:"optional protocol filter: tcp, udp"`
}

type ListeningPort struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     string `json:"port"`
	Process  string `json:"process,omitempty"`
}

type ListeningPortsOutput struct {
	Ports []ListeningPort `json:"ports"`
	OutputErrors
}

func GatherListeningPorts(
	ctx context.Context,
	protocol string,
) (*ListeningPortsOutput, error) {
	cmd := exec.CommandContext(ctx, "ss", "-tulnp")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var ports []ListeningPort
	for line := range strings.SplitSeq(
		strings.TrimSpace(string(out)), "\n",
	) {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 || fields[0] == "Netid" {
			continue
		}
		netid := fields[0]
		if protocol != "" && netid != protocol {
			continue
		}
		addr, port, _ := SplitHostPort(fields[4])
		proc := ""
		if len(fields) >= 7 {
			proc = ParseProcessField(fields[6])
		}
		ports = append(ports, ListeningPort{
			Protocol: netid,
			Address:  addr,
			Port:     port,
			Process:  proc,
		})
	}
	return &ListeningPortsOutput{Ports: ports}, nil
}

func HandleGetListeningPorts(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetListeningPortsInput,
) (*mcp.CallToolResult, *ListeningPortsOutput, error) {
	return handleToolCall(
		ctx,
		"get_listening_ports",
		0,
		func(ctx context.Context) (*ListeningPortsOutput, error) {
			return GatherListeningPorts(ctx, input.Protocol)
		},
	)
}

type ResolveDNSInput struct {
	Hostname string `json:"hostname" jsonschema:"hostname to resolve (e.g. 'example.com')"`
}

type ResolveDNSOutput struct {
	Hostname  string   `json:"hostname"`
	Addresses []string `json:"addresses"`
	OutputErrors
}

func GatherDNSResolve(
	ctx context.Context,
	hostname string,
) (*ResolveDNSOutput, error) {
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return &ResolveDNSOutput{Hostname: hostname}, err
	}
	return &ResolveDNSOutput{Hostname: hostname, Addresses: addrs}, nil
}

func HandleResolveDNS(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input ResolveDNSInput,
) (*mcp.CallToolResult, *ResolveDNSOutput, error) {
	if input.Hostname == "" {
		return nil, nil, net.InvalidAddrError("hostname is required")
	}
	return handleToolCall(
		ctx,
		"resolve_dns",
		0,
		func(ctx context.Context) (*ResolveDNSOutput, error) {
			return GatherDNSResolve(ctx, input.Hostname)
		},
	)
}
