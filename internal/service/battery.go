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
	basePath, err := batteryPath(jsPath)
	if err != nil {
		return 0, err
	}

	data, err := os.ReadFile(filepath.Join(basePath, "capacity"))
	if err != nil {
		return 0, err
	}

	level, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return level, nil
}

func ChargingStatus(jsPath string) (string, error) {
	basePath, err := batteryPath(jsPath)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(filepath.Join(basePath, "status"))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func batteryPath(jsPath string) (string, error) {

	devicePath := fmt.Sprintf("/sys/class/input/%s/device/device/power_supply", filepath.Base(jsPath))

	// On cherche un dossier qui commence par ps-controller-battery-
	matches, err := filepath.Glob(filepath.Join(devicePath, "ps-controller-battery-*"))
	if err != nil {
		return "", err
	}
	if len(matches) > 0 {
		return matches[0], nil
	}
	return "", fmt.Errorf("Battery path not found")
}
