package collectors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// MeshtasticCollector tries two methods:
// 1. HTTP JSON API (if 'ip' param is set)
// 2. Python CLI (if no IP, assumes USB/Serial via 'meshtastic' command)
func MeshtasticCollector(params map[string]string) (map[string]interface{}, error) {
	ip, hasIP := params["ip"]

	if hasIP {
		return fetchMeshtasticHTTP(ip)
	}

	// Fallback to CLI Wrapper
	// Requires: pip install meshtastic
	return fetchMeshtasticCLI()
}

func fetchMeshtasticHTTP(ip string) (map[string]interface{}, error) {
	url := fmt.Sprintf("http://%s/json/report", ip)
	client := http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(url)
	if err != nil { return nil, err }
	defer resp.Body.Close()

	// This assumes the device returns a JSON report
	// Actual Meshtastic JSON structure varies by version,
	// capturing raw generic map for safety.
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func fetchMeshtasticCLI() (map[string]interface{}, error) {
	// Runs: meshtastic --info --test
	// NOTE: Parsing raw text output is fragile.
	// In a real prod environment, you'd want the python script to output JSON.
	// For now, we check basic connection.

	cmd := exec.Command("meshtastic", "--info")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("CLI failed (is meshtastic installed?): %v", err)
	}

	// Mocking parser - in production, you would write a Regex parser here
	// to extract battery, lat, long from the text output.
	// Since that's complex, we return raw status for now.

	return map[string]interface{}{
		"status": "connected",
		"raw_output_length": len(output),
		"method": "usb_cli",
	}, nil
}
