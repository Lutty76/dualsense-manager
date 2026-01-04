package service

import (
	"context"
	"dualsense/internal/config"
	"dualsense/internal/ui"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	Debug bool
)

func ManageBatteryAndLEDs(ctx context.Context, app fyne.App, state *ui.AppState, path string, id int) {
	var animCancelPlayer context.CancelFunc = func() {}
	var animCancelRGB context.CancelFunc = func() {}
	var animActivePlayer bool
	var animActiveRGB bool
	batteryChan := make(chan float64)
	previousLevel := -1

	if Debug {
		log.Default().Println("Starting battery loop for controller at path:", path)
	}

	// Au cas où la goroutine principale s'arrête, on nettoie l'animation
	defer func() {
		animCancelPlayer()
		animCancelRGB()
		if Debug {
			log.Default().Println("Stopping battery loop for controller at path:", path)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			animCancelPlayer()
			animCancelRGB()
			return
		default:
			level, err := ActualBatteryLevel(path)
			if err != nil {
				err = state.StateText.Set("Dualsense not found")
				if err != nil {
					log.Default().Println("Error setting state text:", err)
				}
				err = state.BatteryText.Set("Battery : --%")
				if err != nil {
					log.Default().Println("Error setting battery text:", err)
				}
				err = state.BatteryValue.Set(0)
				if err != nil {
					log.Default().Println("Error setting battery value:", err)
				}
				time.Sleep(5 * time.Second)
				continue
			}
			status, err := ChargingStatus(path)
			if err != nil {
				continue
			}

			// Mise à jour de l'UI Fyne
			err = state.BatteryValue.Set(float64(level) / 100.0)
			if err != nil {
				log.Default().Println("Error setting battery value:", err)
			}
			err = state.BatteryText.Set(fmt.Sprintf("Battery : %d%%", level))
			if err != nil {
				log.Default().Println("Error setting battery text:", err)
			}
			err = state.StateText.Set("State : " + status)
			if err != nil {
				log.Default().Println("Error setting state text:", err)
			}
			if level != previousLevel {
				select {
				case batteryChan <- float64(level):
					previousLevel = level
				default:
				}
			}
			ledPref, err := state.LedPlayerPreference.Get()
			if err != nil {
				ledPref = ui.PlayerModeNumber
			}
			rgbPref, err := state.LedRGBPreference.Get()
			if err != nil {
				rgbPref = ui.RGBModeBattery
			}

			// 1. Gestion des animations
			if (ledPref == ui.PlayerModeBattery) && status == "Charging" {
				if !animActivePlayer {
					var animCtxPlayer context.Context
					animCtxPlayer, animCancelPlayer = context.WithCancel(ctx)
					animActivePlayer = true
					go RunChargingAnimation(animCtxPlayer, path)
				}

			} else {
				if animActivePlayer {
					animCancelPlayer()
					animCancelPlayer = func() {}
					animActivePlayer = false
				}

				if ledPref == ui.PlayerModeBattery {
					SetBatteryLeds(path, float64(level))
				} else {
					SetPlayerNumber(path, id) // Mode Numéro de manette
				}

			}

			if status == "Charging" && (rgbPref == ui.RGBModeBattery) {
				if !animActiveRGB {
					var animCtxRGB context.Context
					animCtxRGB, animCancelRGB = context.WithCancel(ctx)
					animActiveRGB = true
					go RunRGBChargingAnimation(animCtxRGB, path, batteryChan)
					if level != previousLevel {
						select {
						case batteryChan <- float64(level):
							previousLevel = level
						default:
						}
					}
				}
			} else {
				// 2. Pas d'animation : on arrête tout et on applique le fixe
				if animActiveRGB {
					animCancelRGB()
					animCancelRGB = func() {}
					animActiveRGB = false
				}

				switch rgbPref {
				case ui.RGBModeBattery:
					SetBatteryColor(path, float64(level))
				case ui.RGBModeStatic:
					hexColor, err := state.LedRGBStaticColor.Get()
					if err != nil {
						hexColor = "0000FF"
					}
					r, g, b := hexToRGB(hexColor)
					setLightbarRGB(path, r, g, b)

				case ui.RGBModeOff:
					setLightbarRGB(path, 0, 0, 0)
				}
			}

			time.Sleep(1 * time.Second)
		}
	}
}
func StartActivityLoop(ctx context.Context, state *ui.AppState, activityChan chan time.Time, path string) {

	if Debug {
		log.Default().Println("Starting activity loop for controller at path:", path)
	}

	lastActivityTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done(): // Si on annule le contexte, on arrête TOUT
			if Debug {
				log.Default().Println("Stopping activity loop for controller at path:", path)
			}
			return
		case t := <-activityChan:
			lastActivityTime = t
			err := state.LastActivityBinding.Set("In use")
			if err != nil {
				log.Default().Println("Error setting last activity binding:", err)
			}
		case <-ticker.C:
			status, err := state.StateText.Get()
			if err != nil {
				continue
			}

			if strings.Contains(status, "not found") || strings.Contains(status, "Recherche") {
				lastActivityTime = time.Now()
				err = state.LastActivityBinding.Set("Disconnected")
				if err != nil {
					log.Default().Println("Error setting last activity binding:", err)
				}
				continue
			}
			diff := time.Since(lastActivityTime)

			currentChoice, err := state.SelectedDuration.Get()
			if err != nil {
				continue
			}

			if currentChoice == "" {
				continue
			}

			if currentChoice == "Jamais" {
				err = state.LastActivityBinding.Set(fmt.Sprintf("Inactive : %s (Auto-off Disabled)", diff.Truncate(time.Second)))
				if err != nil {
					log.Default().Println("Error setting last activity binding:", err)
				}
				continue
			}
			if strings.Contains(status, "Charging") || strings.Contains(status, "Full") {
				err = state.LastActivityBinding.Set(fmt.Sprintf("Inactive : %s (disabled due to charging)", diff.Truncate(time.Second)))
				if err != nil {
					log.Default().Println("Error setting last activity binding:", err)
				}
				continue
			}

			parts := strings.Split(currentChoice, " ")
			minutes, err := strconv.Atoi(parts[0])
			if err != nil || minutes <= 0 {
				continue
			}

			limit := time.Duration(minutes) * time.Minute
			err = state.LastActivityBinding.Set(fmt.Sprintf("Inactive : %s / %s", diff.Truncate(time.Second), currentChoice))
			if err != nil {
				log.Default().Println("Error setting last activity binding:", err)
			}
			if diff > limit {
				log.Default().Println("Auto disconnect !")
				// prefer cached MAC from UI state to avoid repeated sysfs reads
				macText, err := state.MacText.Get()
				if err != nil {
					continue
				}
				mac := strings.TrimSpace(strings.TrimPrefix(macText, "MAC :"))
				if mac != "" {
					err := DisconnectDualSenseNative(mac)
					if err != nil {
						log.Default().Println("Fail D-Bus:", err)
					}
				}
			}
		}
	}

}

func StartControllerManager(myApp fyne.App, conf *config.Config) *container.AppTabs {
	if Debug {
		log.Default().Println("StartControllerManager: Debug mode enabled")
	}
	emptyTab := container.NewTabItem("Info", widget.NewLabel("Waiting for DualSense..."))
	tabs := container.NewAppTabs(emptyTab)
	activeControllers := make(map[string]*ui.ControllerTab)

	refreshTabs := func() {
		var items []*container.TabItem
		if len(activeControllers) == 0 {
			items = append(items, emptyTab)
		} else {
			for _, ctrl := range activeControllers {
				tabName := fmt.Sprintf("DualSense %s", ShortMAC(ctrl.MacAddress))
				items = append(items, container.NewTabItem(tabName, ctrl.Container))
			}
		}

		tabs.Items = items
		tabs.Refresh()
	}

	go func() {
		for {
			foundPaths, err := FindAllDualSense()
			if err != nil {
				log.Default().Println("Error finding DualSense controllers:", err)
				return
			}
			changed := false

			for id, path := range foundPaths {
				if _, exists := activeControllers[path]; !exists {

					if Debug {
						log.Default().Println("New DualSense detected at path:", path)
					}

					ctx, cancel := context.WithCancel(context.Background())
					mac := ControllerMAC(path)
					ctrlConf := conf.ControllerConfig(mac)
					newTab := ui.CreateNewControllerTab(path, conf, ctrlConf, mac, id+1)
					newTab.CancelFunc = cancel
					activeControllers[path] = newTab

					go MonitorJoystick(path, newTab.ActivityChan, newTab.State)
					go ManageBatteryAndLEDs(ctx, myApp, newTab.State, path, id+1)
					go StartActivityLoop(ctx, newTab.State, newTab.ActivityChan, path)

					changed = true
				}
			}

			for path, ctrl := range activeControllers {
				if !pathExists(path) {
					ctrl.CancelFunc()
					delete(activeControllers, path)
					changed = true
				}
			}

			if changed {
				fyne.Do(func() {
					refreshTabs()
				})
			}

			time.Sleep(2 * time.Second)
		}
	}()

	return tabs
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func ShortMAC(fullMAC string) string {
	if len(fullMAC) > 5 {
		return fullMAC[len(fullMAC)-5:]
	}
	return fullMAC
}

func hexToRGB(hexStr string) (int, int, int) {
	hexStr = strings.TrimPrefix(hexStr, "#")
	if len(hexStr) != 6 {
		return 0, 0, 0
	}
	r, err := strconv.ParseInt(hexStr[0:2], 16, 0)
	if err != nil {
		return 0, 0, 0
	}
	g, err := strconv.ParseInt(hexStr[2:4], 16, 0)
	if err != nil {
		return 0, 0, 0
	}
	b, err := strconv.ParseInt(hexStr[4:6], 16, 0)
	if err != nil {
		return 0, 0, 0
	}
	return int(r), int(g), int(b)
}
