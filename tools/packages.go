package tools

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetInstalledPackagesInput struct {
	Name string `json:"name,omitempty" jsonschema:"optional package name filter"`
}

type InstalledPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InstalledPackagesOutput struct {
	Packages []InstalledPackage `json:"packages"`
	Total    int                `json:"total"`
	Errors   []string           `json:"errors,omitempty"`
}

type CheckUpdatesInput struct{}

type AvailableUpdate struct {
	Name    string `json:"name"`
	Current string `json:"current,omitempty"`
	New     string `json:"new,omitempty"`
}

type CheckUpdatesOutput struct {
	Updates []AvailableUpdate `json:"updates"`
	Total   int               `json:"total"`
	Errors  []string          `json:"errors,omitempty"`
}

func detectPkgManager() string {
	if _, err := exec.LookPath("pacman"); err == nil {
		return "pacman"
	}
	if _, err := exec.LookPath("dpkg"); err == nil {
		return "dpkg"
	}
	return ""
}

func GatherInstalledPackages(
	ctx context.Context,
	name string,
) (InstalledPackagesOutput, error) {
	pm := detectPkgManager()
	switch pm {
	case "pacman":
		return gatherPacmanPackages(ctx, name)
	case "dpkg":
		return gatherDpkgPackages(ctx, name)
	default:
		return InstalledPackagesOutput{}, exec.ErrNotFound
	}
}

func gatherPacmanPackages(
	ctx context.Context,
	name string,
) (InstalledPackagesOutput, error) {
	var args []string
	if name == "" {
		args = []string{"-Q"}
	} else {
		args = []string{"-Qs", name}
	}
	out, err := exec.CommandContext(ctx, "pacman", args...).Output()
	if err != nil {
		return InstalledPackagesOutput{}, err
	}
	return parsePacmanQOutput(string(out)), nil
}

func parsePacmanQOutput(output string) InstalledPackagesOutput {
	var pkgs []InstalledPackage
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, " ") {
			continue
		}
		if idx := strings.Index(line, "/"); idx >= 0 {
			line = line[idx+1:]
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pkgs = append(pkgs, InstalledPackage{
				Name:    parts[0],
				Version: parts[1],
			})
		}
	}
	return InstalledPackagesOutput{
		Packages: pkgs,
		Total:    len(pkgs),
	}
}

func gatherDpkgPackages(
	ctx context.Context,
	name string,
) (InstalledPackagesOutput, error) {
	args := []string{"-l"}
	if name != "" {
		args = []string{"-l", name}
	}
	out, err := exec.CommandContext(ctx, "dpkg", args...).Output()
	if err != nil {
		return InstalledPackagesOutput{}, err
	}
	return parseDpkgLOutput(string(out)), nil
}

func parseDpkgLOutput(output string) InstalledPackagesOutput {
	var pkgs []InstalledPackage
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		if len(line) < 4 || line[:2] != "ii" {
			continue
		}
		fields := strings.Fields(line[3:])
		if len(fields) >= 2 {
			pkgs = append(pkgs, InstalledPackage{
				Name:    fields[0],
				Version: fields[1],
			})
		}
	}
	return InstalledPackagesOutput{
		Packages: pkgs,
		Total:    len(pkgs),
	}
}

func GatherCheckUpdates(ctx context.Context) (CheckUpdatesOutput, error) {
	pm := detectPkgManager()
	switch pm {
	case "pacman":
		return gatherPacmanUpdates(ctx)
	case "dpkg":
		return gatherAptUpdates(ctx)
	default:
		return CheckUpdatesOutput{}, exec.ErrNotFound
	}
}

func gatherPacmanUpdates(ctx context.Context) (CheckUpdatesOutput, error) {
	out, err := exec.CommandContext(ctx, "pacman", "-Qu").Output()
	if err != nil {
		if len(out) == 0 {
			return CheckUpdatesOutput{}, nil
		}
	}
	return parsePacmanQuOutput(string(out)), nil
}

func parsePacmanQuOutput(output string) CheckUpdatesOutput {
	var updates []AvailableUpdate
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if before, after, ok := strings.Cut(line, " -> "); ok {
			fields := strings.Fields(before)
			if len(fields) >= 2 {
				updates = append(updates, AvailableUpdate{
					Name:    fields[0],
					Current: fields[1],
					New:     strings.TrimSpace(after),
				})
			}
		} else {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				updates = append(updates, AvailableUpdate{
					Name: fields[0],
					New:  fields[1],
				})
			}
		}
	}
	return CheckUpdatesOutput{
		Updates: updates,
		Total:   len(updates),
	}
}

func gatherAptUpdates(ctx context.Context) (CheckUpdatesOutput, error) {
	out, err := exec.CommandContext(ctx, "apt", "list", "--upgradable").Output()
	if err != nil {
		if len(out) == 0 {
			return CheckUpdatesOutput{}, err
		}
	}
	return parseAptListOutput(string(out)), nil
}

func parseAptListOutput(output string) CheckUpdatesOutput {
	var updates []AvailableUpdate
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" ||
			strings.HasPrefix(line, "Listing...") ||
			strings.HasPrefix(line, "WARNING:") {
			continue
		}
		if before, after, ok := strings.Cut(line, "/"); ok {
			name := before
			rest := after
			if before, after, ok := strings.Cut(rest, " "); ok {
				version := before
				if _, after0, ok0 := strings.Cut(after, "from: "); ok0 {
					current := after0
					current = strings.TrimRight(current, "]")
					current = strings.TrimSpace(current)
					updates = append(updates, AvailableUpdate{
						Name:    name,
						Current: current,
						New:     version,
					})
				} else {
					updates = append(updates, AvailableUpdate{
						Name: name,
						New:  version,
					})
				}
			}
		}
	}
	return CheckUpdatesOutput{
		Updates: updates,
		Total:   len(updates),
	}
}

func HandleGetInstalledPackages(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input GetInstalledPackagesInput,
) (*mcp.CallToolResult, InstalledPackagesOutput, error) {
	if config.IsDisabled("get_installed_packages") {
		return nil, InstalledPackagesOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_installed_packages", 15*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherInstalledPackages(ctx, input.Name)
	LogToolCall(ctx, "get_installed_packages",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}

func HandleCheckUpdates(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ CheckUpdatesInput,
) (*mcp.CallToolResult, CheckUpdatesOutput, error) {
	if config.IsDisabled("check_updates") {
		return nil, CheckUpdatesOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "check_updates", 15*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherCheckUpdates(ctx)
	LogToolCall(ctx, "check_updates",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
