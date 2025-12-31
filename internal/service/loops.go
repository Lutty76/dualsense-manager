package service

import (
	"context"
	"dualsense/internal/config"
	"dualsense/internal/ui"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func StartBatteryLoop(ctx context.Context, app fyne.App, state *ui.AppState, path string) {
	var animCancel context.CancelFunc = func() {}
	var animActive bool

	defer func() {
		animCancel()
		fmt.Println("Stopping battery loop for controller at path:", path)
	}()

	for {
		select {
		case <-ctx.Done():
			animCancel()
			return
		default:
			level, err := GetActualBatteryLevel(path)
			status, _ := GetChargingStatus(path)
			mac := GetControllerMAC(path)

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
			state.MacText.Set("MAC : " + mac)
			ledPref, _ := state.LedPlayerPreference.Get()
			rgbPref, _ := state.LedRGBPreference.Get()

			shouldAnimate := status == "Charging" &&
				(ledPref == ui.PlayerModeBattery || rgbPref == ui.RGBModeBattery)

			if shouldAnimate {
				if !animActive {
					var animCtx context.Context
					animCtx, animCancel = context.WithCancel(ctx)
					animActive = true

					if ledPref == ui.PlayerModeBattery {
						go RunChargingAnimation(animCtx, path)
					}
					if rgbPref == ui.RGBModeBattery {
						go RunRGBChargingAnimation(animCtx, path, float64(level))
					}
				}
			} else {
				if animActive {
					animCancel()
					animCancel = func() {}
					animActive = false
				}

				// --- Gestion LEDS PLAYER ---
				if ledPref == ui.PlayerModeBattery {
					SetBatteryLeds(path, float64(level))
				} else {
					SetPlayerNumber(path)
				}

				// --- Gestion RGB ---
				switch rgbPref {
				case ui.RGBModeBattery:
					SetBatteryColor(path, float64(level))
				case ui.RGBModeStatic:
					setLightbarRGB(path, 0, 50, 255)

				case ui.RGBModeOff:
					setLightbarRGB(path, 0, 0, 0)
				}
			}

			time.Sleep(1 * time.Second)
		}
	}
}
func StartActivityLoop(ctx context.Context, state *ui.AppState, activityChan chan time.Time, path string) {
	go func() {
		lastActivityTime := time.Now()
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Stopping activity loop for controller at path:", path)
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
					mac := GetControllerMAC(path)
					if mac != "" {
						err := DisconnectDualSenseNative(mac)
						if err != nil {
							fmt.Println("Fail D-Bus:", err)
						}
					}
				}
			}
		}
	}()
}

func StartControllerManager(myApp fyne.App, conf *config.Config) *container.AppTabs {
	emptyTab := container.NewTabItem("Info", widget.NewLabel("Waiting for DualSense..."))
	tabs := container.NewAppTabs(emptyTab)
	activeControllers := make(map[string]*ui.ControllerTab)

	refreshTabs := func() {
		var items []*container.TabItem
		if len(activeControllers) == 0 {
			items = append(items, emptyTab)
		} else {
			for path, ctrl := range activeControllers {
				tabName := fmt.Sprintf("DualSense %s", getShortMAC(path))
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

			for _, path := range foundPaths {
				if _, exists := activeControllers[path]; !exists {
					ctx, cancel := context.WithCancel(context.Background())
					newTab := ui.CreateNewControllerTab(path, conf)
					newTab.CancelFunc = cancel
					activeControllers[path] = newTab

					go MonitorJoystick(path, newTab.ActivityChan, newTab.State)
					go StartBatteryLoop(ctx, myApp, newTab.State, path)
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

func getShortMAC(path string) string {
	fullMAC := GetControllerMAC(path)
	if len(fullMAC) > 5 {
		return fullMAC[len(fullMAC)-5:]
	}
	return filepath.Base(path)
}
