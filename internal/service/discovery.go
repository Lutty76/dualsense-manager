package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FindAllDualSense() []string {
	var found []string
	matches, _ := filepath.Glob("/dev/input/js*")

	for _, path := range matches {
		namePath := fmt.Sprintf("/sys/class/input/%s/device/name", filepath.Base(path))
		nameBytes, _ := os.ReadFile(namePath)
		name := strings.ToLower(string(nameBytes))

		if strings.Contains(name, "sony") || strings.Contains(name, "dualsense") {
			if strings.Contains(name, "motion sensors") { //We ignore gyroscope endpoint
				continue
			}
			found = append(found, path)
		}
	}
	return found
}
