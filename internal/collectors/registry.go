package collectors

import (
	"fmt"
)

type Collector func(p map[string]string) ([]map[string]interface{}, error)

func Get(name string) (Collector, error) {
	switch name {
	case "linux", "system", "windows", "macos":
		return SystemCollector, nil
	case "exec", "script", "custom":
		return ExecCollector, nil
	case "uptime":
		return UptimeCollector, nil
	case "docker":
		return DockerCollector, nil
	case "ollama", "llm", "ai":
    return OllamaCollector, nil
  case "starlink", "dishy":
    return StarlinkCollector, nil
	default:
		return nil, fmt.Errorf("unknown source: %s", name)
	}
}
