package collectors

import (
	"encoding/json"
	"net/http"
	"time"
)

// StarlinkCollector gathers telemetry from the local Starlink Dish.
// Target: http://192.168.100.1/api/get_status_data (Standard Dishy endpoint)
func StarlinkCollector(params map[string]string) ([]map[string]interface{}, error) {
	// 1. Configuration
	targetURL := "http://192.168.100.1/api/get_status_data"
	if url, ok := params["url"]; ok && url != "" {
		targetURL = url
	}

	client := &http.Client{
		Timeout: 3 * time.Second, // Dishy can be slow to respond during storms
	}

	// 2. Fetch Data
	resp, err := client.Get(targetURL)
	if err != nil {
		// If unreachable, return 0 for up status.
		return []map[string]interface{}{
			{
				"starlink_connected": 0,
			},
		}, nil
	}
	defer resp.Body.Close()

	// 3. Parse JSON
	// We use a struct to safely grab only the fields we care about
	var data struct {
		DeviceState struct {
			UptimeS int64 `json:"uptimeS"`
		} `json:"deviceState"`

		PopPingDropRate     float64 `json:"popPingDropRate"`     // Packet loss to POP
		PopPingLatencyMs    float64 `json:"popPingLatencyMs"`    // Latency
		DownlinkThroughput  float64 `json:"downlinkThroughputBps"`
		UplinkThroughput    float64 `json:"uplinkThroughputBps"`

		// Obstruction stats (Using generic map as structure varies by firmware)
		DishGetContext struct {
			ObstructionStats struct {
				FractionObstructed float64 `json:"fractionObstructed"`
				ValidS             float64 `json:"validS"`
			} `json:"obstructionStats"`
		} `json:"dishGetContext"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		// If JSON is malformed (firmware changed), we report error
		return nil, err
	}

	// 4. Return Normalized Data
	return []map[string]interface{}{
		{
			"starlink_connected":         1,
			"starlink_uptime_seconds":    data.DeviceState.UptimeS,
			"starlink_latency_ms":        data.PopPingLatencyMs,
			"starlink_packet_loss_pct":   data.PopPingDropRate * 100, // Convert 0.01 to 1%
			"starlink_downlink_mbps":     data.DownlinkThroughput / 1000000, // Bits -> Megabits
			"starlink_uplink_mbps":       data.UplinkThroughput / 1000000,
			"starlink_obstruction_pct":   data.DishGetContext.ObstructionStats.FractionObstructed * 100,
		},
	}, nil
}
