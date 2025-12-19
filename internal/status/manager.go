package status

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

const StatusFile = "lighthouse_status.json"

type InstanceStatus struct {
	LastContact int64  `json:"last_contact"`
	LastError   string `json:"last_error"`
	Healthy     bool   `json:"healthy"`
}

type StatusDB struct {
	Instances map[string]InstanceStatus `json:"instances"`
	mu        sync.Mutex
}

var db = StatusDB{Instances: make(map[string]InstanceStatus)}

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

	// Save to disk asynchronously or immediately? Immediately for simplicity.
	data, _ := json.Marshal(db)
	os.WriteFile(StatusFile, data, 0644)
}

func Load() map[string]InstanceStatus {
	data, err := os.ReadFile(StatusFile)
	if err != nil { return make(map[string]InstanceStatus) }
	var temp StatusDB
	json.Unmarshal(data, &temp)
	return temp.Instances
}
