package tools

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type FirewallInfo struct {
	IptablesOutput string `json:"iptables_output,omitempty"`
	NftablesOutput string `json:"nftables_output,omitempty"`
	UFWStatus      string `json:"ufw_status,omitempty"`
	Active         bool   `json:"active"`
}

type SSHHardeningInfo struct {
	PermitRootLogin        string `json:"permit_root_login,omitempty"`
	PasswordAuthentication string `json:"password_authentication,omitempty"`
	PubkeyAuthentication   string `json:"pubkey_authentication,omitempty"`
	X11Forwarding          string `json:"x11_forwarding,omitempty"`
	MaxAuthTries           string `json:"max_auth_tries,omitempty"`
	Protocol               string `json:"protocol,omitempty"`
	ConfigPresent          bool   `json:"config_present"`
}

type PasswordPolicyInfo struct {
	PassMaxDays string `json:"pass_max_days,omitempty"`
	PassMinDays string `json:"pass_min_days,omitempty"`
	PassWarnAge string `json:"pass_warn_age,omitempty"`
}

type SecurityAuditOutput struct {
	Firewall       FirewallInfo       `json:"firewall"`
	SSHHardening   SSHHardeningInfo   `json:"ssh_hardening"`
	SUIDBinaries   []string           `json:"suid_binaries"`
	WorldWritable  []string           `json:"world_writable_files"`
	Umask          string             `json:"umask"`
	PasswordPolicy PasswordPolicyInfo `json:"password_policy"`
	Score          int                `json:"security_score"`
	OutputErrors
}

func parseSSHDConfig(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	result := make(map[string]string)
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(parts[0])
		val := strings.TrimSpace(parts[1])
		// ponytail: include directive means drop-in dir has priority
		if key == "include" {
			globPattern := strings.TrimSpace(val)
			matches, _ := filepath.Glob(globPattern)
			for _, m := range matches {
				if sub := parseSSHDConfig(m); sub != nil {
					maps.Copy(result, sub)
				}
			}
			continue
		}
		result[key] = val
	}
	return result
}

func parseLoginDefs(path string) map[string]string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	result := make(map[string]string)
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func gatherFirewall(ctx context.Context) FirewallInfo {
	var info FirewallInfo

	iptOut, err := execOutput(ctx, "iptables", "-L", "-n", "--line-numbers")
	if err == nil && iptOut != "" {
		info.IptablesOutput = iptOut
		info.Active = true
	}

	nftOut, err := execOutput(ctx, "nft", "list", "ruleset")
	if err == nil && nftOut != "" {
		info.NftablesOutput = nftOut
		info.Active = true
	}

	ufwOut, err := execOutput(ctx, "ufw", "status")
	if err == nil && ufwOut != "" {
		info.UFWStatus = ufwOut
		if strings.Contains(strings.ToLower(ufwOut), "active") {
			info.Active = true
		}
	}

	return info
}

func gatherSSHHardening() SSHHardeningInfo {
	configs := []string{"/etc/ssh/sshd_config"}
	for _, path := range configs {
		vals := parseSSHDConfig(path)
		if vals == nil {
			continue
		}
		info := SSHHardeningInfo{ConfigPresent: true}
		info.PermitRootLogin = vals["permitrootlogin"]
		info.PasswordAuthentication = vals["passwordauthentication"]
		info.PubkeyAuthentication = vals["pubkeyauthentication"]
		info.X11Forwarding = vals["x11forwarding"]
		info.MaxAuthTries = vals["maxauthtries"]
		info.Protocol = vals["protocol"]
		return info
	}
	return SSHHardeningInfo{}
}

func gatherSUIDBinaries(ctx context.Context) []string {
	out, err := execOutput(ctx, "find", "/",
		"-perm", "-4000", "-type", "f",
		"-maxdepth", "4",
		"2>/dev/null")
	if err != nil || out == "" {
		return []string{}
	}
	var bins []string
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			bins = append(bins, line)
		}
	}
	return bins
}

func gatherWorldWritable(ctx context.Context) []string {
	out, err := execOutput(ctx, "find", "/etc", "/usr", "/var",
		"-perm", "-0002", "-type", "f",
		"-maxdepth", "4",
		"2>/dev/null")
	if err != nil || out == "" {
		return []string{}
	}
	var files []string
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

func gatherUmask() string {
	mask := syscall.Umask(0)
	syscall.Umask(mask)
	return fmt.Sprintf("%04o", mask)
}

func gatherPasswordPolicy() PasswordPolicyInfo {
	defs := parseLoginDefs("/etc/login.defs")
	if defs == nil {
		return PasswordPolicyInfo{}
	}
	info := PasswordPolicyInfo{}
	info.PassMaxDays = defs["PASS_MAX_DAYS"]
	info.PassMinDays = defs["PASS_MIN_DAYS"]
	info.PassWarnAge = defs["PASS_WARN_AGE"]
	return info
}

func computeSecurityScore(out *SecurityAuditOutput) int {
	score := 100

	if !out.Firewall.Active {
		score -= 15
	}

	root := strings.ToLower(out.SSHHardening.PermitRootLogin)
	if root == "yes" || root == "" && out.SSHHardening.ConfigPresent {
		score -= 20
	}

	pwd := strings.ToLower(out.SSHHardening.PasswordAuthentication)
	if pwd == "yes" {
		score -= 10
	}

	if len(out.SUIDBinaries) > 10 {
		score -= 10
	}

	if len(out.WorldWritable) > 0 {
		score -= 10
	}

	umask := strings.TrimSpace(out.Umask)
	if umask != "0027" && umask != "027" && umask != "0077" && umask != "077" {
		score -= 5
	}

	if out.PasswordPolicy.PassMaxDays != "" {
		days, err := strconv.Atoi(out.PasswordPolicy.PassMaxDays)
		if err != nil || days > 90 {
			score -= 5
		}
	} else {
		score -= 5
	}

	if score < 0 {
		score = 0
	}
	return score
}

func GatherSecurityAudit(ctx context.Context) (*SecurityAuditOutput, error) {
	out := &SecurityAuditOutput{}

	out.Firewall = gatherFirewall(ctx)
	out.SSHHardening = gatherSSHHardening()
	out.SUIDBinaries = nilToEmpty(gatherSUIDBinaries(ctx))
	out.WorldWritable = nilToEmpty(gatherWorldWritable(ctx))
	out.Umask = gatherUmask()
	out.PasswordPolicy = gatherPasswordPolicy()
	out.Score = computeSecurityScore(out)

	return out, nil
}

func HandleGetSecurityAudit(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *SecurityAuditOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetSecurityAudit,
		0,
		GatherSecurityAudit,
	)
}
