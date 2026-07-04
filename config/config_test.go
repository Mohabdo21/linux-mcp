package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaults(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	cfg := Get()
	if cfg.LogLevel != "info" {
		t.Errorf("expected info, got %s", cfg.LogLevel)
	}
	if cfg.Timeouts["get_cpu_info"] != "5s" {
		t.Errorf("expected 5s, got %s", cfg.Timeouts["get_cpu_info"])
	}
	if len(cfg.Disabled) != 0 {
		t.Errorf("expected empty disabled, got %v", cfg.Disabled)
	}
}

func TestToolTimeout(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	d := ToolTimeout("get_cpu_info", 10*time.Second)
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
	d = ToolTimeout("nonexistent", 10*time.Second)
	if d != 10*time.Second {
		t.Errorf("expected 10s, got %v", d)
	}
}

func TestIsDisabled(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(
		p, []byte(`{"disabled":["ping_host"]}`), 0644,
	); err != nil {
		t.Fatal(err)
	}
	if err := loadAtPath(p); err != nil {
		t.Fatal(err)
	}
	if !IsDisabled("ping_host") {
		t.Error("expected ping_host disabled")
	}
	if IsDisabled("get_cpu_info") {
		t.Error("expected get_cpu_info enabled")
	}
}

func TestInvalidLogLevel(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(
		p, []byte(`{"log_level":"trace"}`), 0644,
	); err != nil {
		t.Fatal(err)
	}
	err := loadAtPath(p)
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
}

func TestInvalidTimeout(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(
		p, []byte(`{"timeouts":{"get_cpu_info":"not-a-duration"}}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}
	err := loadAtPath(p)
	if err == nil {
		t.Fatal("expected error for invalid timeout")
	}
}

func TestEnvOverride(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(
		p, []byte(`{"log_level":"debug"}`), 0644,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINUX_MCP_CONFIG", p)
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	cfg := Get()
	if cfg.LogLevel != "debug" {
		t.Errorf("expected debug, got %s", cfg.LogLevel)
	}
}

func TestMissingConfigIsOk(t *testing.T) {
	if err := loadAtPath("/nonexistent/path/config.json"); err != nil {
		t.Fatal(err)
	}
	cfg := Get()
	if cfg.LogLevel != "info" {
		t.Errorf("expected info, got %s", cfg.LogLevel)
	}
}

func TestReload(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	if err := os.WriteFile(
		p, []byte(`{"log_level":"debug"}`), 0644,
	); err != nil {
		t.Fatal(err)
	}
	if err := loadAtPath(p); err != nil {
		t.Fatal(err)
	}
	if Get().LogLevel != "debug" {
		t.Fatal("expected debug after load")
	}
	if err := os.WriteFile(
		p, []byte(`{"log_level":"warn"}`), 0644,
	); err != nil {
		t.Fatal(err)
	}
	if err := Reload(); err != nil {
		t.Fatal(err)
	}
	if Get().LogLevel != "warn" {
		t.Errorf("expected warn after reload, got %s", Get().LogLevel)
	}
}

func TestToolTimeoutFallback(t *testing.T) {
	if err := Load(); err != nil {
		t.Fatal(err)
	}
	d := ToolTimeout("get_cpu_info", 10*time.Second)
	if d != 5*time.Second {
		t.Errorf("expected 5s, got %v", d)
	}
	d = ToolTimeout("nonexistent_tool", 10*time.Second)
	if d != 10*time.Second {
		t.Errorf("expected 10s, got %v", d)
	}
}
