package collectors

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func ExecCollector(params map[string]string) ([]map[string]interface{}, error) {
	commandStr, ok := params["command"]
	if !ok {
		commandStr = params["script_path"]
	}
	if commandStr == "" {
		return nil, fmt.Errorf("missing 'command' or 'script_path' param")
	}

	// 1. TIMEOUT CONFIGURATION
	// Default to 10 seconds if not provided
	timeoutDuration := 10 * time.Second
	if val, ok := params["timeout_ms"]; ok {
		if ms, err := strconv.Atoi(val); err == nil && ms > 0 {
			timeoutDuration = time.Duration(ms) * time.Millisecond
		}
	}

	// Parse command (basic split)
	parts := strings.Fields(commandStr)
	head := parts[0]
	args := parts[1:]

	// 2. APPLY TIMEOUT CONTEXT
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	cmd := exec.CommandContext(ctx, head, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		line := scanner.Text()

		// Try to parse line as a single JSON object
		var row map[string]interface{}
		if err := json.Unmarshal([]byte(line), &row); err == nil {
			results = append(results, row)
			continue
		}

		// Try to parse line as a JSON Array (Mother Collector Mode)
		var rows []map[string]interface{}
		if err := json.Unmarshal([]byte(line), &rows); err == nil {
			results = append(results, rows...)
			continue
		}
	}

	// Wait for process to finish
	if err := cmd.Wait(); err != nil {
		// If the context deadline exceeded, it means we timed out
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("command timed out after %v", timeoutDuration)
		}
		// Other runtime errors (e.g. exit code 1) are just logged,
		// but we still return whatever data we managed to grab.
	}

	if len(results) == 0 {
		return []map[string]interface{}{}, nil
	}

	return results, nil
}
