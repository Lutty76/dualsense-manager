package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func FindAllDualSense() ([]string, error) {
	var found []string
	matches, err := filepath.Glob("/dev/input/js*")
	if err != nil {
		return found, err
	}
	for _, path := range matches {
		namePath := fmt.Sprintf("/sys/class/input/%s/device/name", filepath.Base(path))
		nameBytes, err := os.ReadFile(namePath)
		if err != nil {
			log.Default().Printf("Unable to read %s :%s\n", namePath, err)
			continue
		}
		name := strings.ToLower(string(nameBytes))

		if strings.Contains(name, "sony") || strings.Contains(name, "dualsense") {
			if strings.Contains(name, "motion sensors") { //We ignore gyroscope endpoint
				continue
			}
			found = append(found, path)
		}
	}
	return found, nil
}
