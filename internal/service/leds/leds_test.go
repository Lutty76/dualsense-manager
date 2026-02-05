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

	tests := []struct {
		id   int
		want map[string]string
	}{
		{
			id: 1,
			want: map[string]string{
				"player-1": "0",
				"player-5": "0",
				"player-2": "0",
				"player-4": "0",
				"player-3": "1",
			},
		},
		{
			id: 2,
			want: map[string]string{
				"player-1": "0",
				"player-5": "0",
				"player-2": "1",
				"player-4": "1",
				"player-3": "0",
			},
		},
		{
			id: 3,
			want: map[string]string{
				"player-1": "0",
				"player-5": "0",
				"player-2": "1",
				"player-4": "1",
				"player-3": "1",
			},
		},
		{
			id: 4,
			want: map[string]string{
				"player-1": "1",
				"player-5": "1",
				"player-2": "1",
				"player-4": "1",
				"player-3": "0",
			},
		},
		{
			id: 5,
			want: map[string]string{
				"player-1": "1",
				"player-5": "1",
				"player-2": "1",
				"player-4": "1",
				"player-3": "1",
			},
		},
	}

	for _, tt := range tests {
		fake.ResetWrites()
		SetPlayerNumber(jsPath, tt.id)

		// expect 5 writes (player-1, player-5, player-2, player-4, player-3)
		if len(fake.writes) != 5 {
			t.Fatalf("expected 5 writes, got %d", len(fake.writes))
		}

		for _, w := range fake.writes {
			var led string
			for k := range tt.want {
				if strings.Contains(w.path, ":"+k+"/brightness") || strings.Contains(w.path, ":"+k) {
					led = k
					break
				}
			}
			if led == "" {
				t.Fatalf("write to unexpected path: %s", w.path)
			}
			if string(w.data) != tt.want[led] {
				t.Fatalf("led %s: want %q for player number %d got %q", led, tt.want[led], tt.id, string(w.data))
			}
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

	tests := []struct {
		batteryLevel int
		want         string
	}{
		{batteryLevel: 100, want: "0 255 0"},
		{batteryLevel: 75, want: "127 255 0"},
		{batteryLevel: 50, want: "255 255 0"},
		{batteryLevel: 25, want: "255 127 0"},
		{batteryLevel: 10, want: "255 51 0"},
	}

	for _, tt := range tests {
		fake.ResetWrites()
		SetBatteryColor(jsPath, float64(tt.batteryLevel))

		// expect 2 write
		if len(fake.writes) != 2 {
			t.Fatalf("expected 2 write, got %d", len(fake.writes))
		}

		if string(fake.writes[0].data) != tt.want {
			t.Fatalf("expected data %q got %q", tt.want, string(fake.writes[0].data))
		}
		wantPath := base + "/mock:indicator/multi_intensity"
		if fake.writes[0].path != wantPath {
			t.Fatalf("expected path %q got %q", wantPath, fake.writes[0].path)
		}
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

	tests := []struct {
		batteryLevel int
		want         map[string]string
		blinking     bool
	}{
		{
			batteryLevel: 100,
			want: map[string]string{
				"player-1": "1",
				"player-5": "1",
				"player-2": "1",
				"player-4": "1",
				"player-3": "1",
			},
			blinking: false,
		},
		{
			batteryLevel: 50,
			want: map[string]string{
				"player-1": "0",
				"player-5": "0",
				"player-2": "1",
				"player-4": "1",
				"player-3": "1",
			},
			blinking: false,
		},
		{
			batteryLevel: 25,
			want: map[string]string{
				"player-1": "0",
				"player-5": "0",
				"player-2": "0",
				"player-4": "0",
				"player-3": "1",
			},
			blinking: false,
		},
		{
			batteryLevel: 15,
			want: map[string]string{
				"player-1": "0",
				"player-5": "0",
				"player-2": "1",
				"player-4": "1",
				"player-3": "0",
			},
			blinking: true,
		},
		{
			batteryLevel: 5,
			want: map[string]string{
				"player-1": "0",
				"player-5": "0",
				"player-2": "0",
				"player-4": "0",
				"player-3": "1",
			},
			blinking: true,
		},
	}

	for _, tt := range tests {
		fake.ResetWrites()
		SetBatteryLeds(jsPath, float64(tt.batteryLevel))
		// expect 5 writes (player-1, player-5, player-2, player-4, player-3)
		if len(fake.writes) != 5 {
			t.Fatalf("expected 5 writes, got %d", len(fake.writes))
		}

		for _, w := range fake.writes {
			var led string
			for k := range tt.want {
				if strings.Contains(w.path, ":"+k+"/brightness") || strings.Contains(w.path, ":"+k) {
					if tt.blinking && (k == "player-3" || k == "player-2" || k == "player-4") {
						// for blinking case, player-3 can be either "0" or "1"
						if string(w.data) != "0" && string(w.data) != "1" {
							t.Fatalf("led %s: want blinking value for %d%% got %q", k, tt.batteryLevel, string(w.data))
						}
						led = k
						break
					}
					led = k
					break
				}
			}
			if led == "" {
				t.Fatalf("write to unexpected path: %s", w.path)
			}
			if string(w.data) != tt.want[led] && !(tt.blinking && (led == "player-3" || led == "player-2" || led == "player-4")) {
				t.Fatalf("led %s: want %q for %d%% got %q", led, tt.want[led], tt.batteryLevel, string(w.data))
			}
		}
	}

}
