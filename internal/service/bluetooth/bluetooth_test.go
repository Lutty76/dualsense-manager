package bluetooth

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"dualsense/internal/sysfs"

	"github.com/godbus/dbus/v5"
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

func TestControllerMAC_UniqAndAddress(t *testing.T) {
	old := sysfs.FS
	fake := fakeFS{files: map[string][]byte{}, globs: map[string][]string{}}

	fake.files[filepath.Join("/sys/class/input", "js0", "device", "uniq")] = []byte("aa:bb:cc:dd:ee:ff\n")
	sysfs.FS = fake
	defer func() { sysfs.FS = old }()

	mac := ControllerMAC("/dev/input/js0")
	if mac != "AA:BB:CC:DD:EE:FF" {
		t.Fatalf("unexpected mac: %s", mac)
	}

	// fallback to address when uniq missing
	fake2 := fakeFS{files: map[string][]byte{}, globs: map[string][]string{}}
	fake2.files[filepath.Join("/sys/class/input", "js1", "device", "address")] = []byte("11:22:33:44:55:66\n")
	sysfs.FS = fake2
	mac2 := ControllerMAC("/dev/input/js1")
	if mac2 != "11:22:33:44:55:66" {
		t.Fatalf("unexpected mac2: %s", mac2)
	}
}

func TestDisconnectDualSenseNative_Error(t *testing.T) {
	old := ConnectSystemBus
	// override to return an error (avoid calling system bus in tests)
	ConnectSystemBus = func(_ ...dbus.ConnOption) (*dbus.Conn, error) { return nil, fmt.Errorf("no bus available") }
	defer func() { ConnectSystemBus = old }()

	err := DisconnectDualSenseNative("AA:BB:CC:DD:EE:FF")
	if err == nil {
		t.Fatalf("expected error from DisconnectDualSenseNative when bus unavailable")
	}
}
