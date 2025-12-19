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

func SystemCollector(params map[string]string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// --------------------
	// CPU
	// --------------------
	if c, err := cpu.Percent(0, false); err == nil && len(c) > 0 {
		data["cpu_percent"] = c[0]
	}

	if cores, err := cpu.Counts(true); err == nil {
		data["cpu_cores"] = cores
	}

	if times, err := cpu.Times(false); err == nil && len(times) > 0 {
		t := times[0]
		data["cpu_user"] = t.User
		data["cpu_system"] = t.System
		data["cpu_idle"] = t.Idle
		data["cpu_iowait"] = t.Iowait
	}

	// Load average (very important for servers)
	if l, err := load.Avg(); err == nil {
		data["load_1"] = l.Load1
		data["load_5"] = l.Load5
		data["load_15"] = l.Load15
	}

	// --------------------
	// Memory
	// --------------------
	if v, err := mem.VirtualMemory(); err == nil {
		data["ram_used_percent"] = v.UsedPercent
		data["ram_total_mb"] = v.Total / 1024 / 1024
		data["ram_used_mb"] = v.Used / 1024 / 1024
		data["ram_free_mb"] = v.Free / 1024 / 1024
		data["ram_available_mb"] = v.Available / 1024 / 1024
	}

	if s, err := mem.SwapMemory(); err == nil {
		data["swap_used_percent"] = s.UsedPercent
		data["swap_used_mb"] = s.Used / 1024 / 1024
	}

	// --------------------
	// Disk (root usage)
	// --------------------
	if d, err := disk.Usage("/"); err == nil {
		data["disk_used_percent"] = d.UsedPercent
		data["disk_free_gb"] = d.Free / 1024 / 1024 / 1024
		data["disk_total_gb"] = d.Total / 1024 / 1024 / 1024
	}

	// Disk IO (aggregated, numeric only)
	if io, err := disk.IOCounters(); err == nil {
		var readBytes, writeBytes, readCount, writeCount uint64
		for _, stat := range io {
			readBytes += stat.ReadBytes
			writeBytes += stat.WriteBytes
			readCount += stat.ReadCount
			writeCount += stat.WriteCount
		}
		data["disk_read_bytes"] = readBytes
		data["disk_write_bytes"] = writeBytes
		data["disk_read_count"] = readCount
		data["disk_write_count"] = writeCount
	}

	// --------------------
	// Network (aggregated)
	// --------------------
	if n, err := net.IOCounters(false); err == nil && len(n) > 0 {
		data["net_bytes_sent"] = n[0].BytesSent
		data["net_bytes_recv"] = n[0].BytesRecv
		data["net_packets_sent"] = n[0].PacketsSent
		data["net_packets_recv"] = n[0].PacketsRecv
	}

	// --------------------
	// Host / OS
	// --------------------
	if h, err := host.Info(); err == nil {
		data["uptime_seconds"] = h.Uptime
		data["processes"] = h.Procs
	}

	// Process count (fallback / more accurate on some systems)
	if pids, err := process.Pids(); err == nil {
		data["process_count"] = len(pids)
	}

	return data, nil
}
