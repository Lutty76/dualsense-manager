package service

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

func RunChargingAnimation(ctx context.Context, jsPath string) {
	ticker := time.NewTicker(800 * time.Millisecond) // Vitesse de l'animation
	defer ticker.Stop()

	step := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var p15, p24, p3 string

			switch step % 3 {
			case 0: // --X--
				p15, p24, p3 = "0", "0", "1"
			case 1: // -XXX-
				p15, p24, p3 = "0", "1", "1"
			case 2: // XXXXX
				p15, p24, p3 = "1", "1", "1"
			}

			ledBase := getLedPath(jsPath)
			applyLed(ledBase, "player-1", p15)
			applyLed(ledBase, "player-5", p15)
			applyLed(ledBase, "player-2", p24)
			applyLed(ledBase, "player-4", p24)
			applyLed(ledBase, "player-3", p3)

			step++
		}
	}

}

func RunRGBChargingAnimation(ctx context.Context, jsPath string, batteryLevel chan float64, debug bool) {
	ticker := time.NewTicker(25 * time.Millisecond) // Animation fluide
	defer ticker.Stop()

	percent := <-batteryLevel
	if debug {
		fmt.Println("Starting RGB charging animation with battery level:", percent)
	}
	baseR, baseG := 255, 255
	if percent > 50 {
		baseR = int(255 * (100 - percent) / 50)
	} else {
		baseG = int(255 * percent / 50)
	}

	theta := 0.0
	for {
		select {
		case <-ctx.Done():
			return
		case percent := <-batteryLevel:
			if debug {
				fmt.Println("updating RGB charging animation with battery level:", percent)
			}
			baseR, baseG = 255, 255
			if percent > 50 {
				baseR = int(255 * (100 - percent) / 50)
			} else {
				baseG = int(255 * percent / 50)
			}
		case <-ticker.C:
			// Utilisation d'un sinus pour une variation fluide (0.2 à 1.0)
			brightness := 0.6 + 0.4*math.Sin(theta)

			r := int(float64(baseR) * brightness)
			g := int(float64(baseG) * brightness)

			setLightbarRGB(jsPath, r, g, 0)

			theta += 0.1
			if theta > 2*math.Pi {
				theta = 0
			}
		}
	}
}

func SetBatteryColor(jsPath string, percent float64) {
	var r, g int
	if percent > 50 {
		r = int(255 * (100 - percent) / 50)
		g = 255
	} else {
		r = 255
		g = int(255 * percent / 50)
	}
	setLightbarRGB(jsPath, r, g, 0)
}

func SetBatteryLeds(jsPath string, percent float64) {
	ledBase := getLedPath(jsPath)

	p15 := "0" // Off
	p24 := "0"
	p3 := "0"
	blink := false

	if percent >= 75 {
		p15, p24, p3 = "1", "1", "1" // XXXXX
	} else if percent >= 50 {
		p15, p24, p3 = "0", "1", "1" // -XXX-
	} else if percent >= 20 {
		p15, p24, p3 = "0", "0", "1" // --X--
	} else if percent >= 10 {
		p15, p24, p3 = "0", "1", "0" // -X-X-
		blink = true
	} else {
		p15, p24, p3 = "0", "0", "1" // --X--
		blink = true
	}

	applyLed(ledBase, "player-1", p15)
	applyLed(ledBase, "player-5", p15)
	if blink {
		if time.Now().Unix()%2 == 0 {
			applyLed(ledBase, "player-2", p24)
			applyLed(ledBase, "player-4", p24)

		} else {
			applyLed(ledBase, "player-2", "0")
			applyLed(ledBase, "player-4", "0")
		}
	} else {
		applyLed(ledBase, "player-2", p24)
		applyLed(ledBase, "player-4", p24)
	}
	if blink {
		if time.Now().Unix()%2 == 0 {
			applyLed(ledBase, "player-3", p3)

		} else {
			applyLed(ledBase, "player-3", "0")
		}
	} else {
		applyLed(ledBase, "player-3", p3)
	}
}

func SetPlayerNumber(jsPath string, id int) {
	ledBase := getLedPath(jsPath)

	playerNum := id
	// Reset initial
	p15, p24, p3 := "0", "0", "0"

	switch playerNum {
	case 1:
		p3 = "1" // Un point au centre
	case 2:
		p24 = "1" // Deux points
	case 3:
		p24, p3 = "1", "1" // Trois points
	case 4:
		p15, p24 = "1", "1" // Quatre points
	default:
		p15, p24, p3 = "1", "1", "1" // Cinq points
	}

	applyLed(ledBase, "player-1", p15)
	applyLed(ledBase, "player-5", p15)
	applyLed(ledBase, "player-2", p24)
	applyLed(ledBase, "player-4", p24)
	applyLed(ledBase, "player-3", p3)
}

func applyLed(basePath, ledName, value string) {

	matches, err := filepath.Glob(filepath.Join(basePath, "*:"+ledName))
	if err != nil {
		fmt.Println("Error finding LED path:", err)
		return
	}
	if len(matches) == 0 {
		return
	}

	path := matches[0]

	err = os.WriteFile(filepath.Join(path, "brightness"), []byte(value), 0644)
	if err != nil {
		fmt.Println("Error writing LED value:", err)
	}

}

func getLedPath(jsPath string) string {
	base := fmt.Sprintf("/sys/class/input/%s/device", filepath.Base(jsPath))

	path := filepath.Join(base, "leds")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	path = filepath.Join(base, "device/leds")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}

func setLightbarRGB(jsPath string, r, g, b int) {
	basePath := getLedPath(jsPath)
	matches, err := filepath.Glob(filepath.Join(basePath, "*:rgb:indicator"))
	if err != nil {
		fmt.Println("Error finding RGB path:", err)
		return
	}
	if len(matches) == 0 {
		return
	}

	path := matches[0]
	colorStr := fmt.Sprintf("%d %d %d", r, g, b)
	err = os.WriteFile(filepath.Join(path, "multi_intensity"), []byte(colorStr), 0644)
	if err != nil {
		fmt.Println("Error writing RGB value:", err)
	}

	err = os.WriteFile(filepath.Join(path, "brightness"), []byte("255"), 0644)
	if err != nil {
		fmt.Println("Error writing brightness value:", err)
	}
}
