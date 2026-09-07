package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// resourceDef carries one resource's metadata and how to resolve its content.
type resourceDef struct {
	Path        string
	Name        string
	Description string
	resolver    func(ctx context.Context, rest string) (any, error)
}

// staticResource builds a resourceDef from a no-argument gather.
func staticResource[Out any](
	path, name, description string,
	gather func(context.Context) (Out, error),
) resourceDef {
	r := resourceDef{Path: path, Name: name, Description: description,
		resolver: func(ctx context.Context, _ string) (any, error) {
			return gather(ctx)
		}}
	return r
}

// URI returns the fully-qualified resource URI for the entry.
func (r *resourceDef) URI() string {
	return scheme(strings.TrimPrefix(r.Path, "/"))
}

// prefix returns the static prefix before the first template parameter,
// or an empty string for static resources.
func (r *resourceDef) prefix() string {
	if i := strings.Index(r.Path, "{"); i >= 0 {
		return r.Path[:i]
	}
	return ""
}

func (r *resourceDef) isTemplate() bool { return r.prefix() != "" }

var resourceRegistry = []resourceDef{
	staticResource(
		"/info",
		"System Information",
		"Hostname, OS, kernel version, architecture, and uptime",
		GatherSystemInfo,
	),
	staticResource("/cpu", "CPU Information",
		"CPU usage, model, frequency, and core counts", GatherCPUInfo),
	staticResource("/memory", "Memory Information",
		"RAM and swap usage statistics", GatherMemoryInfo),
	{Path: "/disk", Name: "Disk Information",
		Description: "Disk usage for all mounted partitions",
		resolver: func(ctx context.Context, _ string) (any, error) {
			return GatherDiskInfo(ctx, "", 0)
		}},
	staticResource("/network", "Network Information",
		"Network I/O statistics per interface", GatherNetworkInfo),
	staticResource("/load", "Load Average",
		"1-, 5-, and 15-minute load averages", GatherLoadAverage),
	staticResource("/temperature", "CPU Temperature",
		"Current CPU temperature from available sensors", GatherCPUTemperature),
	staticResource("/gpu", "GPU Information",
		"GPU usage, memory, temperature, and power "+
			"(NVIDIA/AMD/Intel)", GatherGPUInfo),
	staticResource("/logged_in_users", "Logged In Users",
		"Active user sessions", GatherLoggedInUsers),
	{Path: "/listening_ports", Name: "Listening Ports",
		Description: "Listening ports and associated processes",
		resolver: func(ctx context.Context, _ string) (any, error) {
			return GatherListeningPorts(ctx, "")
		}},
	{Path: "/failed_logins", Name: "Failed Logins",
		Description: "Recent failed login attempts",
		resolver: func(ctx context.Context, _ string) (any, error) {
			return GatherFailedLogins(ctx, 20)
		}},
	staticResource(
		"/block_devices",
		"Block Devices",
		"Block devices and partitions detected on the system",
		GatherBlockDevices,
	),
	staticResource("/raid", "RAID Status",
		"Software RAID status from /proc/mdstat", GatherRAIDStatus),
	staticResource("/time_sync", "Time Sync Status",
		"NTP/Chrony time synchronization status", GatherTimeSyncStatus),
	staticResource(
		"/selinux_apparmor",
		"SELinux/AppArmor Status",
		"Status of SELinux and AppArmor security modules",
		GatherSELinuxAppArmorStatus,
	),
	staticResource("/logrotate", "Logrotate Status",
		"Logrotate configuration and state file", GatherLogrotateStatus),
	staticResource("/health", "System Health Check",
		"Comprehensive system health assessment", GatherSystemHealthCheck),
	{Path: "/disk/{mount_point}", Name: "Disk Information (filtered)",
		Description: "Disk usage for a specific mount point, e.g. " +
			"system:///disk/ (root) or system:///disk/boot",
		resolver: func(ctx context.Context, rest string) (any, error) {
			mountPoint := rest
			if mountPoint != "" && !strings.HasPrefix(mountPoint, "/") {
				mountPoint = "/" + mountPoint
			}
			return GatherDiskInfo(ctx, mountPoint, 0)
		}},
	{Path: "/service/{name}", Name: "Service Status",
		Description: "Detailed status of a systemd service, e.g. " +
			"system:///service/sshd or system:///service/nginx.service",
		resolver: func(ctx context.Context, rest string) (any, error) {
			return GatherServiceStatus(ctx, rest, false)
		}},
}

func RegisterResources(server *mcp.Server) {
	for i := range resourceRegistry {
		r := &resourceRegistry[i]
		if r.isTemplate() {
			server.AddResourceTemplate(&mcp.ResourceTemplate{
				URITemplate: r.URI(), Name: r.Name,
				Description: r.Description, MIMEType: "application/json",
			}, handleReadResource)
			continue
		}
		server.AddResource(&mcp.Resource{
			URI: r.URI(), Name: r.Name,
			Description: r.Description, MIMEType: "application/json",
		}, handleReadResource)
	}
}

func scheme(path string) string {
	return "system:///" + path
}

// findResource returns the registry entry matching a resource path and the
// remainder left after stripping its template prefix (empty for statics).
func findResource(path string) (*resourceDef, string) {
	for i := range resourceRegistry {
		r := &resourceRegistry[i]
		if prefix := r.prefix(); prefix != "" {
			if after, ok := strings.CutPrefix(path, prefix); ok {
				return r, after
			}
		} else if path == r.Path {
			return r, ""
		}
	}
	return nil, ""
}

func handleReadResource(
	ctx context.Context,
	req *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	uri := req.Params.URI
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid resource URI: %w", err)
	}
	if u.Scheme != "system" {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	r, rest := findResource(u.Path)
	if r == nil {
		return nil, mcp.ResourceNotFoundError(uri)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	data, nerr := r.resolver(ctx, rest)
	if data == nil {
		if nerr != nil {
			return nil, fmt.Errorf("resource unavailable: %w", nerr)
		}
		return nil, fmt.Errorf("resource unavailable: no data")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource: %w", err)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{
			{
				URI:      uri,
				MIMEType: "application/json",
				Text:     string(jsonData),
			},
		},
	}, nil
}
