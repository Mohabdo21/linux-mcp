package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	gnet "github.com/shirou/gopsutil/v4/net"
)

const (
	maxNetworkConnections = 200
)

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
	result := make([]InterfaceStats, 0)
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
	_ NoArgs,
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
	lines, err := execLines(ctx, "ss", "-tulnp")
	if err != nil {
		return nil, err
	}
	ports := make([]ListeningPort, 0)
	for _, line := range lines {
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
		return &ResolveDNSOutput{Hostname: hostname, Addresses: []string{}}, err
	}
	return &ResolveDNSOutput{Hostname: hostname, Addresses: addrs}, nil
}

type GetNetworkConnectionsInput struct {
	Status           string `json:"status,omitempty"            jsonschema:"optional status filter (e.g. ESTABLISHED, LISTEN, TIME_WAIT)"`
	Type             string `json:"type,omitempty"              jsonschema:"optional type filter: tcp, udp"`
	ResolveHostnames bool   `json:"resolve_hostnames,omitempty" jsonschema:"optional: resolve remote hostnames via reverse DNS (default: false)"`
	Grouped          bool   `json:"grouped,omitempty"           jsonschema:"optional: group connections by PID (default: false)"`
	MaxConnections   int    `json:"max_connections,omitempty"   jsonschema:"optional: limit results (max: 200)"`
}

type NetworkConnection struct {
	FD             uint32 `json:"fd"`
	Family         string `json:"family"`
	Type           string `json:"type"`
	LocalAddr      string `json:"local_addr"`
	LocalPort      uint32 `json:"local_port"`
	RemoteAddr     string `json:"remote_addr"`
	RemotePort     uint32 `json:"remote_port"`
	Status         string `json:"status"`
	PID            int32  `json:"pid"`
	ProcessName    string `json:"process_name,omitempty"`
	RemoteHostname string `json:"remote_hostname,omitempty"`
}

type ConnectionGroup struct {
	PID         int32               `json:"pid"`
	ProcessName string              `json:"process_name"`
	Connections []NetworkConnection `json:"connections"`
}

type NetworkConnectionsOutput struct {
	Connections []NetworkConnection `json:"connections"`
	Groups      []ConnectionGroup   `json:"groups,omitempty"`
	OutputErrors
}

func resolveProcessNames(pids map[int32]struct{}) map[int32]string {
	names := make(map[int32]string, len(pids))
	for pid := range pids {
		if pid <= 0 {
			continue
		}
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid))
		if err != nil {
			continue
		}
		names[pid] = strings.TrimSpace(string(data))
	}
	return names
}

func resolveRemoteHostnames(
	ctx context.Context,
	addrs map[string]struct{},
) map[string]string {
	names := make(map[string]string, len(addrs))
	for addr := range addrs {
		hosts, err := net.LookupAddr(addr)
		if err != nil || len(hosts) == 0 {
			continue
		}
		names[addr] = strings.TrimRight(hosts[0], ".")
	}
	return names
}

func GatherNetworkConnections(
	ctx context.Context,
	status string,
	connType string,
	resolveHostnames bool,
	grouped bool,
	maxConnections int,
) (*NetworkConnectionsOutput, error) {
	conns, err := gnet.Connections("all")
	if err != nil {
		return nil, err
	}

	var filterType uint32
	switch connType {
	case "tcp":
		filterType = syscall.SOCK_STREAM
	case "udp":
		filterType = syscall.SOCK_DGRAM
	}

	result := make([]NetworkConnection, 0, len(conns))
	for _, c := range conns {
		if status != "" && c.Status != status {
			continue
		}
		if filterType != 0 && c.Type != filterType {
			continue
		}
		family := "unknown"
		switch c.Family {
		case syscall.AF_INET:
			family = "ipv4"
		case syscall.AF_INET6:
			family = "ipv6"
		}
		connTypeStr := "unknown"
		switch c.Type {
		case syscall.SOCK_STREAM:
			connTypeStr = "tcp"
		case syscall.SOCK_DGRAM:
			connTypeStr = "udp"
		}
		nc := NetworkConnection{
			FD:         c.Fd,
			Family:     family,
			Type:       connTypeStr,
			LocalAddr:  c.Laddr.IP,
			LocalPort:  c.Laddr.Port,
			RemoteAddr: c.Raddr.IP,
			RemotePort: c.Raddr.Port,
			Status:     c.Status,
			PID:        c.Pid,
		}
		result = append(result, nc)
	}

	// Resolve process names from unique PIDs.
	pids := make(map[int32]struct{})
	for i := range result {
		pids[result[i].PID] = struct{}{}
	}
	procNames := resolveProcessNames(pids)
	for i := range result {
		if name, ok := procNames[result[i].PID]; ok {
			result[i].ProcessName = name
		}
	}

	// Resolve remote hostnames if requested.
	if resolveHostnames {
		addrs := make(map[string]struct{})
		for i := range result {
			ra := result[i].RemoteAddr
			if ra != "" && ra != "0.0.0.0" && ra != "::" {
				addrs[ra] = struct{}{}
			}
		}
		hostnames := resolveRemoteHostnames(ctx, addrs)
		for i := range result {
			if name, ok := hostnames[result[i].RemoteAddr]; ok {
				result[i].RemoteHostname = name
			}
		}
	}

	// Apply max connections truncation.
	if maxConnections > 0 && len(result) > maxConnections {
		result = result[:maxConnections]
	}

	// Group by PID if requested.
	var groups []ConnectionGroup
	if grouped {
		pidGroups := make(map[int32][]NetworkConnection)
		for _, conn := range result {
			pidGroups[conn.PID] = append(pidGroups[conn.PID], conn)
		}
		for pid, groupConns := range pidGroups {
			groups = append(groups, ConnectionGroup{
				PID:         pid,
				ProcessName: procNames[pid],
				Connections: groupConns,
			})
		}
		sort.Slice(groups, func(i, j int) bool {
			return groups[i].PID < groups[j].PID
		})
	}

	return &NetworkConnectionsOutput{
		Connections: result,
		Groups:      groups,
	}, nil
}

func HandleGetNetworkConnections(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetNetworkConnectionsInput,
) (*mcp.CallToolResult, *NetworkConnectionsOutput, error) {
	maxConn := input.MaxConnections
	if maxConn < 0 {
		maxConn = 0
	} else if maxConn > maxNetworkConnections {
		maxConn = maxNetworkConnections
	}
	return handleToolCall(
		ctx,
		"get_network_connections",
		0,
		func(ctx context.Context) (*NetworkConnectionsOutput, error) {
			return GatherNetworkConnections(
				ctx,
				input.Status,
				input.Type,
				input.ResolveHostnames,
				input.Grouped,
				maxConn,
			)
		},
	)
}

func HandleResolveDNS(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input ResolveDNSInput,
) (*mcp.CallToolResult, *ResolveDNSOutput, error) {
	if err := requireField(input.Hostname, "hostname"); err != nil {
		return nil, nil, err
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

type GetIPInfoInput struct {
	IP string `json:"ip,omitempty" jsonschema:"optional IP address to lookup (defaults to your public IP)"`
}

type IPInfoOutput struct {
	IP          string   `json:"ip"`
	ASN         string   `json:"asn,omitempty"`
	Org         string   `json:"org,omitempty"`
	Country     string   `json:"country,omitempty"`
	City        string   `json:"city,omitempty"`
	Region      string   `json:"region,omitempty"`
	ServiceTags []string `json:"service_tags,omitempty"`
	OutputErrors
}

type ipAPIResponse struct {
	Status  string `json:"status"`
	Country string `json:"country"`
	City    string `json:"city"`
	Region  string `json:"regionName"`
	ISP     string `json:"isp"`
	Org     string `json:"org"`
	AS      string `json:"as"`
	Query   string `json:"query"`
}

var serviceTagPatterns = []struct {
	patterns []string
	tag      string
}{
	{
		[]string{
			"AMAZON-02",
			"AMAZON-08",
			"AMAZON-09",
			"AMAZON ",
			"AWS",
			"AMAZOW",
		},
		"AWS",
	},
	{[]string{"GOOGLE", "GOOGLE-FIBER", "GCP"}, "Google Cloud"},
	{[]string{"CLOUDFLARE"}, "Cloudflare"},
	{[]string{"GITHUB"}, "GitHub"},
	{[]string{"MICROSOFT", "AZURE"}, "Azure"},
	{[]string{"DIGITALOCEAN"}, "DigitalOcean"},
	{[]string{"LINODE"}, "Linode"},
	{[]string{"OVH"}, "OVH"},
	{[]string{"HETZNER"}, "Hetzner"},
	{[]string{"ORACLE-", "ORACLE"}, "Oracle Cloud"},
	{[]string{"FASTLY"}, "Fastly"},
}

func detectServiceTags(isp, org, asn string) []string {
	combined := strings.ToUpper(isp + " " + org + " " + asn)
	seen := make(map[string]bool)
	var tags []string
	for _, entry := range serviceTagPatterns {
		for _, p := range entry.patterns {
			if strings.Contains(combined, p) {
				if !seen[entry.tag] {
					tags = append(tags, entry.tag)
					seen[entry.tag] = true
				}
				break
			}
		}
	}
	return tags
}

func parseASN(asField string) string {
	if asField == "" {
		return ""
	}
	parts := strings.SplitN(asField, " ", 2)
	return parts[0]
}

const ipAPIURL = "http://ip-api.com/json/"

func GatherIPInfo(ctx context.Context, ip string) (*IPInfoOutput, error) {
	url := ipAPIURL
	if ip != "" {
		if parsed := net.ParseIP(ip); parsed == nil {
			return &IPInfoOutput{
					IP: ip,
				}, net.InvalidAddrError(
					"invalid IP address",
				)
		}
		url += ip
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "linux-mcp/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	var apiResp ipAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if apiResp.Status != "success" {
		return &IPInfoOutput{IP: apiResp.Query}, nil
	}

	asn := parseASN(apiResp.AS)
	tags := detectServiceTags(apiResp.ISP, apiResp.Org, apiResp.AS)

	return &IPInfoOutput{
		IP:          apiResp.Query,
		ASN:         asn,
		Org:         apiResp.Org,
		Country:     apiResp.Country,
		City:        apiResp.City,
		Region:      apiResp.Region,
		ServiceTags: tags,
	}, nil
}

func HandleGetIPInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetIPInfoInput,
) (*mcp.CallToolResult, *IPInfoOutput, error) {
	return handleToolCall(
		ctx,
		"get_ip_info",
		0,
		func(ctx context.Context) (*IPInfoOutput, error) {
			return GatherIPInfo(ctx, input.IP)
		},
	)
}
