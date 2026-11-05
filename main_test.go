package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMainConfig(t *testing.T) {
	if os.Getenv("LINUX_MCP_TEST_MAIN") == "1" {
		main()
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		config string
	}{
		{name: "malformed", config: `{`},
		{name: "invalid_log_level", config: `{"log_level":"trace"}`},
		{name: "invalid_timeout", config: `{"timeouts":{"get_cpu_info":"bad"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tc.config), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("LINUX_MCP_CONFIG", path)
			t.Setenv("LINUX_MCP_TEST_MAIN", "1")
			ctx, cancel := context.WithTimeout(
				context.Background(),
				10*time.Second,
			)
			defer cancel()
			cmd := exec.CommandContext(
				ctx,
				executable,
				"-test.run=^TestMainConfig$",
			)
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("server did not exit: %v\n%s", ctx.Err(), output)
			}
			var exitErr *exec.ExitError
			if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("got error %v, want exit status 1\n%s", err, output)
			}
			if strings.Contains(string(output), "server starting") {
				t.Fatalf("server started with invalid config:\n%s", output)
			}
			if !strings.Contains(string(output), "config load failed") {
				t.Fatalf("config failure was not logged:\n%s", output)
			}
		})
	}
}
