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

// ControllerConfig returns the configuration for a specific controller MAC.
// If a per-MAC config is not present or fields are zero, fall back to top-level defaults.
func (c *Config) ControllerConfig(mac string) *ControllerConfig {
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

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".config", "dualsense-manager")
	_ = os.MkdirAll(path, 0755)
	return filepath.Join(path, "config.yaml"), nil
}

func Save(conf *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	conf := &Config{
		IdleMinutes:  10,
		BatteryAlert: 15,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		err = Save(conf)
		if err != nil {
			return nil, err
		}
		return conf, nil
	}

	err = yaml.Unmarshal(data, conf)
	if err != nil {
		return nil, nil
	}

	return conf, nil
}
