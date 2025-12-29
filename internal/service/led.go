package service

import (
	"fmt"
	"os"
	"time"
)

const ledPath = "/sys/class/input/js1/device/device/leds/input68:rgb:indicator/brightness"
const multiPath = "/sys/class/input/js1/device/device/leds/input68:rgb:indicator/multi_intensity"

func SetLED(value string) {
	err := os.WriteFile(ledPath, []byte(value), 0644)
	if err != nil {
		fmt.Println("Erreur écriture LED:", err)
	}
}
func SetLEDColor(multiPath string, r, g, b int) {
	// Format attendu par le noyau : "R G B"
	colorStr := fmt.Sprintf("%d %d %d", r, g, b)
	err := os.WriteFile(multiPath, []byte(colorStr), 0644)
	if err != nil {
		fmt.Println("Erreur écriture couleur:", err)
	}
}

func ChargeAnimation(multiPath string, stop chan bool) {
	for {
		select {
		case <-stop:
			return
		default:
			// Pulse du bleu (0 0 0 -> 0 0 255)
			for i := 0; i <= 255; i += 5 {
				SetLEDColor(multiPath, 0, 0, i)
				time.Sleep(30 * time.Millisecond)
			}
			for i := 255; i >= 0; i -= 5 {
				SetLEDColor(multiPath, 0, 0, i)
				time.Sleep(30 * time.Millisecond)
			}
		}
	}
}
