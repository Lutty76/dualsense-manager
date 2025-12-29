package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IdleMinutes  int    `yaml:"idle_minutes"`
	BatteryAlert int    `yaml:"battery_alert"`
	LastMAC      string `yaml:"last_mac"`
}

func getConfigPath() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "dualsense-manager")
	_ = os.MkdirAll(path, os.ModePerm)
	return filepath.Join(path, "config.yaml") // Extension .yaml
}

func Save(conf Config) error {
	path := getConfigPath()

	data, err := yaml.Marshal(&conf)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func Load() Config {
	path := getConfigPath()
	// Valeurs par défaut si le fichier n'existe pas
	conf := Config{
		IdleMinutes:  10,
		BatteryAlert: 15,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Si le fichier n'existe pas, on le crée avec les valeurs par défaut
		Save(conf)
		return conf
	}

	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return conf
	}
	return conf
}
