package tools

import (
	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerTool[In, Out any](
	s *mcp.Server,
	name, description string,
	h mcp.ToolHandlerFor[In, Out],
) {
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: description}, h)
}

func RegisterTools(server *mcp.Server) {
	registerTool(
		server,
		config.ToolNameGetSystemInfo,
		"Returns host info: hostname, OS, kernel, architecture, uptime, process count, boot time, virtualization, host UUID, and DMI hardware/BIOS/TPM details. Read-only (gopsutil host.Info plus sysfs). Fails only if host info is unavailable. Use for a single-machine identity summary; for a health verdict use get_system_health_check.",
		HandleGetSystemInfo,
	)
	registerTool(
		server,
		config.ToolNameGetCPUInfo,
		"Returns CPU model, frequency, core count, and current usage percent. Read-only from /proc/stat and /proc/cpuinfo. Fatal only if either source is unreadable. Use for capacity planning or busy/idle checks.",
		HandleGetCPUInfo,
	)
	registerTool(
		server,
		config.ToolNameGetCPUTemperature,
		"Returns current CPU temperature from hwmon sensors if available. Read-only. If no sensors are exposed, returns a message in the errors field instead of failing. Use for thermal monitoring; prefer get_system_health_check for an overall verdict.",
		HandleGetCPUTemperature,
	)
	registerTool(
		server,
		config.ToolNameGetMemoryInfo,
		"Returns RAM and swap usage including total, used, and free. Read-only from /proc/meminfo. Fatal only if /proc/meminfo is unreadable. Use for memory pressure checks; get_system_health_check for thresholds.",
		HandleGetMemoryInfo,
	)
	registerTool(
		server,
		config.ToolNameGetDiskInfo,
		"Returns disk usage per mounted partition. Params: mount_point (exact match, must start with /) and threshold (only partitions at or above it). Read-only via statfs. Fatal only if partitions cannot be listed; per-partition read errors are skipped. Use for capacity; get_inode_usage when 'disk full' persists with free space.",
		HandleGetDiskInfo,
	)
	registerTool(
		server,
		config.ToolNameGetNetworkInfo,
		"Returns per-interface network I/O counters (bytes, packets, errors, drops). Read-only from /proc/net/dev. Fatal only if that file is unreadable. Use for bandwidth and drop analysis; get_network_connections for sockets.",
		HandleGetNetworkInfo,
	)
	registerTool(
		server,
		config.ToolNameGetProcessInfo,
		"Returns running processes sortable by CPU, memory, or both with a configurable limit (default 10, max 100). Read-only but heavy: walks /proc and reads smaps for every process, so keep limit low. Per-process read errors are ignored; fails only if the process list cannot be read. Prefer get_top_io_processes for disk I/O ranking.",
		HandleGetProcessInfo,
	)
	registerTool(
		server,
		config.ToolNameGetDockerInfo,
		"Returns Docker containers (including stopped) and images. Read-only via the Docker API. Fatal if the daemon is unreachable. Use for a Docker inventory; get_docker_system_snapshot for a fuller view.",
		HandleGetDockerInfo,
	)
	registerTool(
		server,
		config.ToolNameGetDockerContainerDetails,
		"Returns a container's state, config, env, mounts, and network settings. Read-only via the Docker API; fatal if the daemon is unreachable or the container is unknown. Use to inspect one container; get_docker_info to list them.",
		HandleGetContainerDetail,
	)
	registerTool(
		server,
		config.ToolNameGetDockerContainerLogs,
		"Returns a container's stdout/stderr log lines. Params: tail (line count) and timestamps. Read-only via the Docker API. Fatal if the daemon is unreachable or the container is unknown. Environment values are redacted.",
		HandleGetContainerLogs,
	)
	registerTool(
		server,
		config.ToolNameGetDockerContainerStats,
		"Returns live CPU, memory, network I/O, and PID stats for running containers. Params: container_ids comma-separated or 'all' (running only). Read-only, one-shot sample. Stats only exist for running containers; unknown or stopped ids error. For all containers in one call use get_docker_stats_all.",
		HandleGetContainerStats,
	)
	registerTool(
		server,
		config.ToolNameGetDockerContainerTop,
		"Returns the processes running inside a container. Read-only via the Docker API; fatal if the daemon is unreachable or the container is unknown. Use to debug container-level process state.",
		HandleGetContainerTop,
	)
	registerTool(
		server,
		config.ToolNameGetDockerContainerDiff,
		"Returns filesystem changes (added, modified, deleted) in a container since it started. Read-only via the Docker API. Use to see what a container wrote to its writable layer.",
		HandleGetContainerDiff,
	)
	registerTool(
		server,
		config.ToolNameGetDockerImageHistory,
		"Returns an image's layer history: commands, sizes, and creation times. Read-only via the Docker API. Use to understand what an image is built from; get_docker_image_details for config and labels.",
		HandleGetImageHistory,
	)
	registerTool(
		server,
		config.ToolNameGetDockerImageDetails,
		"Returns an image's config, env, entrypoint, labels, and layers. Read-only via the Docker API; fatal if the daemon is unreachable or the image is unknown. Use to inspect a single image; get_docker_image_history for build steps.",
		HandleGetImageDetail,
	)
	registerTool(
		server,
		config.ToolNameGetDockerNetworks,
		"Returns Docker networks with driver, scope, and configuration details. Read-only via the Docker API. Use for network topology debugging.",
		HandleGetDockerNetworks,
	)
	registerTool(
		server,
		config.ToolNameGetDockerVolumes,
		"Returns Docker volumes with driver, mountpoint, size, and labels. Read-only via the Docker API. Use to inventory storage; get_docker_disk_usage for space accounting.",
		HandleGetDockerVolumes,
	)
	registerTool(
		server,
		config.ToolNameGetDockerSystemInfo,
		"Returns Docker daemon info: version, storage driver, runtimes, and resource counts. Read-only via the Docker API; fatal if the daemon is unreachable. Use to check daemon config and health.",
		HandleGetDockerSystemInfo,
	)
	registerTool(
		server,
		config.ToolNameGetDockerDiskUsage,
		"Returns Docker disk usage broken down by containers, images, volumes, and build cache. Read-only via the Docker API. Use to find what is consuming disk space.",
		HandleGetDockerDiskUsage,
	)
	registerTool(
		server,
		config.ToolNameGetDockerStatsAll,
		"Returns CPU, memory, network, and block I/O for all running containers in one call. Optional containers param filters by name or ID. Read-only, one-shot. Use instead of repeated get_docker_container_stats calls for a fleet overview.",
		HandleGetDockerStatsAll,
	)
	registerTool(
		server,
		config.ToolNameGetDockerSystemSnapshot,
		"Returns a combined Docker snapshot: containers, images, running stats, disk usage, and networks. Read-only via the Docker API. Per-part failures land in the errors field; the call still succeeds. Heaviest Docker call - prefer narrower get_docker_* tools for a specific question.",
		HandleGetDockerSystemSnapshot,
	)
	registerTool(
		server,
		config.ToolNameGetSystemSnapshot,
		"Returns a comprehensive snapshot combining system, CPU, temperature, memory, disk, network, load, top processes, and Docker data in one call. Read-only. Individual gather failures fall back to zero values in the errors field - never fatal, but the heaviest call in the server. Use for a broad overview; prefer targeted tools for deep questions.",
		HandleGetSystemSnapshot,
	)
	registerTool(
		server,
		config.ToolNameGetJournalLogs,
		"Reads systemd journal entries with optional filtering by unit, priority, and time range (since/until). Set user=true for the user journal. Read-only via journalctl. Fatal only if journalctl fails or an invalid unit is given. Returns structured entries with timestamp, message, priority, unit, and PID. For kernel-only events use get_audit_logs.",
		HandleGetJournalLogs,
	)
	registerTool(
		server,
		config.ToolNameGetInodeUsage,
		"Returns inode usage per filesystem via df -i. mount_point is an optional exact filter. Read-only; fatal if df is missing or fails. Use when 'disk full' errors persist despite free space - inode exhaustion.",
		HandleGetInodeUsage,
	)
	registerTool(
		server,
		config.ToolNameGetNetworkConnections,
		"Returns active TCP/UDP connections with state, addresses, process info, and optional reverse-DNS hostnames. Read-only from /proc/net. Params: status and type filters, grouped by PID, max_connections (0-200). resolve_hostnames does network lookups and is slow - keep it off unless needed. For listening sockets prefer get_listening_ports, which also gives process names.",
		HandleGetNetworkConnections,
	)
	registerTool(
		server,
		config.ToolNameGetListeningPorts,
		"Returns listening TCP/UDP ports and associated processes via ss -tulnp. protocol filters to tcp or udp. Read-only; fatal if ss is missing. Process names may be empty without root. Use for port-conflict and exposure checks; get_network_connections for established connections.",
		HandleGetListeningPorts,
	)
	registerTool(
		server,
		config.ToolNameGetServiceStatus,
		"Returns detailed status of a systemd service (or --user service). name is required and validated. Read-only via systemctl status. Errors from systemctl (e.g. unit not found) appear in the errors field with the raw output; only a missing systemctl is fatal. Use to check why a service failed; get_systemd_units to list states.",
		HandleGetServiceStatus,
	)
	registerTool(
		server,
		config.ToolNameGetProcessFDs,
		"Lists open file descriptors (files, sockets, pipes) and the total count for a pid. Read-only via /proc/<pid>/fd. pid is required; an unknown pid returns an error in the errors field, not a failure. Use for fd-leak and resource-hold debugging.",
		HandleGetProcessFDs,
	)
	registerTool(
		server,
		config.ToolNameGetTopIOProcesses,
		"Returns processes with the highest disk I/O activity via pidstat -d 1 1 (samples for 1 second). Read-only. If pidstat is missing, returns an error in the errors field with empty results. Use to find which process is hammering the disk; get_disk_io_metrics for per-device totals.",
		HandleGetTopIOProcesses,
	)
	registerTool(
		server,
		config.ToolNameGetFailedLogins,
		"Returns recent failed login attempts (excluding Boot records) with summary statistics. Read-only: lastb -n N, falling back to journalctl if btmp is not readable. Default 20 entries. Errors are non-fatal. Use for security triage; get_audit_logs for kernel and auditd events.",
		HandleGetFailedLogins,
	)
	registerTool(
		server,
		config.ToolNameGetGPUInfo,
		"Returns GPU usage, memory, temperature, and power draw. Read-only, via nvidia-smi, rocm-smi, or intel_gpu_top in that order of availability. Fatal only if no GPU tool is installed. The Intel fallback reports presence, not metrics. Use for GPU workload monitoring.",
		HandleGetGPUInfo,
	)
	registerTool(
		server,
		config.ToolNameGetLargestFiles,
		"Lists the top N largest entries in a directory by size (default path '.', limit 10-100). Read-only and non-recursive - only one directory level, and directory sizes show their own stat size, not contents. Failures silently return empty results. Use for quick space triage; get_disk_info for partition-level capacity.",
		HandleGetLargestFiles,
	)
	registerTool(
		server,
		config.ToolNamePingHost,
		"Sends ICMP packets and returns latency, packet loss, and response times. host is required and validated; count defaults to 4, timeout to 10s. Read-only but network-bound, blocking up to timeout seconds. Fatal only for an invalid host; a failed ping returns partial results in the errors field. Use for reachability checks; resolve_dns to separate DNS from connectivity.",
		HandlePingHost,
	)
	registerTool(
		server,
		config.ToolNameGetInstalledPackages,
		"Queries installed packages: pacman -Q on Arch, dpkg -l on Debian. Optional name filter. Read-only. Fatal if the package manager is unsupported (e.g. rpm/dnf) or the query fails. Use to check what is installed; check_updates for available upgrades.",
		HandleGetInstalledPackages,
	)
	registerTool(
		server,
		config.ToolNameCheckUpdates,
		"Counts or lists available package updates without applying them (pacman -Qu or apt list --upgradable). Read-only, no cache refresh. Fatal if the package manager is unsupported; a pacman error with empty output returns an empty list. Use to see pending updates; get_installed_packages for current versions.",
		HandleCheckUpdates,
	)
	registerTool(
		server,
		config.ToolNameGetLoadAverage,
		"Returns 1-, 5-, and 15-minute load averages. Read-only from /proc/loadavg; fatal only if unreadable. Use as a quick utilization check; get_system_health_check compares load against core count.",
		HandleGetLoadAverage,
	)
	registerTool(
		server,
		config.ToolNameGetLoggedInUsers,
		"Returns active user sessions via who -u: username, terminal, origin, and login time. Read-only; fatal only if who is missing. Use for security awareness and multi-user workload checks.",
		HandleGetLoggedInUsers,
	)
	registerTool(
		server,
		config.ToolNameResolveDNS,
		"Resolves a hostname to IP addresses via the system resolver. hostname is required; a lookup failure returns an error in the errors field. Network-dependent. Use to distinguish DNS failures from connectivity problems; ping_host to test reachability.",
		HandleResolveDNS,
	)
	registerTool(
		server,
		config.ToolNameGetMountOptions,
		"Returns mount sources, targets, filesystem types, and options via findmnt. mount_point filters to an exact target. Read-only; fatal if findmnt is missing. Use for mount flags (rw/ro, noexec, etc.); get_disk_info for usage.",
		HandleGetMountOptions,
	)
	registerTool(
		server,
		config.ToolNameGetSystemdUnits,
		"Lists all systemd units and their states. The state param filters by exact match on the Active column ('failed', 'active', 'inactive'). Read-only via systemctl; fatal if systemctl is missing or fails. Use for a full service inventory; get_service_status for one unit's details.",
		HandleGetSystemdUnits,
	)
	registerTool(
		server,
		config.ToolNameGetBootTime,
		"Returns the boot-time breakdown from systemd-analyze time: per-phase durations (firmware, loader, kernel, initrd, userspace), total startup time, and the reached target. Read-only; fatal only if systemd-analyze is missing or fails. Use to quantify where boot time goes.",
		HandleGetBootTime,
	)
	registerTool(
		server,
		config.ToolNameGetBootBlame,
		"Returns systemd units ordered by init time from systemd-analyze blame: unit names with their start durations. Read-only; fatal only if systemd-analyze is missing or fails. Use to find slow-starting services; get_boot_critical_chain for the dependency chain.",
		HandleGetBootBlame,
	)
	registerTool(
		server,
		config.ToolNameGetBootCriticalChain,
		"Returns the time-critical boot chain from systemd-analyze critical-chain: a dependency tree of units with their active-time point and start duration. Optional unit param starts the chain from a specific unit. Read-only; fatal only if systemd-analyze is missing or fails. Use to find what delays the boot target.",
		HandleGetBootCriticalChain,
	)
	registerTool(
		server,
		config.ToolNameGetManPage,
		"Fetches the authoritative man page for a command as plain text. command is required and validated. Read-only via man -P cat. Options: max_lines (500-10000) with a truncated flag, case-insensitive search with context_lines, and offset. Fatal if man is missing or there is no manual entry. Use when the user asks about flags, syntax, or edge cases.",
		HandleGetManPage,
	)
	registerTool(
		server,
		config.ToolNameGetEnvironmentVariables,
		"Returns the server process's environment as a sorted key-value map. search filters by name prefix or substring, case-insensitive. Read-only (os.Environ). Sensitive names (SECRET, TOKEN, PASSWORD, etc.) are redacted to ***. Use to debug PATH, locale, and server configuration.",
		HandleGetEnvironmentVariables,
	)
	registerTool(
		server,
		config.ToolNameGetHardwareBusInfo,
		"Lists PCI and USB devices detected on the system. Read-only via lspci/lsusb with a sysfs fallback; if both fail the error goes in the errors field. search filters any field (bus, slot, class, vendor, device). Use to identify network cards, audio interfaces, and expansion cards for driver troubleshooting.",
		HandleGetHardwareBusInfo,
	)
	registerTool(
		server,
		config.ToolNameGetUserAutomation,
		"Aggregates the current user's scheduled tasks: crontab -l entries and systemd user timers. Read-only; failures such as a missing crontab land in the errors field. Use for user-level automation; get_cron_jobs for system-wide cron.",
		HandleGetUserAutomation,
	)
	registerTool(
		server,
		config.ToolNameGetDesktopSessionInfo,
		"Returns display protocol (Wayland/X11), desktop environment, and related environment config. Read-only from environment variables; never fails. Use to understand the GUI session the server runs under.",
		HandleGetDesktopSessionInfo,
	)
	registerTool(
		server,
		config.ToolNameGetPowerAnalytics,
		"Returns power state (AC vs battery), discharge rate in watts, battery percentage, and capacity degradation. Read-only from /sys/class/power_supply. Missing sysfs data lands in the errors field. Use for laptop power monitoring.",
		HandleGetPowerAnalytics,
	)
	registerTool(
		server,
		config.ToolNameGetUserInfo,
		"Lists system users from /etc/passwd and /etc/group: username, UID, GID, home, shell, and supplementary groups. search does a case-insensitive substring match. Read-only; a passwd read failure is non-fatal. Use for account inventory and membership checks.",
		HandleGetUserInfo,
	)
	registerTool(
		server,
		config.ToolNameGetIPInfo,
		"Returns geolocation, ASN/organization, and provider tags (e.g. AWS, Cloudflare) for an IP or your public IP. Makes an external call to ip-api.com - network-dependent and can be slow offline. Invalid IPs return an error in the errors field. Use for network egress and remote-peer context.",
		HandleGetIPInfo,
	)
	registerTool(
		server,
		config.ToolNameGetBlockDevices,
		"Returns block devices and partitions: names, sizes, filesystem types, and mount points. Read-only from sysfs and /proc/mounts. Loop, ram, and zram devices are skipped. Errors land in the errors field. Use for storage inventory; get_disk_info for usage percent.",
		HandleGetBlockDevices,
	)
	registerTool(
		server,
		config.ToolNameGetSELinuxAppArmorStatus,
		"Returns SELinux and AppArmor enforcement status. Read-only via getenforce and sysfs/aa-status; never fatal, missing modules report 'not_enabled'. Use for security posture checks.",
		HandleGetSELinuxAppArmorStatus,
	)
	registerTool(
		server,
		config.ToolNameGetTimeSyncStatus,
		"Returns NTP/Chrony sync state: service, sync status, system and RTC time, stratum, and last offset. Read-only via timedatectl and chronyc; failures are non-fatal (errors field). Use for clock-drift diagnosis.",
		HandleGetTimeSyncStatus,
	)
	registerTool(
		server,
		config.ToolNameGetRAIDStatus,
		"Returns software RAID status from /proc/mdstat: devices, levels, sizes, and active/degraded/inactive health. Read-only; an unreadable file is silently swallowed. Use to check array health.",
		HandleGetRAIDStatus,
	)
	registerTool(
		server,
		config.ToolNameGetLogrotateStatus,
		"Returns logrotate configuration files and the state file path. Read-only from /etc/logrotate.conf and /etc/logrotate.d. Errors land in the errors field. Use to verify rotation policies.",
		HandleGetLogrotateStatus,
	)
	registerTool(
		server,
		config.ToolNameGetCronJobs,
		"Returns system-level cron jobs from /etc/crontab and the periodic cron directories (/etc/cron.daily, .weekly, .hourly). Read-only; no root needed. Missing paths land in the errors field. Use for system automation; get_user_automation for user-level tasks.",
		HandleGetCronJobs,
	)
	registerTool(
		server,
		config.ToolNameGetSystemHealthCheck,
		"Returns an overall OK/WARNING/CRITICAL verdict from memory, disk (partitions at 80%+), load vs core count, and failed systemd units. Read-only aggregate; sub-failures land in the errors field, never fatal. Use as the first-line health check; get_system_snapshot for full detail.",
		HandleGetSystemHealthCheck,
	)
	registerTool(
		server,
		config.ToolNameGetSMARTHealth,
		"Returns SMART disk health: status, temperature, power-on hours, and key attributes via smartctl. device is optional; empty probes all block devices, which is slow with many disks. Requires smartctl and typically root to read /dev. Per-device failures set status to unknown in the errors field; only a missing smartctl is fatal.",
		HandleGetSMARTHealth,
	)
	registerTool(
		server,
		config.ToolNameGetSecurityAudit,
		"Runs a security audit: firewall rules, SSH hardening, SUID binaries, world-writable files, umask, and password policy, with a 0-100 score. Read-only but heavy - scans the filesystem for SUID and world-writable files. Needs root to see everything. Use for hardening reviews, ideally during low load.",
		HandleGetSecurityAudit,
	)
	registerTool(
		server,
		config.ToolNameGetDiskIOMetrics,
		"Returns per-device disk I/O: reads, writes, sectors, and timings. Read-only from /proc/diskstats (loop/ram/zram skipped); fatal only if unreadable. Use for storage performance; get_top_io_processes for per-process I/O.",
		HandleGetDiskIOMetrics,
	)
	registerTool(
		server,
		config.ToolNameGetProcDiagnostics,
		"Returns deep /proc diagnostics: interrupts, softirqs, vmstat, diskstats, filesystems, version, and slabinfo. The sections param selects a comma-separated subset (empty = all). Read-only; per-section failures land in the errors field. Use for kernel-level debugging.",
		HandleGetProcDiagnostics,
	)
	registerTool(
		server,
		config.ToolNameGetAuditLogs,
		"Returns kernel audit events from journalctl -k or /var/log/audit/audit.log (AVC denials, system calls). source param: journalctl, audit.log, or auto (default). lines defaults to 50. Read-only; fatal if the chosen source is unavailable. Use for security forensics; get_failed_logins for login attempts.",
		HandleGetAuditLogs,
	)
	registerTool(
		server,
		config.ToolNameGetFileLocks,
		"Returns active file locks from /proc/locks: type, mode, PID, byte range, and path. Read-only; an unreadable file is silently swallowed, parse errors go in the errors field. Use to find who holds a lock.",
		HandleGetFileLocks,
	)
	registerTool(
		server,
		config.ToolNameGetSharedMemorySegments,
		"Returns System V shared memory segments from /proc/sysvipc/shm: key, size, attached processes, and timestamps. Read-only; an unreadable file is silently swallowed. Use for IPC and leak investigation.",
		HandleGetSharedMemorySegments,
	)
	registerTool(
		server,
		config.ToolNameGetProcessTree,
		"Returns a flat process list with depth info showing parent-child hierarchy. Optional pid param shows only the subtree rooted at that PID. Read-only from /proc/*/stat and /proc/*/comm. Use to understand process hierarchy and trace spawned children.",
		HandleGetProcessTree,
	)
	registerTool(
		server,
		config.ToolNameGetKernelModules,
		"Returns loaded kernel modules with size, reference count, and dependency info. Read-only from /proc/modules. Use for driver and module troubleshooting.",
		HandleGetKernelModules,
	)
	registerTool(
		server,
		config.ToolNameGetRoutingTable,
		"Returns the kernel routing table: destination, gateway, interface, proto, scope, metric, and MTU. Read-only via ip route show. Use for network path and gateway troubleshooting.",
		HandleGetRoutingTable,
	)
	registerTool(
		server,
		config.ToolNameGetIOStats,
		"Returns per-device extended I/O statistics: throughput, IOPS, latency, queue depth, and utilization. Read-only via iostat -xd 1 1. Requires sysstat/iostat installed. Use for storage performance analysis; get_disk_io_metrics for cumulative counters.",
		HandleGetIOStats,
	)
}
