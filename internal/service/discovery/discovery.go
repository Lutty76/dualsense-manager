// Package discovery provides functions to discover DualSense controllers connected to the system.
package discovery

import (
	"fmt"
	"path/filepath"
	"strings"

	"dualsense/internal/sysfs"
)

// Discovery interface defines methods for discovering DualSense controllers.
type Discovery interface {
	FindAllDualSense() ([]string, error)
}

// FindAllDualSense discovers DualSense joystick device nodes under /dev/input.
func FindAllDualSense() ([]string, error) {
	var found []string
	matches, err := sysfs.FS.Glob("/dev/input/js*")
	if err != nil {
		return found, err
	}
	for _, path := range matches {
		namePath := fmt.Sprintf("/sys/class/input/%s/device/name", filepath.Base(path))
		nameBytes, err := sysfs.FS.ReadFile(namePath)
		if err != nil {
			continue
		}
		name := strings.ToLower(string(nameBytes))

		if strings.Contains(name, "sony") || strings.Contains(name, "dualsense") {
			if strings.Contains(name, "motion sensors") {
				continue
			}
			found = append(found, path)
		}
	}
	return found, nil
}
