package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ConfigFileName = "lighthouse_config.json"

type Instance struct {
	Name         string            `json:"name"`
	HarborID     string            `json:"harbor_id"`
	APIKey       string            `json:"api_key"`
	Source       string            `json:"source"`
	HarborType   string            `json:"harbor_type"`
	Params       map[string]string `json:"params"`
	Interval     int               `json:"interval"`
	MaxBatchSize int               `json:"max_batch_size"`
	Endpoint     string            `json:"endpoint,omitempty"`
}

type Config struct {
	AutoUpdate bool       `json:"auto_update"`
	Instances  []Instance `json:"instances"`
}

// getConfigPath returns the absolute path to the config file
// located next to the running binary.
func getConfigPath() string {
	exePath, err := os.Executable()
	if err != nil {
		// Fallback to current directory if we can't find self (rare)
		return ConfigFileName
	}
	return filepath.Join(filepath.Dir(exePath), ConfigFileName)
}

func Load() (Config, error) {
	c := Config{AutoUpdate: true, Instances: []Instance{}}

	path := getConfigPath()
	data, err := os.ReadFile(path)

	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, err
	}

	err = json.Unmarshal(data, &c)
	return c, nil
}

func Save(c Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	path := getConfigPath()
	return os.WriteFile(path, data, 0o644)
}

func (c *Config) Add(n Instance) error {
	for _, i := range c.Instances {
		if i.Name == n.Name {
			return fmt.Errorf("instance '%s' already exists", n.Name)
		}
	}
	c.Instances = append(c.Instances, n)
	return nil
}

func (c *Config) Remove(n string) bool {
	for i, v := range c.Instances {
		if v.Name == n {
			c.Instances = append(c.Instances[:i], c.Instances[i+1:]...)
			return true
		}
	}
	return false
}
