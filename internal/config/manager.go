package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	ConfigFileName = "config.json"
)

var (
	// Public variables so main.go and status.go can read them
	GlobalDir     string
	GlobalLogPath string
)

// Initialize calculates the paths based on OS and creates directories.
// It is called once at the start of main().
func Initialize() {
	if runtime.GOOS == "windows" {
		// C:\ProgramData\HarborLighthouse
		GlobalDir = filepath.Join(os.Getenv("ProgramData"), "HarborLighthouse")
	} else {
		// /etc/harbor-lighthouse
		GlobalDir = "/etc/harbor-lighthouse"
	}

	// Try to create the system directory.
	// If this fails (e.g. running as non-root), we FALLBACK to local directory.
	err := os.MkdirAll(GlobalDir, 0755)
	if err != nil {
		// FALLBACK MODE (Dev/User)
		wd, _ := os.Getwd()
		GlobalDir = wd
	}

	// Set the Log Path based on the decided directory
	if runtime.GOOS == "windows" {
		GlobalLogPath = filepath.Join(GlobalDir, "harbor-lighthouse.log")
	} else {
		// On Linux, try /var/log first because it's standard
		varLog := "/var/log/harbor-lighthouse.log"
		f, err := os.OpenFile(varLog, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.Close()
			GlobalLogPath = varLog
		} else {
			// If we can't write to /var/log (no sudo), write to the config dir
			GlobalLogPath = filepath.Join(GlobalDir, "harbor-lighthouse.log")
		}
	}
}

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

// GetPath returns the absolute path to the config file using the GlobalDir
func GetPath() string {
	if GlobalDir == "" {
		Initialize()
	}
	return filepath.Join(GlobalDir, ConfigFileName)
}

func Load() (Config, error) {
	c := Config{
		AutoUpdate: true,
		Instances:  []Instance{},
	}

	path := GetPath()
	data, err := os.ReadFile(path)

	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return c, fmt.Errorf("failed to read config at %s: %w", path, err)
	}

	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("config parse error: %w", err)
	}

	return c, nil
}

func Save(c Config) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	path := GetPath()
	// Ensure the directory exists (it should, but safety first)
	_ = os.MkdirAll(filepath.Dir(path), 0755)

	return os.WriteFile(path, data, 0644)
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
