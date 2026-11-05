package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGatherTopIOProcessesLimits(t *testing.T) {
	dir := t.TempDir()
	var script strings.Builder
	script.WriteString("#!/bin/sh\nprintf '%s\\n' ")
	for i := 1; i <= 60; i++ {
		fmt.Fprintf(&script, "'12:00:00 %d %d 0 0 0 process-%d' ", i, i, i)
	}
	script.WriteByte('\n')
	if err := os.WriteFile(
		filepath.Join(dir, "pidstat"),
		[]byte(script.String()),
		0700,
	); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	for _, tc := range []struct {
		name  string
		limit int
		want  int
	}{
		{"zero", 0, 10},
		{"negative", -1, 10},
		{"one", 1, 1},
		{"maximum", 50, 50},
		{"over_maximum", 51, 50},
		{"unbounded", int(^uint(0) >> 1), 50},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := GatherTopIOProcesses(context.Background(), tc.limit)
			if err != nil {
				t.Fatal(err)
			}
			if len(out.Errors) != 0 {
				t.Fatalf("gather errors: %v", out.Errors)
			}
			if len(out.Processes) != tc.want {
				t.Fatalf(
					"got %d processes, want %d",
					len(out.Processes),
					tc.want,
				)
			}
			if out.Processes[0].PID != 60 ||
				out.Processes[tc.want-1].PID != 61-tc.want {
				t.Fatalf(
					"processes not sorted by descending I/O: %v",
					out.Processes,
				)
			}
		})
	}
}
