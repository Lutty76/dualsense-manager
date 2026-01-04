package discovery

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

func TestFindAllDualSense(t *testing.T) {
	old := sysfs.FS
	fake := fakeFS{
		files: map[string][]byte{},
		globs: map[string][]string{},
	}

	// create two js devices, one real DualSense and one motion sensors device
	fake.globs["/dev/input/js*"] = []string{"/dev/input/js0", "/dev/input/js1"}
	// name files under /sys/class/input/<dev>/device/name
	fake.files[filepath.Join("/sys/class/input", "js0", "device", "name")] = []byte("Sony Interactive Entertainment Wireless Controller\n")
	fake.files[filepath.Join("/sys/class/input", "js1", "device", "name")] = []byte("DualSense motion sensors\n")

	sysfs.FS = fake
	defer func() { sysfs.FS = old }()

	found, err := FindAllDualSense()
	if err != nil {
		t.Fatalf("FindAllDualSense error: %v", err)
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 controller found, got %d: %v", len(found), found)
	}
	if found[0] != "/dev/input/js0" {
		t.Fatalf("unexpected path: %v", found)
	}
}
