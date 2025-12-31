package service

import (
	"dualsense/internal/ui"
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

func MonitorJoystick(path string, activityChan chan time.Time, state *ui.AppState) {
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
				fmt.Println("Joypad disconnected, stopping ...")
				f.Close()
				break
			}
			val, _ := state.DeadzoneValue.Get()
			deadzone := int16(val)

			evType := buffer[6]
			evValue := int16(binary.LittleEndian.Uint16(buffer[4:6]))

			isReal := false
			switch evType {
			case 1: // Bouton pressé
				isReal = true
			case 2: // Axe bougé
				if evValue > deadzone || evValue < -deadzone {
					isReal = true
				}
			}

			if isReal {
				activityChan <- time.Now()
			}
		}
	}
}
