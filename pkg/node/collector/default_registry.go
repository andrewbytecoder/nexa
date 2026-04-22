package collector

import (
	"github.com/nexa/pkg/ctx"
)

// NewDefaultRegistry registers collectors and placeholders based on the node_exporter README list.
// Implemented collectors are a minimal subset for the first iteration.
func NewDefaultRegistry(cctx *ctx.Ctx) *Registry {
	_ = cctx

	r := NewRegistry()

	// Linux "enabled by default" list (from node_exporter README excerpt).
	linuxEnabled := []string{
		"arp",
		"bcache",
		"bonding",
		"btrfs",
		"conntrack",
		"cpu",
		"cpufreq",
		"diskstats",
		"dmi",
		"edac",
		"entropy",
		"fibrechannel",
		"filefd",
		"filesystem",
		"hwmon",
		"infiniband",
		"ipvs",
		"kernel_hung",
		"loadavg",
		"mdadm",
		"meminfo",
		"netclass",
		"netdev",
		"netstat",
		"nfs",
		"nfsd",
		"nvme",
		"os",
		"powersupplyclass",
		"pressure",
		"rapl",
		"schedstat",
		"selinux",
		"sockstat",
		"softnet",
		"stat",
		"tapestats",
		"thermal_zone",
		"time",
		"timex",
		"udp_queues",
		"uname",
		"vmstat",
		"watchdog",
		"xfs",
		"zfs",
		// Note: some names above may not apply to all distros; we keep list for UX parity.
	}
	r.SetLinuxEnabledByDefault(linuxEnabled)

	// Implemented in this iteration.
	r.RegisterImplemented(NewCPUCollector())
	r.RegisterImplemented(NewMeminfoCollector())
	r.RegisterImplemented(NewFilesystemCollector())
	r.RegisterImplemented(NewDiskstatsCollector())
	r.RegisterImplemented(NewNetdevCollector())
	r.RegisterImplemented(NewLoadavgCollector())
	r.RegisterImplemented(NewOSCollector())
	r.RegisterImplemented(NewUnameCollector())
	r.RegisterImplemented(NewTimeCollector())
	r.RegisterImplemented(NewFilefdCollector())

	// Placeholders for the rest of Linux defaults.
	placeholderDesc := map[string]string{
		"arp":              "Exposes ARP statistics from /proc/net/arp.",
		"bcache":           "Exposes bcache statistics from /sys/fs/bcache/.",
		"bonding":          "Exposes bonding interface statistics.",
		"btrfs":            "Exposes btrfs statistics.",
		"conntrack":        "Shows conntrack statistics.",
		"cpufreq":          "Exposes CPU frequency statistics.",
		"dmi":              "Expose DMI info from /sys/class/dmi/id/.",
		"edac":             "Exposes ECC / EDAC statistics.",
		"entropy":          "Exposes available entropy.",
		"fibrechannel":     "Exposes fibre channel information and stats.",
		"filefd":           "Exposes file descriptor statistics from /proc/sys/fs/file-nr.",
		"hwmon":            "Expose hardware monitoring and sensor data.",
		"infiniband":       "Exposes InfiniBand stats.",
		"ipvs":             "Exposes IPVS status and stats.",
		"kernel_hung":      "Exposes hung task count.",
		"mdadm":            "Exposes mdadm /proc/mdstat stats.",
		"netclass":         "Exposes network interface info from sysfs.",
		"netstat":          "Exposes /proc/net/netstat stats.",
		"nfs":              "Exposes NFS client statistics.",
		"nfsd":             "Exposes NFS kernel server statistics.",
		"nvme":             "Exposes NVMe info from sysfs.",
		"powersupplyclass": "Exposes power supply class stats.",
		"pressure":         "Exposes PSI pressure stall information.",
		"rapl":             "Exposes RAPL powercap statistics.",
		"schedstat":        "Exposes /proc/schedstat statistics.",
		"selinux":          "Exposes SELinux statistics.",
		"sockstat":         "Exposes /proc/net/sockstat statistics.",
		"softnet":          "Exposes /proc/net/softnet_stat statistics.",
		"stat":             "Exposes /proc/stat statistics.",
		"tapestats":        "Exposes SCSI tape statistics.",
		"thermal_zone":     "Exposes thermal zone & cooling device stats.",
		"timex":            "Exposes adjtimex(2) statistics.",
		"udp_queues":       "Exposes UDP queue lengths.",
		"vmstat":           "Exposes /proc/vmstat statistics.",
		"watchdog":         "Exposes watchdog statistics.",
		"xfs":              "Exposes XFS runtime statistics.",
		"zfs":              "Exposes ZFS performance statistics.",
	}
	for _, name := range linuxEnabled {
		if r.Status(name).Implemented {
			continue
		}
		desc := placeholderDesc[name]
		if desc == "" {
			desc = "node_exporter compatible collector (not implemented)"
		}
		r.RegisterPlaceholder(name, desc)
	}

	// Ensure we list even implemented collectors with their desc.
	for _, name := range linuxEnabled {
		if _, ok := r.status[name]; !ok {
			r.RegisterPlaceholder(name, "node_exporter compatible collector (not implemented)")
		}
	}

	return r
}

