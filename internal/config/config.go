package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IdleMinutes         int    `yaml:"idle_minutes"`
	BatteryAlert        int    `yaml:"battery_alert"`
	LastMAC             string `yaml:"last_mac"`
	Deadzone            int    `yaml:"deadzone"`
	LedPlayerPreference int    `yaml:"led_player"`
	LedRGBPreference    int    `yaml:"led_indicator"`
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
		IdleMinutes:         10,
		BatteryAlert:        15,
		Deadzone:            1000,
		LedPlayerPreference: 0,
		LedRGBPreference:    0, // Couleur par défaut
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
