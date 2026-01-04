package leds

import (
	"dualsense/internal/sysfs"
	"fmt"
	"os"
	"strings"
	"testing"
)

type writeRec struct {
	path string
	data []byte
	perm os.FileMode
}

type fakeFS struct {
	files  map[string][]byte
	globs  map[string][]string
	writes []writeRec
}

func (f *fakeFS) ReadFile(path string) ([]byte, error) {
	if b, ok := f.files[path]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("not found: %s", path)
}
func (f *fakeFS) WriteFile(path string, data []byte, perm os.FileMode) error {
	f.writes = append(f.writes, writeRec{path: path, data: data, perm: perm})
	return nil
}
func (f *fakeFS) Glob(pattern string) ([]string, error) {
	if v, ok := f.globs[pattern]; ok {
		return v, nil
	}
	// fallback: try to synthesize a match from pattern like <base>/*:player-1
	if i := strings.LastIndex(pattern, ":"); i != -1 {
		led := pattern[i+1:]
		if j := strings.Index(pattern, "/*:"); j != -1 {
			base := pattern[:j]
			return []string{base + "/mock:" + led}, nil
		}
	}
	return nil, nil
}
func (f *fakeFS) Stat(path string) (os.FileInfo, error) {
	// pretend leds dirs exist when path contains "/leds"
	if strings.Contains(path, "/leds") {
		return nil, nil
	}
	return nil, fmt.Errorf("not found")
}

func (f *fakeFS) ResetWrites() {
	f.writes = nil
}

func TestSetPlayerNumber(t *testing.T) {
	old := sysfs.FS
	fake := &fakeFS{
		files:  map[string][]byte{},
		globs:  map[string][]string{},
		writes: nil,
	}

	sysfs.FS = fake
	defer func() { sysfs.FS = old }()
	jsPath := "/dev/input/js0"
	// prefill explicit globs so applyLed finds matching paths
	base := "/sys/class/input/js0/device/leds"
	fake.globs[base+"/*:player-1"] = []string{base + "/mock:player-1"}
	fake.globs[base+"/*:player-5"] = []string{base + "/mock:player-5"}
	fake.globs[base+"/*:player-2"] = []string{base + "/mock:player-2"}
	fake.globs[base+"/*:player-4"] = []string{base + "/mock:player-4"}
	fake.globs[base+"/*:player-3"] = []string{base + "/mock:player-3"}

	// Call function under test
	SetPlayerNumber(jsPath, 2)

	// expect 5 writes (player-1, player-5, player-2, player-4, player-3)
	if len(fake.writes) != 5 {
		t.Fatalf("expected 5 writes, got %d", len(fake.writes))
	}

	// expected values for id==2
	want := map[string]string{
		"player-1": "0",
		"player-5": "0",
		"player-2": "1",
		"player-4": "1",
		"player-3": "0",
	}

	for _, w := range fake.writes {
		var led string
		for k := range want {
			if strings.Contains(w.path, ":"+k+"/brightness") || strings.Contains(w.path, ":"+k) {
				led = k
				break
			}
		}
		if led == "" {
			t.Fatalf("write to unexpected path: %s", w.path)
		}
		if string(w.data) != want[led] {
			t.Fatalf("led %s: want %q got %q", led, want[led], string(w.data))
		}
	}
	fake.ResetWrites()

	// expected values for id==5
	SetPlayerNumber(jsPath, 5)

	// expect 10 writes (second call) (player-1, player-5, player-2, player-4, player-3)
	if len(fake.writes) != 5 {
		t.Fatalf("expected 5 writes, got %d", len(fake.writes))
	}

	// expected values for id==2
	want = map[string]string{
		"player-1": "1",
		"player-5": "1",
		"player-2": "1",
		"player-4": "1",
		"player-3": "1",
	}

	for _, w := range fake.writes {
		var led string
		for k := range want {
			if strings.Contains(w.path, ":"+k+"/brightness") || strings.Contains(w.path, ":"+k) {
				led = k
				break
			}
		}
		if led == "" {
			t.Fatalf("write to unexpected path: %s", w.path)
		}
		if string(w.data) != want[led] {
			t.Fatalf("led %s: want %q got %q", led, want[led], string(w.data))
		}
	}

}

func TestSetLightbarRGB(t *testing.T) {
	old := sysfs.FS
	fake := &fakeFS{
		files:  map[string][]byte{},
		globs:  map[string][]string{},
		writes: nil,
	}

	sysfs.FS = fake
	defer func() { sysfs.FS = old }()
	jsPath := "/dev/input/js0"
	// prefill explicit globs so applyLed finds matching paths
	base := "/sys/class/input/js0/device/leds"
	fake.globs[base+"/*:rgb:indicator"] = []string{base + "/mock:rgb:indicator"}
	// Call function under test
	SetLightbarRGB(jsPath, 100, 150, 200)

	// expect 1 write
	if len(fake.writes) != 2 {
		t.Fatalf("expected 2 write, got %d", len(fake.writes))
	}
	wantData := "100 150 200"
	if string(fake.writes[0].data) != wantData {
		t.Fatalf("expected data %q got %q", wantData, string(fake.writes[0].data))
	}
	wantPath := base + "/mock:rgb:indicator/multi_intensity"
	if fake.writes[0].path != wantPath {
		t.Fatalf("expected path %q got %q", wantPath, fake.writes[0].path)
	}

	wantData = "255"
	if string(fake.writes[1].data) != wantData {
		t.Fatalf("expected data %q got %q", wantData, string(fake.writes[1].data))
	}
	wantPath = base + "/mock:rgb:indicator/brightness"
	if fake.writes[1].path != wantPath {
		t.Fatalf("expected path %q got %q", wantPath, fake.writes[1].path)
	}
}

func TestSetBatteryColor(t *testing.T) {
	old := sysfs.FS
	fake := &fakeFS{
		files:  map[string][]byte{},
		globs:  map[string][]string{},
		writes: nil,
	}

	sysfs.FS = fake
	defer func() { sysfs.FS = old }()

	jsPath := "/dev/input/js0"
	// prefill explicit globs so applyLed finds matching paths
	base := "/sys/class/input/js0/device/leds"
	fake.globs[base+"/*:battery:indicator"] = []string{base + "/mock:battery:indicator"}
	// Call function under test
	SetBatteryColor(jsPath, 75)

	// expect 2 write
	if len(fake.writes) != 2 {
		t.Fatalf("expected 2 write, got %d", len(fake.writes))
	}
	wantData := "127 255 0"
	if string(fake.writes[0].data) != wantData {
		t.Fatalf("expected data %q got %q", wantData, string(fake.writes[0].data))
	}
	wantPath := base + "/mock:indicator/multi_intensity"
	if fake.writes[0].path != wantPath {
		t.Fatalf("expected path %q got %q", wantPath, fake.writes[0].path)
	}

	fake.ResetWrites()

	// Call function under test
	SetBatteryColor(jsPath, 25)
	// expect 2 write
	if len(fake.writes) != 2 {
		t.Fatalf("expected 2 write, got %d", len(fake.writes))
	}
	wantData = "255 127 0"
	if string(fake.writes[0].data) != wantData {
		t.Fatalf("expected data %q got %q", wantData, string(fake.writes[0].data))
	}
	wantPath = base + "/mock:indicator/multi_intensity"
	if fake.writes[0].path != wantPath {
		t.Fatalf("expected path %q got %q", wantPath, fake.writes[0].path)
	}

	fake.ResetWrites()

	// Call function under test
	SetBatteryColor(jsPath, 50)
	// expect 2 write
	if len(fake.writes) != 2 {
		t.Fatalf("expected 2 write, got %d", len(fake.writes))
	}
	wantData = "255 255 0"
	if string(fake.writes[0].data) != wantData {
		t.Fatalf("expected data %q got %q", wantData, string(fake.writes[0].data))
	}
	wantPath = base + "/mock:indicator/multi_intensity"
	if fake.writes[0].path != wantPath {
		t.Fatalf("expected path %q got %q", wantPath, fake.writes[0].path)
	}

	fake.ResetWrites()

	// Call function under test
	SetBatteryColor(jsPath, 10)
	// expect 2 write
	if len(fake.writes) != 2 {
		t.Fatalf("expected 2 write, got %d", len(fake.writes))
	}
	wantData = "255 51 0"
	if string(fake.writes[0].data) != wantData {
		t.Fatalf("expected data %q got %q", wantData, string(fake.writes[0].data))
	}
	wantPath = base + "/mock:indicator/multi_intensity"
	if fake.writes[0].path != wantPath {
		t.Fatalf("expected path %q got %q", wantPath, fake.writes[0].path)
	}
}

func TestSetBatteryLeds(t *testing.T) {
	old := sysfs.FS
	fake := &fakeFS{
		files:  map[string][]byte{},
		globs:  map[string][]string{},
		writes: nil,
	}

	sysfs.FS = fake
	defer func() { sysfs.FS = old }()
	jsPath := "/dev/input/js0"
	// prefill explicit globs so applyLed finds matching paths
	base := "/sys/class/input/js0/device/leds"
	fake.globs[base+"/*:player-1"] = []string{base + "/mock:player-1"}
	fake.globs[base+"/*:player-5"] = []string{base + "/mock:player-5"}
	fake.globs[base+"/*:player-2"] = []string{base + "/mock:player-2"}
	fake.globs[base+"/*:player-4"] = []string{base + "/mock:player-4"}
	fake.globs[base+"/*:player-3"] = []string{base + "/mock:player-3"}

	SetBatteryLeds(jsPath, 50)

	// expect 5 writes (player-1, player-5, player-2, player-4, player-3)
	if len(fake.writes) != 5 {
		t.Fatalf("expected 5 writes, got %d", len(fake.writes))
	}

	want := map[string]string{
		"player-1": "0",
		"player-5": "0",
		"player-2": "1",
		"player-4": "1",
		"player-3": "1",
	}

	for _, w := range fake.writes {
		var led string
		for k := range want {
			if strings.Contains(w.path, ":"+k+"/brightness") || strings.Contains(w.path, ":"+k) {
				led = k
				break
			}
		}
		if led == "" {
			t.Fatalf("write to unexpected path: %s", w.path)
		}
		if string(w.data) != want[led] {
			t.Fatalf("led %s: want %q got %q", led, want[led], string(w.data))
		}
	}

	fake.ResetWrites()

	SetBatteryLeds(jsPath, 100)
	// expect 5 writes (player-1, player-5, player-2, player-4, player-3)
	if len(fake.writes) != 5 {
		t.Fatalf("expected 5 writes, got %d", len(fake.writes))
	}

	want = map[string]string{
		"player-1": "1",
		"player-5": "1",
		"player-2": "1",
		"player-4": "1",
		"player-3": "1",
	}

	for _, w := range fake.writes {
		var led string
		for k := range want {
			if strings.Contains(w.path, ":"+k+"/brightness") || strings.Contains(w.path, ":"+k) {
				led = k
				break
			}
		}
		if led == "" {
			t.Fatalf("write to unexpected path: %s", w.path)
		}
		if string(w.data) != want[led] {
			t.Fatalf("led %s: want %q got %q", led, want[led], string(w.data))
		}
	}

}
