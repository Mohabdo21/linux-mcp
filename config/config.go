// Package config provides configuration loading, validation, and defaults
// for the linux-mcp MCP server.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"time"
)

// Tool name constants for consistent reference across packages.
const (
	ToolNameGetSystemInfo             = "get_system_info"
	ToolNameGetCPUInfo                = "get_cpu_info"
	ToolNameGetCPUTemperature         = "get_cpu_temperature"
	ToolNameGetMemoryInfo             = "get_memory_info"
	ToolNameGetDiskInfo               = "get_disk_info"
	ToolNameGetNetworkInfo            = "get_network_info"
	ToolNameGetProcessInfo            = "get_process_info"
	ToolNameGetDockerInfo             = "get_docker_info"
	ToolNameGetDockerContainerDetails = "get_docker_container_details"
	ToolNameGetDockerContainerLogs    = "get_docker_container_logs"
	ToolNameGetDockerContainerStats   = "get_docker_container_stats"
	ToolNameGetDockerContainerTop     = "get_docker_container_top"
	ToolNameGetDockerContainerDiff    = "get_docker_container_diff"
	ToolNameGetDockerImageHistory     = "get_docker_image_history"
	ToolNameGetDockerImageDetails     = "get_docker_image_details"
	ToolNameGetDockerNetworks         = "get_docker_networks"
	ToolNameGetDockerVolumes          = "get_docker_volumes"
	ToolNameGetDockerSystemInfo       = "get_docker_system_info"
	ToolNameGetDockerDiskUsage        = "get_docker_disk_usage"
	ToolNameGetDockerStatsAll         = "get_docker_stats_all"
	ToolNameGetDockerSystemSnapshot   = "get_docker_system_snapshot"
	ToolNameGetSystemSnapshot         = "get_system_snapshot"
	ToolNameGetJournalLogs            = "get_journal_logs"
	ToolNameGetInodeUsage             = "get_inode_usage"
	ToolNameGetNetworkConnections     = "get_network_connections"
	ToolNameGetListeningPorts         = "get_listening_ports"
	ToolNameGetServiceStatus          = "get_service_status"
	ToolNameGetProcessFDs             = "get_process_fds"
	ToolNameGetTopIOProcesses         = "get_top_io_processes"
	ToolNameGetFailedLogins           = "get_failed_logins"
	ToolNameGetGPUInfo                = "get_gpu_info"
	ToolNameGetLargestFiles           = "get_largest_files"
	ToolNamePingHost                  = "ping_host"
	ToolNameGetInstalledPackages      = "get_installed_packages"
	ToolNameCheckUpdates              = "check_updates"
	ToolNameGetLoadAverage            = "get_load_average"
	ToolNameGetLoggedInUsers          = "get_logged_in_users"
	ToolNameResolveDNS                = "resolve_dns"
	ToolNameGetMountOptions           = "get_mount_options"
	ToolNameGetSystemdUnits           = "get_systemd_units"
	ToolNameGetManPage                = "get_man_page"
	ToolNameGetEnvironmentVariables   = "get_environment_variables"
	ToolNameGetHardwareBusInfo        = "get_hardware_bus_info"
	ToolNameGetUserAutomation         = "get_user_automation"
	ToolNameGetDesktopSessionInfo     = "get_desktop_session_info"
	ToolNameGetPowerAnalytics         = "get_power_analytics"
	ToolNameGetUserInfo               = "get_user_info"
	ToolNameGetIPInfo                 = "get_ip_info"
	ToolNameGetBlockDevices           = "get_block_devices"
	ToolNameGetSELinuxAppArmorStatus  = "get_selinux_apparmor_status"
	ToolNameGetTimeSyncStatus         = "get_time_sync_status"
	ToolNameGetRAIDStatus             = "get_raid_status"
	ToolNameGetLogrotateStatus        = "get_logrotate_status"
	ToolNameGetCronJobs               = "get_cron_jobs"
	ToolNameGetSystemHealthCheck      = "get_system_health_check"
)

type Config struct {
	LogLevel string            `json:"log_level"`
	Timeouts map[string]string `json:"timeouts"`
	Disabled []string          `json:"disabled"`
}

var (
	current  atomic.Value
	filePath string
)

func envPath() string {
	if p, ok := os.LookupEnv("LINUX_MCP_CONFIG"); ok {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "linux-mcp", "config.json")
}

func defaultConfig() *Config {
	return &Config{
		LogLevel: "info",
		Timeouts: map[string]string{
			ToolNameGetCPUInfo:                "5s",
			ToolNameGetCPUTemperature:         "5s",
			ToolNameGetMemoryInfo:             "5s",
			ToolNameGetDiskInfo:               "10s",
			ToolNameGetInodeUsage:             "10s",
			ToolNameGetNetworkInfo:            "5s",
			ToolNameGetSystemInfo:             "5s",
			ToolNameGetProcessInfo:            "10s",
			ToolNameGetProcessFDs:             "10s",
			ToolNameGetEnvironmentVariables:   "5s",
			ToolNameGetHardwareBusInfo:        "10s",
			ToolNameGetUserAutomation:         "10s",
			ToolNameGetDesktopSessionInfo:     "5s",
			ToolNameGetManPage:                "15s",
			ToolNameGetDockerContainerDetails: "10s",
			ToolNameGetDockerContainerLogs:    "30s",
			ToolNameGetDockerContainerStats:   "30s",
			ToolNameGetDockerContainerTop:     "10s",
			ToolNameGetDockerContainerDiff:    "10s",
			ToolNameGetDockerImageHistory:     "10s",
			ToolNameGetDockerImageDetails:     "10s",
			ToolNameGetDockerNetworks:         "10s",
			ToolNameGetDockerVolumes:          "10s",
			ToolNameGetDockerSystemInfo:       "10s",
			ToolNameGetDockerDiskUsage:        "10s",
			ToolNameGetDockerStatsAll:         "30s",
			ToolNameGetDockerSystemSnapshot:   "60s",
			ToolNameGetDockerInfo:             "10s",
			ToolNameGetSystemSnapshot:         "120s",
			ToolNameGetJournalLogs:            "20s",
			ToolNameGetListeningPorts:         "10s",
			ToolNameGetServiceStatus:          "10s",
			ToolNameGetTopIOProcesses:         "15s",
			ToolNameGetFailedLogins:           "10s",
			ToolNameGetGPUInfo:                "5s",
			ToolNameGetLargestFiles:           "30s",
			ToolNamePingHost:                  "10s",
			ToolNameGetInstalledPackages:      "15s",
			ToolNameCheckUpdates:              "15s",
			ToolNameGetLoadAverage:            "5s",
			ToolNameGetLoggedInUsers:          "5s",
			ToolNameResolveDNS:                "10s",
			ToolNameGetMountOptions:           "10s",
			ToolNameGetSystemdUnits:           "10s",
			ToolNameGetNetworkConnections:     "10s",
			ToolNameGetPowerAnalytics:         "10s",
			ToolNameGetUserInfo:               "10s",
			ToolNameGetIPInfo:                 "10s",
			ToolNameGetBlockDevices:           "10s",
			ToolNameGetSELinuxAppArmorStatus:  "5s",
			ToolNameGetTimeSyncStatus:         "10s",
			ToolNameGetRAIDStatus:             "5s",
			ToolNameGetLogrotateStatus:        "10s",
			ToolNameGetCronJobs:               "10s",
			ToolNameGetSystemHealthCheck:      "30s",
		},
		Disabled: []string{},
	}
}

func validate(c *Config) error {
	levels := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	if _, ok := levels[c.LogLevel]; !ok {
		return fmt.Errorf("invalid log_level %q", c.LogLevel)
	}
	for name, d := range c.Timeouts {
		if _, err := time.ParseDuration(d); err != nil {
			return fmt.Errorf("invalid timeout for %q: %w", name, err)
		}
	}
	return nil
}

func loadFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = defaultConfig().LogLevel
	}
	if cfg.Timeouts == nil {
		cfg.Timeouts = map[string]string{}
	}
	if cfg.Disabled == nil {
		cfg.Disabled = []string{}
	}
	if err := validate(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Load() error {
	return loadAtPath(envPath())
}

func loadAtPath(path string) error {
	if path == "" {
		current.Store(defaultConfig())
		return nil
	}
	filePath = path
	cfg, err := loadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			current.Store(defaultConfig())
			return nil
		}
		return err
	}
	current.Store(cfg)
	return nil
}

func Reload() error {
	if filePath == "" {
		return nil
	}
	cfg, err := loadFile(filePath)
	if err != nil {
		return err
	}
	current.Store(cfg)
	return nil
}

func Get() Config {
	if cfg, ok := current.Load().(*Config); ok {
		return *cfg
	}
	return *defaultConfig()
}

func ToolTimeout(name string, fallback time.Duration) time.Duration {
	cfg := Get()
	if s, ok := cfg.Timeouts[name]; ok {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	if fallback > 0 {
		return fallback
	}
	if s, ok := defaultConfig().Timeouts[name]; ok {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return 30 * time.Second
}

func IsDisabled(name string) bool {
	cfg := Get()
	return slices.Contains(cfg.Disabled, name)
}
