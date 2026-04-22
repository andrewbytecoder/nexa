package collector

import (
	"github.com/nexa/pkg/ctx"
)

// NewDefaultRegistry registers collectors and placeholders based on the node_exporter README list.
// In upstream mode, all collectors are marked implemented and collected via upstream node_exporter code.
func NewDefaultRegistry(cctx *ctx.Ctx) *Registry {
	_ = cctx

	r := NewRegistry()

	enabledByDefault := []string{
		"arp", "bcache", "bonding", "btrfs", "boottime", "conntrack", "cpu", "cpufreq", "diskstats", "dmi", "edac",
		"entropy", "exec", "fibrechannel", "filefd", "filesystem", "hwmon", "infiniband", "ipvs", "kernel_hung",
		"loadavg", "mdadm", "meminfo", "netclass", "netdev", "netisr", "netstat", "nfs", "nfsd", "nvme", "os",
		"powersupplyclass", "pressure", "rapl", "schedstat", "selinux", "sockstat", "softnet", "stat", "tapestats",
		"textfile", "thermal", "thermal_zone", "time", "timex", "udp_queues", "uname", "vmstat", "watchdog", "xfs", "zfs",
	}
	r.SetLinuxEnabledByDefault(enabledByDefault)

	disabledByDefault := []string{
		"buddyinfo", "cgroups", "cpu_vulnerabilities", "devstat", "drm", "drbd", "ethtool", "interrupts", "ksmd", "lnstat",
		"logind", "meminfo_numa", "mountstats", "network_route", "pcidevice", "perf", "processes", "qdisc", "slabinfo",
		"softirqs", "sysctl", "swap", "systemd", "tcpstat", "wifi", "xfrm", "zoneinfo",
	}

	deprecated := []string{"ntp", "runit", "supervisord"}

	desc := map[string]string{
		"arp": "Exposes ARP statistics from /proc/net/arp (or via netlink).",
		"bcache": "Exposes bcache statistics from /sys/fs/bcache/.",
		"bonding": "Exposes the number of configured and active slaves of Linux bonding interfaces.",
		"btrfs": "Exposes btrfs statistics.",
		"boottime": "Exposes system boot time derived from sysctl (non-Linux upstream); may no-op on Linux.",
		"conntrack": "Shows conntrack statistics.",
		"cpu": "Exposes CPU statistics.",
		"cpufreq": "Exposes CPU frequency statistics.",
		"diskstats": "Exposes disk I/O statistics.",
		"dmi": "Expose DMI info from /sys/class/dmi/id/.",
		"edac": "Exposes error detection and correction statistics.",
		"entropy": "Exposes available entropy.",
		"exec": "Exposes execution statistics.",
		"fibrechannel": "Exposes fibre channel information and statistics.",
		"filefd": "Exposes file descriptor statistics from /proc/sys/fs/file-nr.",
		"filesystem": "Exposes filesystem statistics.",
		"hwmon": "Expose hardware monitoring and sensor data from /sys/class/hwmon/.",
		"infiniband": "Exposes network statistics specific to InfiniBand configurations.",
		"ipvs": "Exposes IPVS status and statistics.",
		"kernel_hung": "Exposes number of hung tasks.",
		"loadavg": "Exposes load average.",
		"mdadm": "Exposes statistics about devices in /proc/mdstat.",
		"meminfo": "Exposes memory statistics.",
		"netclass": "Exposes network interface info from /sys/class/net/.",
		"netdev": "Exposes network interface statistics such as bytes transferred.",
		"netisr": "Exposes netisr statistics (FreeBSD upstream); may no-op on Linux.",
		"netstat": "Exposes network statistics from /proc/net/netstat.",
		"nfs": "Exposes NFS client statistics.",
		"nfsd": "Exposes NFS kernel server statistics.",
		"nvme": "Exposes NVMe info from /sys/class/nvme/.",
		"os": "Expose OS release info.",
		"powersupplyclass": "Exposes power supply class statistics.",
		"pressure": "Exposes pressure stall information (PSI).",
		"rapl": "Exposes RAPL powercap statistics.",
		"schedstat": "Exposes task scheduler statistics from /proc/schedstat.",
		"selinux": "Exposes SELinux statistics.",
		"sockstat": "Exposes /proc/net/sockstat statistics.",
		"softnet": "Exposes /proc/net/softnet_stat statistics.",
		"stat": "Exposes /proc/stat statistics.",
		"tapestats": "Exposes statistics from /sys/class/scsi_tape.",
		"textfile": "Exposes statistics read from local disk.",
		"thermal": "Exposes thermal statistics.",
		"thermal_zone": "Exposes thermal zone & cooling device statistics.",
		"time": "Exposes the current system time.",
		"timex": "Exposes selected adjtimex(2) system call stats.",
		"udp_queues": "Exposes UDP rx/tx queue lengths.",
		"uname": "Exposes system information as provided by uname.",
		"vmstat": "Exposes statistics from /proc/vmstat.",
		"watchdog": "Exposes watchdog statistics.",
		"xfs": "Exposes XFS runtime statistics.",
		"zfs": "Exposes ZFS performance statistics.",
		// disabled-by-default
		"buddyinfo": "Exposes statistics of memory fragments from /proc/buddyinfo.",
		"cgroups": "Exposes cgroups summary.",
		"cpu_vulnerabilities": "Exposes CPU vulnerability information from sysfs.",
		"drm": "Expose GPU metrics using sysfs / DRM.",
		"drbd": "Exposes DRBD statistics.",
		"ethtool": "Exposes ethtool information and network driver statistics.",
		"interrupts": "Exposes detailed interrupts statistics.",
		"ksmd": "Exposes kernel samepage merging stats.",
		"lnstat": "Exposes stats from /proc/net/stat/.",
		"logind": "Exposes session counts from logind.",
		"meminfo_numa": "Exposes NUMA memory statistics.",
		"mountstats": "Exposes filesystem mount stats from /proc/self/mountstats.",
		"network_route": "Exposes routing table as metrics.",
		"pcidevice": "Exposes PCI device information.",
		"perf": "Exposes perf based metrics.",
		"processes": "Exposes aggregate process statistics from /proc.",
		"qdisc": "Exposes queuing discipline statistics.",
		"slabinfo": "Exposes slab statistics from /proc/slabinfo.",
		"softirqs": "Exposes detailed softirq statistics.",
		"sysctl": "Expose sysctl values from /proc/sys.",
		"swap": "Expose swap information from /proc/swaps.",
		"systemd": "Exposes service and system status from systemd.",
		"tcpstat": "Exposes TCP connection status information.",
		"wifi": "Exposes WiFi device and station statistics.",
		"xfrm": "Exposes statistics from /proc/net/xfrm_stat.",
		"zoneinfo": "Exposes NUMA memory zone metrics.",
		// deprecated
		"ntp": "Exposes NTP daemon health (deprecated upstream).",
		"runit": "Exposes service status from runit (deprecated upstream).",
		"supervisord": "Exposes service status from supervisord (deprecated upstream).",
	}

	all := append(append(append([]string(nil), enabledByDefault...), disabledByDefault...), deprecated...)
	for _, name := range all {
		r.RegisterImplemented(NewUpstreamCollector(name, desc[name]))
	}

	return r
}

