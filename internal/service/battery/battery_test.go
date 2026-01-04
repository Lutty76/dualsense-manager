package battery

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"dualsense/internal/sysfs"
)

type fakeFS struct {
	files map[string][]byte
	globs map[string][]string
}

func (f fakeFS) ReadFile(path string) ([]byte, error) {
	if b, ok := f.files[path]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("not found: %s", path)
}
func (f fakeFS) WriteFile(_ string, _ []byte, _ os.FileMode) error {
	return fmt.Errorf("not implemented")
}
func (f fakeFS) Glob(pattern string) ([]string, error) { return f.globs[pattern], nil }
func (f fakeFS) Stat(_ string) (os.FileInfo, error)    { return nil, fmt.Errorf("not implemented") }

func TestActualBatteryLevelAndChargingStatus(t *testing.T) {
	old := sysfs.FS
	fake := fakeFS{
		files: map[string][]byte{},
		globs: map[string][]string{},
	}

	jsPath := "/dev/input/js0"
	devicePath := filepath.Join("/sys/class/input", filepath.Base(jsPath), "device/device/power_supply")
	globPattern := filepath.Join(devicePath, "ps-controller-battery-*")
	baseMatch := "/sys/class/power_supply/ps-controller-battery-0"

	fake.globs[globPattern] = []string{baseMatch}
	fake.files[filepath.Join(baseMatch, "capacity")] = []byte("85\n")
	fake.files[filepath.Join(baseMatch, "status")] = []byte("Charging\n")

	sysfs.FS = fake
	defer func() { sysfs.FS = old }()

	level, err := ActualBatteryLevel(jsPath)
	if err != nil {
		t.Fatalf("ActualBatteryLevel error: %v", err)
	}
	if level != 85 {
		t.Fatalf("expected 85 got %d", level)
	}

	status, err := ChargingStatus(jsPath)
	if err != nil {
		t.Fatalf("ChargingStatus error: %v", err)
	}
	if status != "Charging" {
		t.Fatalf("expected Charging got %q", status)
	}
}
