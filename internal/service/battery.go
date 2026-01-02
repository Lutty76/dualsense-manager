package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const powerPath = "/sys/class/power_supply/ps-controller-battery-*/"

func ActualBatteryLevel(jsPath string) (int, error) {
	basePath := batteryPath(jsPath)
	if basePath == "" {
		return 0, fmt.Errorf("Disconnected")
	}

	data, err := os.ReadFile(filepath.Join(basePath, "capacity"))
	if err != nil {
		return 0, fmt.Errorf("Error read")
	}

	level, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("Error format")
	}

	return level, nil
}

func ChargingStatus(jsPath string) (string, error) {
	basePath := batteryPath(jsPath)
	if basePath == "" {
		return "", fmt.Errorf("Disconnected")
	}

	data, err := os.ReadFile(filepath.Join(basePath, "status"))
	if err != nil {
		return "", fmt.Errorf("Error read")
	}

	return strings.TrimSpace(string(data)), nil
}

func batteryPath(jsPath string) string {

	devicePath := fmt.Sprintf("/sys/class/input/%s/device/device/power_supply", filepath.Base(jsPath))

	// On cherche un dossier qui commence par ps-controller-battery-
	matches, _ := filepath.Glob(filepath.Join(devicePath, "ps-controller-battery-*"))
	if len(matches) > 0 {
		return matches[0]
	}
	return ""
}
