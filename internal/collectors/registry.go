package collectors

import (
	"fmt"
)

type Collector func(p map[string]string) (map[string]interface{}, error)

func Get(name string) (Collector, error) {
	switch name {
	case "linux", "system", "windows", "macos":
		return SystemCollector, nil
	case "meshtastic":
		return MeshtasticCollector, nil
	case "exec", "script", "custom":
		return ExecCollector, nil
	default:
		return nil, fmt.Errorf("unknown source: %s", name)
	}
}
