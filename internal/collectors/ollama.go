package collectors

import (
	"encoding/json"
	"net/http"
	"time"
)

// OllamaCollector checks the status of a local LLM server.
// Default URL: http://localhost:11434
func OllamaCollector(params map[string]string) ([]map[string]interface{}, error) {
	// 1. Configuration
	targetURL := "http://localhost:11434"
	if url, ok := params["url"]; ok && url != "" {
		targetURL = url
	}

	client := &http.Client{
		Timeout: 2 * time.Second, // Fast timeout, local AI should be instant
	}

	// 2. Check: Is Ollama Up? (GET /api/version)
	// We check version first to ensure the service is alive.
	start := time.Now()
	resp, err := client.Get(targetURL + "/api/version")
	if err != nil {
		// Service is down, return specific "down" metrics rather than erroring out completely
		return []map[string]interface{}{
			{
				"ollama_up": 0,
			},
		}, nil
	}
	defer resp.Body.Close()
	latency := time.Since(start).Milliseconds()

	// 3. Check: What is running? (GET /api/ps)
	// This endpoint tells us about loaded models and VRAM usage.
	// Note: Available in Ollama v0.1.30+.
	req, _ := http.NewRequest("GET", targetURL+"/api/ps", nil)
	respPs, err := client.Do(req)

	var loadedModels int64 = 0
	var totalVramBytes int64 = 0
	var totalSizeBytes int64 = 0

	if err == nil && respPs.StatusCode == 200 {
		defer respPs.Body.Close()
		var psData struct {
			Models []struct {
				Name      string `json:"name"`
				Size      int64  `json:"size"`       // Model size on disk
				SizeVRAM  int64  `json:"size_vram"`  // Critical: VRAM usage
			} `json:"models"`
		}

		if json.NewDecoder(respPs.Body).Decode(&psData) == nil {
			loadedModels = int64(len(psData.Models))
			for _, m := range psData.Models {
				totalVramBytes += m.SizeVRAM
				totalSizeBytes += m.Size
			}
		}
	}

	// 4. Return Data
	return []map[string]interface{}{
		{
			"ollama_up":              1,
			"ollama_latency_ms":      latency,
			"ollama_models_loaded":   loadedModels,
			"ollama_vram_usage_mb":   totalVramBytes / 1024 / 1024, // Convert to MB for readability
			"ollama_model_size_mb":   totalSizeBytes / 1024 / 1024,
		},
	}, nil
}
