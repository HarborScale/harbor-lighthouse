package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const ConfigFileName = "lighthouse_config.json"

type Instance struct {
	Name       string            `json:"name"`        // ship_id
	HarborID   string            `json:"harbor_id"`   // e.g. "786"
	APIKey     string            `json:"api_key"`     // X-API-Key header
	Source     string            `json:"source"`      // linux, meshtastic
	HarborType string            `json:"harbor_type"` // general, gps
	Params     map[string]string `json:"params"`      // Extra config
}

type Config struct {
	AutoUpdate bool       `json:"auto_update"`
	Instances  []Instance `json:"instances"`
}

func Load() (Config, error) {
	c := Config{AutoUpdate: true, Instances: []Instance{}}
	data, err := os.ReadFile(ConfigFileName)
	if os.IsNotExist(err) { return c, nil }
	if err != nil { return c, err }
	err = json.Unmarshal(data, &c)
	return c, nil
}

func Save(c Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil { return err }
	return os.WriteFile(ConfigFileName, data, 0644)
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
