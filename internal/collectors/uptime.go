package collectors

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"time"
)

// UptimeCollector checks the availability of a target URL with detailed timing metrics.
// It returns a slice containing ONE map with numerical values only.
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

	// 3. Timing variables for detailed metrics
	var dnsStart, dnsDone, connectStart, connectDone, tlsStart, tlsDone, firstByteTime time.Time
	var redirectCount int64 = 0

	// 4. Create HTTP trace for detailed timing
	trace := &httptrace.ClientTrace{
		DNSStart: func(_ httptrace.DNSStartInfo) {
			dnsStart = time.Now()
		},
		DNSDone: func(_ httptrace.DNSDoneInfo) {
			dnsDone = time.Now()
		},
		ConnectStart: func(_, _ string) {
			connectStart = time.Now()
		},
		ConnectDone: func(_, _ string, _ error) {
			connectDone = time.Now()
		},
		TLSHandshakeStart: func() {
			tlsStart = time.Now()
		},
		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			tlsDone = time.Now()
		},
		GotFirstResponseByte: func() {
			firstByteTime = time.Now()
		},
	}

	// 5. Client Setup: Custom client with redirect counting
	client := &http.Client{
		Timeout: timeoutDuration,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
		    redirectCount = int64(len(via))
		    return http.ErrUseLastResponse 
		},
	}

	// 6. Create request with trace
	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	// 7. Execution
	start := time.Now()
	resp, err := client.Do(req)
	totalDuration := time.Since(start).Milliseconds()

	// 8. Initialize result with default values (all numeric)
	result := map[string]interface{}{
		"http_up":                  int64(0),
		"http_latency_ms":          totalDuration,
		"http_status_code":         int64(0),
		"http_dns_lookup_ms":       int64(0),
		"http_tcp_connect_ms":      int64(0),
		"http_tls_handshake_ms":    int64(0),
		"http_ttfb_ms":             int64(0),
		"http_download_ms":         int64(0),
		"http_response_size_bytes": int64(0),
		"http_redirect_count":      redirectCount,
		"http_cache_hit":           int64(0),
	}

	// 9. Process successful response
	if err == nil {
		defer resp.Body.Close()

		result["http_up"] = int64(1)
		result["http_status_code"] = int64(resp.StatusCode)

		// Calculate DNS lookup time
		if !dnsStart.IsZero() && !dnsDone.IsZero() {
			result["http_dns_lookup_ms"] = dnsDone.Sub(dnsStart).Milliseconds()
		}

		// Calculate TCP connect time
		if !connectStart.IsZero() && !connectDone.IsZero() {
			result["http_tcp_connect_ms"] = connectDone.Sub(connectStart).Milliseconds()
		}

		// Calculate TLS handshake time
		if !tlsStart.IsZero() && !tlsDone.IsZero() {
			result["http_tls_handshake_ms"] = tlsDone.Sub(tlsStart).Milliseconds()
		}

		// Calculate Time to First Byte (TTFB)
		if !firstByteTime.IsZero() {
			result["http_ttfb_ms"] = firstByteTime.Sub(start).Milliseconds()

			// Calculate download time (total - TTFB)
			downloadTime := totalDuration - firstByteTime.Sub(start).Milliseconds()
			if downloadTime > 0 {
				result["http_download_ms"] = downloadTime
			}
		}

		// Read response body to get accurate size
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			result["http_response_size_bytes"] = int64(len(bodyBytes))
		} else if resp.ContentLength > 0 {
			// Fallback to Content-Length header if body read fails
			result["http_response_size_bytes"] = resp.ContentLength
		}

		// Check cache status from headers (convert to numeric)
		cacheStatus := getCacheStatus(resp.Header)
		if cacheStatus == "HIT" || cacheStatus == "hit" || cacheStatus == "Hit" {
			result["http_cache_hit"] = int64(1)
		}
	}

	return []map[string]interface{}{result}, nil
}

// getCacheStatus extracts cache status from various CDN headers
func getCacheStatus(headers http.Header) string {
	// Check common cache headers from major CDNs
	cacheHeaders := []string{
		"x-vercel-cache",
		"cf-cache-status",
		"x-cache",
		"x-fastly-cache-status",
		"x-netlify-cache",
		"x-cache-status",
	}

	for _, header := range cacheHeaders {
		if val := headers.Get(header); val != "" {
			return val
		}
	}
	return ""
}
