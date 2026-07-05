package tools

import (
	"strings"
	"testing"
)

// skipOnErr skips the test if err is non-nil with a formatted message.
func skipOnErr(t *testing.T, err error, format string, args ...any) {
	t.Helper()
	if err != nil {
		t.Skipf(format, args...)
	}
}

// checkNotEmpty fails the test if s is empty.
func checkNotEmpty(t *testing.T, s, name string) {
	t.Helper()
	if s == "" {
		t.Errorf("%s should not be empty", name)
	}
}

// checkNotZeroUint64 fails the test if val == 0.
func checkNotZeroUint64(t *testing.T, val uint64, name string) {
	t.Helper()
	if val == 0 {
		t.Errorf("%s should not be 0", name)
	}
}

// checkNotNegative fails the test if val < 0.
func checkNotNegative(t *testing.T, val float64, name string) {
	t.Helper()
	if val < 0 {
		t.Errorf("%s should be >= 0, got %f", name, val)
	}
}

// requireDockerContainers returns Docker containers or skips the test.
func requireDockerContainers(t *testing.T) []DockerContainer {
	t.Helper()
	containers, err := ListDockerContainers(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	if len(containers) == 0 {
		t.Skip("No containers available")
	}
	return containers
}

// requireDockerRunningContainer returns the ID of a running container
// or skips the test.
func requireDockerRunningContainer(t *testing.T) string {
	t.Helper()
	containers, err := ListDockerContainers(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	for _, c := range containers {
		if strings.Contains(c.Status, "Up") {
			return c.ID
		}
	}
	t.Skip("No running container available")
	return ""
}

// requireDockerImages returns Docker images or skips the test.
func requireDockerImages(t *testing.T) []DockerImage {
	t.Helper()
	images, err := ListDockerImages(t.Context())
	if err != nil {
		t.Skipf("Docker not available: %v", err)
	}
	if len(images) == 0 {
		t.Skip("No images available")
	}
	return images
}
