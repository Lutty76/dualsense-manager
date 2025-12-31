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
			// On calcule le palier en fonction de l'étape
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

func RunRGBChargingAnimation(ctx context.Context, jsPath string, percent float64) {
	ticker := time.NewTicker(25 * time.Millisecond) // Animation fluide
	defer ticker.Stop()

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
		// De Orange (50%) à Vert (100%)
		r = int(255 * (100 - percent) / 50)
		g = 255
	} else {
		// De Rouge (0%) à Orange (50%)
		r = 255
		g = int(255 * percent / 50)
	}
	setLightbarRGB(jsPath, r, g, 0)
}

func SetBatteryLeds(jsPath string, percent float64) {
	ledBase := getLedPath(jsPath)

	// Définition des paliers
	// player-1 et 5 : Extrémités
	// player-2 et 4 : Intermédiaires
	// player-3      : Centre
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
		blink = true                 // Clignotant
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

func SetPlayerNumber(jsPath string) {
	ledBase := getLedPath(jsPath)
	jsName := filepath.Base(jsPath) // "js0", "js1"...

	// On extrait le numéro (0, 1, 2...) et on ajoute 1
	var num int
	fmt.Sscanf(jsName, "js%d", &num)
	playerNum := num

	// Reset initial
	p15, p24, p3 := "0", "0", "0"

	// Logique selon le numéro (en tenant compte des limitations hardware)
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
	// On cherche le dossier qui contient le nom de la LED (ex: input92:white:player-1)

	matches, _ := filepath.Glob(filepath.Join(basePath, "*:"+ledName))
	if len(matches) == 0 {
		return
	}

	path := matches[0]

	_ = os.WriteFile(filepath.Join(path, "brightness"), []byte(value), 0644)

}

func getLedPath(jsPath string) string {
	base := fmt.Sprintf("/sys/class/input/%s/device", filepath.Base(jsPath))

	// Test du premier niveau
	path := filepath.Join(base, "leds")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Test du second niveau (ce que tu as actuellement)
	path = filepath.Join(base, "device/leds")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}

func setLightbarRGB(jsPath string, r, g, b int) {
	// On cherche le dossier rgb:indicator via le jsPath
	basePath := getLedPath(jsPath)
	matches, _ := filepath.Glob(filepath.Join(basePath, "*:rgb:indicator"))
	if len(matches) == 0 {
		return
	}

	path := matches[0]
	// Le format standard pour multi_intensity est "R G B" (0-255)
	colorStr := fmt.Sprintf("%d %d %d", r, g, b)
	_ = os.WriteFile(filepath.Join(path, "multi_intensity"), []byte(colorStr), 0644)

	// On s'assure que la luminosité globale est au max
	_ = os.WriteFile(filepath.Join(path, "brightness"), []byte("255"), 0644)
}
