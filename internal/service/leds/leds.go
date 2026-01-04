// Package leds provides functions to control the LEDs of a DualSense controller.
package leds

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"time"

	"dualsense/internal/sysfs"
)

// Leds interface defines methods to control DualSense LEDs.
type Leds interface {
	RunChargingAnimation(ctx context.Context, jsPath string)
	RunRGBChargingAnimation(ctx context.Context, hidPath string, batteryLevel chan float64)
	SetBatteryColor(jsPath string, percent float64)
	SetBatteryLeds(jsPath string, percent float64)
	SetPlayerNumber(jsPath string, id int)
	SetLightbarRGB(jsPath string, r, g, b int)
}

// RunChargingAnimation animates player LEDs to indicate charging progress.
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

// RunRGBChargingAnimation animates the RGB lightbar while charging.
func RunRGBChargingAnimation(ctx context.Context, hidPath string, batteryLevel chan float64) {
	ticker := time.NewTicker(25 * time.Millisecond) // Animation fluide
	defer ticker.Stop()

	percent := <-batteryLevel
	// On calcule la couleur cible une fois au début
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
			baseR, baseG = 255, 255
			if percent > 50 {
				baseR = int(255 * (100 - percent) / 50)
			} else {
				baseG = int(255 * percent / 50)
			}
		case <-ticker.C:
			brightness := 0.6 + 0.4*math.Sin(theta)

			r := int(float64(baseR) * brightness)
			g := int(float64(baseG) * brightness)

			SetLightbarRGB(hidPath, r, g, 0)

			theta += 0.1
			if theta > 2*math.Pi {
				theta = 0
			}
		}
	}
}

// SetBatteryColor sets the RGB lightbar color based on battery percent.
func SetBatteryColor(jsPath string, percent float64) {
	var r, g int
	if percent > 50 {
		r = int(255 * (100 - percent) / 50)
		g = 255
	} else {
		r = 255
		g = int(255 * percent / 50)
	}
	SetLightbarRGB(jsPath, r, g, 0)
}

// SetBatteryLeds updates the player LEDs to represent battery level.
func SetBatteryLeds(jsPath string, percent float64) {
	ledBase := getLedPath(jsPath)

	var p15 string
	var p24 string
	var p3 string
	blink := false

	if percent >= 75 {
		p15, p24, p3 = "1", "1", "1"
	} else if percent >= 50 {
		p15, p24, p3 = "0", "1", "1"
	} else if percent >= 20 {
		p15, p24, p3 = "0", "0", "1"
	} else if percent >= 10 {
		p15, p24, p3 = "0", "1", "0"
		blink = true
	} else {
		p15, p24, p3 = "0", "0", "1"
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

// SetPlayerNumber updates the player LEDs to indicate controller number.
func SetPlayerNumber(jsPath string, id int) {
	ledBase := getLedPath(jsPath)

	playerNum := id
	p15, p24, p3 := "0", "0", "0"

	switch playerNum {
	case 1:
		p3 = "1"
	case 2:
		p24 = "1"
	case 3:
		p24, p3 = "1", "1"
	case 4:
		p15, p24 = "1", "1"
	default:
		p15, p24, p3 = "1", "1", "1"
	}

	applyLed(ledBase, "player-1", p15)
	applyLed(ledBase, "player-5", p15)
	applyLed(ledBase, "player-2", p24)
	applyLed(ledBase, "player-4", p24)
	applyLed(ledBase, "player-3", p3)
}

func applyLed(basePath, ledName, value string) {
	matches, err := sysfs.FS.Glob(fmt.Sprintf("%s/*:%s", basePath, ledName))
	if err != nil {
		return
	}
	if len(matches) == 0 {
		return
	}

	path := matches[0]

	_ = sysfs.FS.WriteFile(fmt.Sprintf("%s/brightness", path), []byte(value), 0644)
}

func getLedPath(jsPath string) string {
	base := fmt.Sprintf("/sys/class/input/%s/device", filepath.Base(jsPath))

	path := fmt.Sprintf("%s/leds", base)
	if _, err := sysfs.FS.Stat(path); err == nil {
		return path
	}

	path = fmt.Sprintf("%s/device/leds", base)
	if _, err := sysfs.FS.Stat(path); err == nil {
		return path
	}

	return ""
}

// SetLightbarRGB sets the multi_intensity and brightness of the RGB lightbar.
func SetLightbarRGB(jsPath string, r, g, b int) {
	basePath := getLedPath(jsPath)
	matches, err := sysfs.FS.Glob(fmt.Sprintf("%s/*:rgb:indicator", basePath))
	if err != nil {
		return
	}
	if len(matches) == 0 {
		return
	}

	path := matches[0]

	colorStr := fmt.Sprintf("%d %d %d", r, g, b)
	_ = sysfs.FS.WriteFile(fmt.Sprintf("%s/multi_intensity", path), []byte(colorStr), 0644)
	_ = sysfs.FS.WriteFile(fmt.Sprintf("%s/brightness", path), []byte("255"), 0644)
}
