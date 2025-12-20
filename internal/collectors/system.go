package collectors

import (
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

func SystemCollector(params map[string]string) ([]map[string]interface{}, error) {
	snapshot := make(map[string]interface{})

	// --- CPU ---
	if c, err := cpu.Percent(0, false); err == nil && len(c) > 0 {
		snapshot["cpu_percent"] = c[0]
	}
	if cores, err := cpu.Counts(true); err == nil {
		snapshot["cpu_cores"] = cores
	}
	if times, err := cpu.Times(false); err == nil && len(times) > 0 {
		t := times[0]
		snapshot["cpu_user"] = t.User
		snapshot["cpu_system"] = t.System
		snapshot["cpu_idle"] = t.Idle
		snapshot["cpu_iowait"] = t.Iowait
	}

	// --- Load Avg ---
	if l, err := load.Avg(); err == nil {
		snapshot["load_1"] = l.Load1
		snapshot["load_5"] = l.Load5
		snapshot["load_15"] = l.Load15
	}

	// --- RAM ---
	if v, err := mem.VirtualMemory(); err == nil {
		snapshot["ram_used_percent"] = v.UsedPercent
		snapshot["ram_total_mb"] = v.Total / 1024 / 1024
		snapshot["ram_used_mb"] = v.Used / 1024 / 1024
		snapshot["ram_free_mb"] = v.Free / 1024 / 1024
	}

	// --- Swap ---
	if s, err := mem.SwapMemory(); err == nil {
		snapshot["swap_used_percent"] = s.UsedPercent
		snapshot["swap_used_mb"] = s.Used / 1024 / 1024
	}

	// --- Disk (Root) ---
	if d, err := disk.Usage("/"); err == nil {
		snapshot["disk_used_percent"] = d.UsedPercent
		snapshot["disk_free_gb"] = d.Free / 1024 / 1024 / 1024
		snapshot["disk_total_gb"] = d.Total / 1024 / 1024 / 1024
	}

	// --- Disk IO ---
	if io, err := disk.IOCounters(); err == nil {
		var readBytes, writeBytes uint64
		for _, stat := range io {
			readBytes += stat.ReadBytes
			writeBytes += stat.WriteBytes
		}
		snapshot["disk_read_bytes"] = readBytes
		snapshot["disk_write_bytes"] = writeBytes
	}

	// --- Net ---
	if n, err := net.IOCounters(false); err == nil && len(n) > 0 {
		snapshot["net_bytes_sent"] = n[0].BytesSent
		snapshot["net_bytes_recv"] = n[0].BytesRecv
	}

	// --- Host ---
	if h, err := host.Info(); err == nil {
		snapshot["uptime_seconds"] = h.Uptime
		snapshot["processes"] = h.Procs
	}
	if pids, err := process.Pids(); err == nil {
		snapshot["process_count"] = len(pids)
	}

	// ✅ CORRECT RETURN: Wrap the single snapshot in a slice
	return []map[string]interface{}{snapshot}, nil
}
