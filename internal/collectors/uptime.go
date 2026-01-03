package collectors

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// UptimeCollector checks the availability of a target URL.
// It returns a slice containing ONE map with numerical values.
func UptimeCollector(params map[string]string) ([]map[string]interface{}, error) {
	// 1. Validation: Ensure target exists
	target, ok := params["target_url"]
	if !ok || target == "" {
		return nil, fmt.Errorf("missing target_url param")
	}

	// 2. Configuration: Parse timeout (default to 10 seconds if not provided)
	timeoutDuration := 10 * time.Second
	if val, ok := params["timeout_ms"]; ok {
		if ms, err := strconv.Atoi(val); err == nil && ms > 0 {
			timeoutDuration = time.Duration(ms) * time.Millisecond
		}
	}

	// 3. Client Setup: Use a custom client to enforce timeouts
	client := &http.Client{
		Timeout: timeoutDuration,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 4. Execution
	start := time.Now()
	resp, err := client.Get(target)

	// Calculate duration immediately
	duration := time.Since(start).Milliseconds()

	// 5. Data Normalization (Strictly Numerical)
	var up int64 = 0
	var statusCode int64 = 0

	if err == nil {
		defer resp.Body.Close()
		up = 1
		statusCode = int64(resp.StatusCode)
	}

	return []map[string]interface{}{
		{
			"http_up":          up,
			"http_latency_ms":  duration,
			"http_status_code": statusCode,
		},
	}, nil
}
