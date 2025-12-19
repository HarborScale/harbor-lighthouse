package collectors

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ExecCollector runs a shell command and expects JSON output on STDOUT
func ExecCollector(params map[string]string) (map[string]interface{}, error) {
	commandStr, ok := params["command"]
	if !ok || commandStr == "" {
		return nil, fmt.Errorf("missing 'command' parameter")
	}

	var cmd *exec.Cmd

	// Handle Shell Execution based on OS
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", commandStr)
	} else {
		cmd = exec.Command("sh", "-c", commandStr)
	}

	output, err := cmd.Output()
	if err != nil {
		// Try to capture stderr for better debugging
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("command failed: %s (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("command failed: %v", err)
	}

	// Clean whitespace
	cleanOutput := strings.TrimSpace(string(output))
	if cleanOutput == "" {
		return nil, fmt.Errorf("command returned empty output")
	}

	// Parse JSON
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(cleanOutput), &result); err != nil {
		return nil, fmt.Errorf("output was not valid JSON: %v (output: %s)", err, cleanOutput)
	}

	return result, nil
}
