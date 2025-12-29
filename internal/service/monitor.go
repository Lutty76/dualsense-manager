package service

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"
)

func MonitorJoystick(path string, activityChan chan time.Time, stopChan chan bool) {
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
		deadzone := int16(2500) // Pour ignorer le stick drift

		for {
			_, err := f.Read(buffer)
			if err != nil {
				fmt.Println("Manette déconnectée, arrêt de la lecture.")
				f.Close()
				break
			}

			evType := buffer[6]
			evValue := int16(binary.LittleEndian.Uint16(buffer[4:6]))

			isReal := false
			if evType == 1 { // Bouton pressé
				isReal = true
			} else if evType == 2 { // Axe bougé
				// On vérifie si le mouvement dépasse la zone morte
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
