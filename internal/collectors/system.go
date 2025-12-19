package collectors

import (
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

func SystemCollector(params map[string]string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	// 1. Memory
	v, err := mem.VirtualMemory()
	if err == nil {
		data["ram_used_percent"] = v.UsedPercent
		data["ram_total_mb"] = v.Total / 1024 / 1024
		data["ram_free_mb"] = v.Free / 1024 / 1024
	}

	// 2. CPU
	c, err := cpu.Percent(0, false)
	if err == nil && len(c) > 0 {
		data["cpu_percent"] = c[0]
	}

	// 3. Disk (Root)
	d, err := disk.Usage("/")
	if err == nil {
		data["disk_used_percent"] = d.UsedPercent
		data["disk_free_gb"] = d.Free / 1024 / 1024 / 1024
	}

	// 4. Host Info
	h, err := host.Info()
	if err == nil {
		data["uptime_hours"] = h.Uptime / 3600
	}



	return data, nil
}
