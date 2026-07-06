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
			"get_cpu_info":                 "5s",
			"get_cpu_temperature":          "5s",
			"get_memory_info":              "5s",
			"get_disk_info":                "10s",
			"get_inode_usage":              "10s",
			"get_network_info":             "5s",
			"get_system_info":              "5s",
			"get_process_info":             "10s",
			"get_process_fds":              "10s",
			"get_environment_variables":    "5s",
			"get_hardware_bus_info":        "10s",
			"get_user_automation":          "10s",
			"get_desktop_session_info":     "5s",
			"get_man_page":                 "15s",
			"get_docker_container_details": "10s",
			"get_docker_container_logs":    "30s",
			"get_docker_container_stats":   "30s",
			"get_docker_container_top":     "10s",
			"get_docker_container_diff":    "10s",
			"get_docker_image_history":     "10s",
			"get_docker_image_details":     "10s",
			"get_docker_networks":          "10s",
			"get_docker_volumes":           "10s",
			"get_docker_system_info":       "10s",
			"get_docker_disk_usage":        "10s",
			"get_docker_stats_all":         "30s",
			"get_docker_system_snapshot":   "60s",
			"get_docker_info":              "10s",
			"get_system_snapshot":          "120s",
			"get_journal_logs":             "20s",
			"get_listening_ports":          "10s",
			"get_service_status":           "10s",
			"get_top_io_processes":         "15s",
			"get_failed_logins":            "10s",
			"get_gpu_info":                 "5s",
			"get_largest_files":            "30s",
			"ping_host":                    "10s",
			"get_installed_packages":       "15s",
			"check_updates":                "15s",
			"get_load_average":             "5s",
			"get_logged_in_users":          "5s",
			"resolve_dns":                  "10s",
			"get_mount_options":            "10s",
			"get_systemd_units":            "10s",
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
