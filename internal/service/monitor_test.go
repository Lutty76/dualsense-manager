package service

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"dualsense/internal/config"
	"dualsense/internal/ui"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/data/binding"
)

type fakeReadCloser struct {
	*bytes.Reader
}

func (f fakeReadCloser) Close() error { return nil }

func makeEvent(value int16, evType byte, index byte) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint32(b[0:4], uint32(time.Now().Unix()))
	binary.LittleEndian.PutUint16(b[4:6], uint16(value))
	b[6] = evType
	b[7] = index
	return b
}

func TestMonitorJoystickReceivesActivity(t *testing.T) {
	// Prepare two events: a button press and an axis movement above deadzone
	var buf bytes.Buffer
	buf.Write(makeEvent(1, 1, 3))    // button
	buf.Write(makeEvent(2000, 2, 5)) // axis

	// Override OpenJoystick to return our fake reader once, then error
	orig := OpenJoystick
	OpenJoystick = func(_ string) (io.ReadCloser, error) {
		return fakeReadCloser{bytes.NewReader(buf.Bytes())}, nil
	}
	defer func() { OpenJoystick = orig }()

	activityChan := make(chan time.Time, 10)

	// create a Fyne app so binding.Set doesn't panic when it uses fyne.Do
	fyApp := app.New()
	defer fyApp.Quit()
	ctrlConf := config.ControllerConfig{
		Deadzone: 5000,
	}

	state := &ui.ControllerState{
		DeadzoneValue: binding.NewFloat(),
	}
	// set low deadzone to ensure axis event counts
	_ = state.DeadzoneValue.Set(100)

	// Run monitor in background
	go MonitorJoystick("/dev/fakejs0", activityChan, &ctrlConf)

	// Wait for up to 1s to receive at least one activity
	timeout := time.After(1 * time.Second)
	received := 0
LOOP:
	for {
		select {
		case <-activityChan:
			received++
			if received >= 2 {
				break LOOP
			}
		case <-timeout:
			break LOOP
		}
	}

	if received == 0 {
		t.Fatalf("expected at least 1 activity, got %d", received)
	}
}
