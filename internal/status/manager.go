package status

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/harborscale/harbor-lighthouse/internal/config"
)

const StatusFileName = "status.json"

type InstanceStatus struct {
	LastContact int64  `json:"last_contact"`
	LastError   string `json:"last_error"`
	Healthy     bool   `json:"healthy"`
}

type StatusDB struct {
	Instances map[string]InstanceStatus `json:"instances"`
	mu        sync.RWMutex
}

var db = StatusDB{Instances: make(map[string]InstanceStatus)}

// getPath relies strictly on config.GlobalDir
func getPath() string {
	// Ensure config is initialized (safety check)
	if config.GlobalDir == "" {
		config.Initialize()
	}
	return filepath.Join(config.GlobalDir, StatusFileName)
}

func Update(name string, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	s := InstanceStatus{
		LastContact: time.Now().Unix(),
		Healthy:     err == nil,
	}
	if err != nil {
		s.LastError = err.Error()
	}

	db.Instances[name] = s
	saveToDisk()
}

func Load() map[string]InstanceStatus {
	db.mu.RLock()
	defer db.mu.RUnlock()

	path := getPath()
	data, err := os.ReadFile(path)

	if os.IsNotExist(err) {
		return make(map[string]InstanceStatus)
	}

	var temp StatusDB
	if err := json.Unmarshal(data, &temp); err != nil {
		return make(map[string]InstanceStatus)
	}

	return temp.Instances
}

func saveToDisk() {
	temp := struct {
		Instances map[string]InstanceStatus `json:"instances"`
	}{
		Instances: db.Instances,
	}

	data, err := json.MarshalIndent(temp, "", "  ")
	if err != nil {
		return
	}

	path := getPath()
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, data, 0644)
}
