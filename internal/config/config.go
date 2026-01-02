package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IdleMinutes  int `yaml:"idle_minutes"`
	BatteryAlert int `yaml:"battery_alert"`
	// Per-controller configuration keyed by MAC address
	Controllers map[string]ControllerConfig `yaml:"controllers,omitempty"`
}

type ControllerConfig struct {
	Deadzone            int    `yaml:"deadzone,omitempty"`
	LedPlayerPreference int    `yaml:"led_player,omitempty"`
	LedRGBPreference    int    `yaml:"led_indicator,omitempty"`
	LedRGBStatic        string `yaml:"led_rgb_static,omitempty"`
}

// GetControllerConfig returns the configuration for a specific controller MAC.
// If a per-MAC config is not present or fields are zero, fall back to top-level defaults.
func (c *Config) GetControllerConfig(mac string) *ControllerConfig {
	// Start with reasonable defaults
	res := &ControllerConfig{
		Deadzone:            1500,
		LedPlayerPreference: 1,
		LedRGBPreference:    0,
	}

	if c.Controllers == nil {
		return res
	}

	if cc, ok := c.Controllers[mac]; ok {
		if cc.Deadzone != 0 {
			res.Deadzone = cc.Deadzone
		}
		if cc.LedPlayerPreference != 0 {
			res.LedPlayerPreference = cc.LedPlayerPreference
		}
		if cc.LedRGBPreference != 0 {
			res.LedRGBPreference = cc.LedRGBPreference
		}
		if cc.LedRGBStatic != "" {
			res.LedRGBStatic = cc.LedRGBStatic
		}
	}

	return res
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "dualsense-manager")
	_ = os.MkdirAll(path, 0755)
	return filepath.Join(path, "config.yaml")
}

func Save(conf *Config) error {
	path := getConfigPath()

	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func Load() *Config {
	path := getConfigPath()

	conf := &Config{
		IdleMinutes:  10,
		BatteryAlert: 15,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		_ = Save(conf)
		return conf
	}

	err = yaml.Unmarshal(data, conf)
	if err != nil {
		return conf
	}

	return conf
}
