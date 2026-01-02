package service

import (
	"context"
	"dualsense/internal/config"
	"dualsense/internal/ui"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func ManageBatteryAndLEDs(ctx context.Context, app fyne.App, state *ui.AppState, path string, id int, debug bool) {
	var animCancelPlayer context.CancelFunc = func() {}
	var animCancelRGB context.CancelFunc = func() {}
	var animActivePlayer bool
	var animActiveRGB bool

	if debug {
		fmt.Println("Starting battery loop for controller at path:", path)
	}

	// Au cas où la goroutine principale s'arrête, on nettoie l'animation
	defer func() {
		animCancelPlayer()
		animCancelRGB()
		if debug {
			fmt.Println("Stopping battery loop for controller at path:", path)
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
			status, _ := ChargingStatus(path)

			if err != nil {
				state.StateText.Set("Dualsense not found")
				state.BatteryText.Set("Battery : --%")
				state.BatteryValue.Set(0)
				time.Sleep(5 * time.Second)
				continue
			}

			// Mise à jour de l'UI Fyne
			state.BatteryValue.Set(float64(level) / 100.0)
			state.BatteryText.Set(fmt.Sprintf("Battery : %d%%", level))
			state.StateText.Set("State : " + status)
			ledPref, _ := state.LedPlayerPreference.Get()
			rgbPref, _ := state.LedRGBPreference.Get()
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
					go RunRGBChargingAnimation(animCtxRGB, path, float64(level))
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
					hexColor, _ := state.LedRGBStaticColor.Get()
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
func StartActivityLoop(ctx context.Context, state *ui.AppState, activityChan chan time.Time, path string, debug bool) {

	if debug {
		fmt.Println("Starting activity loop for controller at path:", path)
	}

	lastActivityTime := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done(): // Si on annule le contexte, on arrête TOUT
			if debug {
				fmt.Println("Stopping activity loop for controller at path:", path)
			}
			return
		case t := <-activityChan:
			lastActivityTime = t
			state.LastActivityBinding.Set("In use")
		case <-ticker.C:
			status, _ := state.StateText.Get()

			if strings.Contains(status, "not found") || strings.Contains(status, "Recherche") {
				lastActivityTime = time.Now()
				state.LastActivityBinding.Set("Disconnected")
				continue
			}
			diff := time.Since(lastActivityTime)

			currentChoice, _ := state.SelectedDuration.Get()

			if currentChoice == "" {
				continue
			}

			if currentChoice == "Jamais" {
				state.LastActivityBinding.Set(fmt.Sprintf("Inactive : %s (Auto-off Disabled)", diff.Truncate(time.Second)))
				continue
			}
			if strings.Contains(status, "Charging") || strings.Contains(status, "Full") {
				state.LastActivityBinding.Set(fmt.Sprintf("Inactive : %s (disabled due to charging)", diff.Truncate(time.Second)))
				continue
			}

			parts := strings.Split(currentChoice, " ")
			minutes, err := strconv.Atoi(parts[0])
			if err != nil || minutes <= 0 {
				continue
			}

			limit := time.Duration(minutes) * time.Minute
			state.LastActivityBinding.Set(fmt.Sprintf("Inactive : %s / %s", diff.Truncate(time.Second), currentChoice))

			if diff > limit {
				fmt.Println("Auto disconnect !")
				// prefer cached MAC from UI state to avoid repeated sysfs reads
				macText, _ := state.MacText.Get()
				mac := strings.TrimSpace(strings.TrimPrefix(macText, "MAC :"))
				if mac != "" {
					err := DisconnectDualSenseNative(mac)
					if err != nil {
						fmt.Println("Fail D-Bus:", err)
					}
				}
			}
		}
	}

}

func StartControllerManager(myApp fyne.App, conf *config.Config, debug bool) *container.AppTabs {
	if debug {
		fmt.Println("StartControllerManager: debug mode enabled")
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
			foundPaths := FindAllDualSense()
			changed := false

			for id, path := range foundPaths {
				if _, exists := activeControllers[path]; !exists {

					if debug {
						fmt.Println("New DualSense detected at path:", path)
					}

					ctx, cancel := context.WithCancel(context.Background())
					mac := ControllerMAC(path)
					ctrlConf := conf.ControllerConfig(mac)
					newTab := ui.CreateNewControllerTab(path, conf, ctrlConf, mac, id+1)
					newTab.CancelFunc = cancel
					activeControllers[path] = newTab

					go MonitorJoystick(path, newTab.ActivityChan, newTab.State, debug)
					go ManageBatteryAndLEDs(ctx, myApp, newTab.State, path, id+1, debug)
					go StartActivityLoop(ctx, newTab.State, newTab.ActivityChan, path, debug)

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
	r, _ := strconv.ParseInt(hexStr[0:2], 16, 0)
	g, _ := strconv.ParseInt(hexStr[2:4], 16, 0)
	b, _ := strconv.ParseInt(hexStr[4:6], 16, 0)
	return int(r), int(g), int(b)
}
