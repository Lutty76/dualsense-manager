package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func GetActualBatteryLevel() (int, error) {
	// On résout le wildcard (ex: /sys/class/power_supply/ps-controller-battery-*)
	// Note: /sys/class/power_supply est souvent plus simple que le chemin complet uhid
	matches, err := filepath.Glob("/sys/class/power_supply/ps-controller-battery-*/capacity")

	if err != nil || len(matches) == 0 {
		return 0, fmt.Errorf("Déconnecté")
	}

	// Lecture du fichier capacity
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return 0, fmt.Errorf("Erreur lecture")
	}

	// Conversion en entier
	levelStr := strings.TrimSpace(string(data))
	level, err := strconv.Atoi(levelStr)
	if err != nil {
		return 0, fmt.Errorf("Erreur format")
	}

	return level, nil
}

func GetChargingStatus() (string, error) {

	matches, err := filepath.Glob("/sys/class/power_supply/ps-controller-battery-*/status")

	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("Déconnecté")
	}

	// Lecture du fichier capacity
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return "", fmt.Errorf("Erreur lecture")
	}

	return string(data), nil

}
