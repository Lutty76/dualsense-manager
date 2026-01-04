// Package battery provides functions to read the battery status of a DualSense controller.
package battery

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"dualsense/internal/sysfs"
)

// Battery interface defines methods to read battery information.
type Battery interface {
	ActualBatteryLevel(jsPath string) (int, error)
	ChargingStatus(jsPath string) (string, error)
}

// ActualBatteryLevel reads the capacity file and returns the current battery percent.
func ActualBatteryLevel(jsPath string) (int, error) {
	basePath, err := batteryPath(jsPath)
	if err != nil {
		return 0, err
	}

	data, err := sysfs.FS.ReadFile(filepath.Join(basePath, "capacity"))
	if err != nil {
		return 0, err
	}

	level, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}

	return level, nil
}

// ChargingStatus returns the charging status string for the controller battery.
func ChargingStatus(jsPath string) (string, error) {
	basePath, err := batteryPath(jsPath)
	if err != nil {
		return "", err
	}

	data, err := sysfs.FS.ReadFile(filepath.Join(basePath, "status"))
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

func batteryPath(jsPath string) (string, error) {
	devicePath := fmt.Sprintf("/sys/class/input/%s/device/device/power_supply", filepath.Base(jsPath))
	matches, err := sysfs.FS.Glob(filepath.Join(devicePath, "ps-controller-battery-*"))
	if err != nil {
		return "", err
	}
	if len(matches) > 0 {
		return matches[0], nil
	}
	return "", fmt.Errorf("battery path not found")
}
