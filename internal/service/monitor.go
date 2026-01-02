package service

import (
	"dualsense/internal/ui"
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

func MonitorJoystick(path string, activityChan chan time.Time, state *ui.AppState, debug bool) {
	if debug {
		fmt.Println("Starting joystick monitor for controller at path:", path)
	}

	for {
		f, err := os.Open(path)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue

		}
		defer f.Close()

		// Structure d'un événement joystick Linux (8 octets)
		// Time (4) | Value (2) | Type (1) | Index (1)
		buffer := make([]byte, 8)

		for {
			_, err := f.Read(buffer)
			if err != nil {
				if debug {
					fmt.Println("Stopping joystick monitor for controller at path:", path)
				}
				f.Close()
				break
			}
			val, err := state.DeadzoneValue.Get()
			if err != nil {
				val = 1500
			}
			deadzone := int16(val)

			evType := buffer[6]
			evValue := int16(binary.LittleEndian.Uint16(buffer[4:6]))

			isReal := false
			switch evType {
			case 1: // Bouton pressé
				isReal = true
				if debug {
					fmt.Printf("Button %d event detected with value: %d\n", buffer[7], evValue)
				}
			case 2: // Axe bougé
				if evValue > deadzone || evValue < -deadzone {
					isReal = true
					if debug {
						fmt.Printf("Axis %d event detected with value: %d\n", buffer[7], evValue)
					}
				} else {
					if debug {
						fmt.Printf("Axis %d event ignored due to deadzone with value: %d\n", buffer[7], evValue)
					}
				}
			}

			if isReal {
				activityChan <- time.Now()
			}
		}
	}
}
