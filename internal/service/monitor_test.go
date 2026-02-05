package service

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"dualsense/internal/config"
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

	tests := []struct {
		name     string
		events   []byte
		deadzone int
		want     int
	}{
		{
			name: "button press",
			events: func() []byte {
				var buf bytes.Buffer
				buf.Write(makeEvent(1, 1, 3)) // button event
				return buf.Bytes()
			}(),
			deadzone: 0,
			want:     1,
		},
		{
			name: "axis movement above deadzone",
			events: func() []byte {
				var buf bytes.Buffer
				buf.Write(makeEvent(2000, 2, 5)) // axis event above deadzone
				return buf.Bytes()
			}(),
			deadzone: 5000,
			want:     0,
		},
		{
			name: "axis movement below deadzone",
			events: func() []byte {
				var buf bytes.Buffer
				buf.Write(makeEvent(3000, 2, 5)) // axis event above deadzone
				return buf.Bytes()
			}(),
			deadzone: 2000,
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override OpenJoystick to return our fake reader once, then error
			orig := OpenJoystick
			OpenJoystick = func(_ string) (io.ReadCloser, error) {
				return fakeReadCloser{bytes.NewReader(tt.events)}, nil
			}
			defer func() { OpenJoystick = orig }()

			activityChan := make(chan time.Time, 10)

			ctrlConf := config.ControllerConfig{
				Deadzone: tt.deadzone,
			}

			// Run monitor in background
			go MonitorJoystick("/dev/fakejs0", activityChan, &ctrlConf)

			// Wait for up to 1s to receive expected number of activities
			timeout := time.After(1 * time.Second)
			received := 0
		LOOP:
			for {
				select {
				case <-activityChan:
					received++
					if received >= tt.want {
						break LOOP
					}
				case <-timeout:
					break LOOP
				}
			}

			if received != tt.want {
				t.Fatalf("expected %d activities, got %d", tt.want, received)
			}
		})
	}
}
