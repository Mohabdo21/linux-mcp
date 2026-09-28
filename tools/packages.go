package tools

import (
	"context"
	"errors"
	"os/exec"
	"regexp"
	"strings"

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
	OutputErrors
}

type AvailableUpdate struct {
	Name    string `json:"name"`
	Current string `json:"current,omitempty"`
	New     string `json:"new,omitempty"`
}

type AvailableUpdatesOutput struct {
	Updates []AvailableUpdate `json:"updates"`
	Total   int               `json:"total"`
	OutputErrors
}

func detectPkgManager() string {
	if _, err := exec.LookPath("pacman"); err == nil {
		return "pacman"
	}
	if _, err := exec.LookPath("dpkg"); err == nil {
		return "dpkg"
	}
	if hasAnyBinary("rpm", "dnf", "yum") {
		return "rpm"
	}
	if hasAnyBinary("apk") {
		return "apk"
	}
	return ""
}

func hasAnyBinary(names ...string) bool {
	for _, name := range names {
		if _, err := exec.LookPath(name); err == nil {
			return true
		}
	}
	return false
}

func GatherInstalledPackages(
	ctx context.Context,
	name string,
) (*InstalledPackagesOutput, error) {
	pm := detectPkgManager()
	switch pm {
	case "pacman":
		return gatherPacmanPackages(ctx, name)
	case "dpkg":
		return gatherDpkgPackages(ctx, name)
	case "rpm":
		return gatherRpmPackages(ctx, name)
	case "apk":
		return gatherApkPackages(ctx, name)
	default:
		return nil, exec.ErrNotFound
	}
}

func gatherPacmanPackages(
	ctx context.Context,
	name string,
) (*InstalledPackagesOutput, error) {
	var args []string
	if name == "" {
		args = []string{"-Q"}
	} else {
		args = []string{"-Qs", name}
	}
	out, err := execOutput(ctx, "pacman", args...)
	if err != nil {
		return nil, err
	}
	return parsePacmanQOutput(out), nil
}

func parsePacmanQOutput(output string) *InstalledPackagesOutput {
	pkgs := make([]InstalledPackage, 0)
	for line := range strings.SplitSeq(output, "\n") {
		// pacman -Qs indents description lines; test before trimming.
		if line == "" || strings.HasPrefix(line, " ") {
			continue
		}
		line = strings.TrimSpace(line)
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
	return &InstalledPackagesOutput{
		Packages: pkgs,
		Total:    len(pkgs),
	}
}

func gatherDpkgPackages(
	ctx context.Context,
	name string,
) (*InstalledPackagesOutput, error) {
	args := []string{"-l"}
	if name != "" {
		args = []string{"-l", name}
	}
	out, err := execOutput(ctx, "dpkg", args...)
	if err != nil {
		return nil, err
	}
	return parseDpkgLOutput(out), nil
}

func parseDpkgLOutput(output string) *InstalledPackagesOutput {
	pkgs := make([]InstalledPackage, 0)
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
	return &InstalledPackagesOutput{
		Packages: pkgs,
		Total:    len(pkgs),
	}
}

// apk joins name and version with a dash in its human-readable output, and a
// name may itself contain -<digits>, e.g. xf86-video-r128-6.13.0-r0. So the
// version starts at the last dash followed by a digit, and the -rN release
// suffix belongs to the version. Verified against all 68490 unique packages in
// the Alpine v3.18, v3.22 and edge indexes.
var apkNameVersion = regexp.MustCompile(`^(.+)-(\d[^-]*(?:-r\d+)?)$`)

func splitApkNameVersion(token string) (string, string, bool) {
	m := apkNameVersion.FindStringSubmatch(token)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

func gatherApkPackages(
	ctx context.Context,
	name string,
) (*InstalledPackagesOutput, error) {
	// --manifest is apk-tools 3.x only, so parse the default listing instead.
	args := []string{"list", "--installed"}
	if name != "" {
		// apk patterns are globs, a bare name matches nothing.
		args = append(args, name+"*")
	}
	out, err := execOutput(ctx, "apk", args...)
	if err != nil {
		return nil, err
	}
	return parseApkListOutput(out), nil
}

func parseApkListOutput(output string) *InstalledPackagesOutput {
	pkgs := make([]InstalledPackage, 0)
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		pkgName, version, ok := splitApkNameVersion(fields[0])
		if !ok {
			continue
		}
		pkgs = append(pkgs, InstalledPackage{
			Name:    pkgName,
			Version: version,
		})
	}
	return &InstalledPackagesOutput{
		Packages: pkgs,
		Total:    len(pkgs),
	}
}

func gatherApkUpdates(ctx context.Context) (*AvailableUpdatesOutput, error) {
	// Unlike dnf, apk exits 0 whether or not updates exist.
	out, err := execOutput(ctx, "apk", "version", "-l", "<")
	if err != nil {
		return nil, err
	}
	return parseApkVersionOutput(out), nil
}

func parseApkVersionOutput(output string) *AvailableUpdatesOutput {
	updates := make([]AvailableUpdate, 0)
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "WARNING:") {
			continue
		}
		before, after, ok := strings.Cut(line, "<")
		if !ok {
			continue
		}
		pkgName, current, ok := splitApkNameVersion(strings.TrimSpace(before))
		if !ok {
			continue
		}
		updates = append(updates, AvailableUpdate{
			Name:    pkgName,
			Current: current,
			New:     strings.TrimSpace(after),
		})
	}
	return &AvailableUpdatesOutput{
		Updates: updates,
		Total:   len(updates),
	}
}

// See rpm-queryformat(7).
const rpmQueryFormat = "%{NAME} %{VERSION}-%{RELEASE}\n"

func gatherRpmPackages(
	ctx context.Context,
	name string,
) (*InstalledPackagesOutput, error) {
	args := []string{"-qa", "--qf", rpmQueryFormat}
	if name != "" {
		args = append(args, `name="`+name+`*"`)
	}
	out, err := execOutput(ctx, "rpm", args...)
	if err != nil {
		return nil, err
	}
	return parseRpmQOutput(out), nil
}

func parseRpmQOutput(output string) *InstalledPackagesOutput {
	pkgs := make([]InstalledPackage, 0)
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pkgs = append(pkgs, InstalledPackage{
			Name:    fields[0],
			Version: fields[1],
		})
	}
	return &InstalledPackagesOutput{
		Packages: pkgs,
		Total:    len(pkgs),
	}
}

func GatherAvailableUpdates(
	ctx context.Context,
) (*AvailableUpdatesOutput, error) {
	pm := detectPkgManager()
	switch pm {
	case "pacman":
		return gatherPacmanUpdates(ctx)
	case "dpkg":
		return gatherAptUpdates(ctx)
	case "rpm":
		return gatherRpmUpdates(ctx)
	case "apk":
		return gatherApkUpdates(ctx)
	default:
		return nil, exec.ErrNotFound
	}
}

func gatherPacmanUpdates(ctx context.Context) (*AvailableUpdatesOutput, error) {
	out, err := execOutput(ctx, "pacman", "-Qu")
	if err != nil {
		if out == "" {
			return &AvailableUpdatesOutput{Updates: []AvailableUpdate{}}, nil
		}
	}
	return parsePacmanQuOutput(out), nil
}

func parsePacmanQuOutput(output string) *AvailableUpdatesOutput {
	updates := make([]AvailableUpdate, 0)
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
	return &AvailableUpdatesOutput{
		Updates: updates,
		Total:   len(updates),
	}
}

func gatherAptUpdates(ctx context.Context) (*AvailableUpdatesOutput, error) {
	out, err := execOutput(ctx, "apt", "list", "--upgradable")
	if err != nil {
		if out == "" {
			return nil, err
		}
	}
	return parseAptListOutput(out), nil
}

func parseAptListOutput(output string) *AvailableUpdatesOutput {
	updates := make([]AvailableUpdate, 0)
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
			fields := strings.Fields(rest)
			if len(fields) >= 2 {
				version := fields[1]
				current := ""
				for i, f := range fields[2:] {
					if f == "from:" && i+3 < len(fields) {
						current = strings.TrimRight(fields[i+3], "]")
						break
					}
				}
				updates = append(updates, AvailableUpdate{
					Name:    name,
					Current: current,
					New:     version,
				})
			}
		}
	}
	return &AvailableUpdatesOutput{
		Updates: updates,
		Total:   len(updates),
	}
}

// dnf: 0 = no updates, 100 = updates available, 1 = error. yum matches.
const dnfUpdatesExit = 100

func gatherRpmUpdates(ctx context.Context) (*AvailableUpdatesOutput, error) {
	bin := ""
	if hasAnyBinary("dnf") {
		bin = "dnf"
	} else if hasAnyBinary("yum") {
		bin = "yum"
	}
	if bin == "" {
		return nil, exec.ErrNotFound
	}

	out, err := execOutput(ctx, bin, "check-update")
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != dnfUpdatesExit {
			return nil, err
		}
	}
	return parseDnfCheckUpdateOutput(out), nil
}

func parseDnfCheckUpdateOutput(output string) *AvailableUpdatesOutput {
	updates := make([]AvailableUpdate, 0)
	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" ||
			strings.HasPrefix(line, "Last metadata expiration") ||
			strings.HasPrefix(line, "Obsoleting Packages") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		updates = append(updates, AvailableUpdate{
			Name: stripRPMArch(fields[0]),
			New:  fields[1],
		})
	}
	return &AvailableUpdatesOutput{
		Updates: updates,
		Total:   len(updates),
	}
}

// Split on the last dot: package names can contain one, e.g. python3.11.x86_64.
var rpmArches = map[string]bool{
	"noarch": true, "x86_64": true, "x86": true,
	"i386": true, "i486": true, "i586": true, "i686": true,
	"aarch64": true, "armv7hl": true, "armv6hl": true,
	"ppc64": true, "ppc64le": true, "s390x": true,
	"riscv64": true, "src": true,
}

func stripRPMArch(name string) string {
	idx := strings.LastIndex(name, ".")
	if idx < 0 || !rpmArches[name[idx+1:]] {
		return name
	}
	return name[:idx]
}

func HandleGetInstalledPackages(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetInstalledPackagesInput,
) (*mcp.CallToolResult, *InstalledPackagesOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetInstalledPackages,
		0,
		func(ctx context.Context) (*InstalledPackagesOutput, error) {
			return GatherInstalledPackages(ctx, input.Name)
		},
	)
}

func HandleGetAvailableUpdates(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *AvailableUpdatesOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetAvailableUpdates,
		0,
		GatherAvailableUpdates,
	)
}
