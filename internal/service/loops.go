// Package service contains the main service loops for managing DualSense controllers.
package service

import (
	"context"
	"dualsense/internal/config"
	"dualsense/internal/service/battery"
	"dualsense/internal/service/bluetooth"
	"dualsense/internal/service/discovery"
	"dualsense/internal/service/leds"
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
	// Debug enables debug logging within the service package.
	Debug bool
)

// ManageBatteryAndLEDs handles battery monitoring and LED management for a controller.
func ManageBatteryAndLEDs(ctx context.Context, _ fyne.App, state *ui.ControllerState, path string, id int) {
	var animCancelPlayer context.CancelFunc = func() {}
	var animCancelRGB context.CancelFunc = func() {}
	var animActivePlayer bool
	var animActiveRGB bool
	batteryChan := make(chan float64)
	previousLevel := -1

	if Debug {
		log.Default().Println("Starting battery loop for controller at path:", path)
	}

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
			level, err := battery.ActualBatteryLevel(path)
			if err != nil {
				err = state.State.Set("Dualsense not found")
				if err != nil {
					log.Default().Println("Error setting state text:", err)
				}
				err = state.BatteryValue.Set(0)
				if err != nil {
					log.Default().Println("Error setting battery value:", err)
				}
				time.Sleep(5 * time.Second)
				continue
			}
			status, err := battery.ChargingStatus(path)
			if err != nil {
				continue
			}

			// Mise à jour de l'UI Fyne
			err = state.BatteryValue.Set(float64(level) / 100.0)
			if err != nil {
				log.Default().Println("Error setting battery value:", err)
			}
			err = state.State.Set(status)
			if err != nil {
				log.Default().Println("Error setting state text:", err)
			}
			if level != previousLevel {
				select {
				case batteryChan <- float64(level):
				default:
				}

				if level <= state.GlobalState.BatteryAlert && state.GlobalState.BatteryAlert != 0 {
					log.Default().Printf("Battery low (%d%%) for controller at path: %s\n", level, path)
					fyne.CurrentApp().SendNotification(&fyne.Notification{
						Title:   "DualSense Battery Low",
						Content: fmt.Sprintf("Controller %d battery is at %d%%", id, level),
					})
				}
				previousLevel = level
			}
			ledPref, err := state.LedPlayerPreference.Get()
			if err != nil {
				ledPref = ui.PlayerModeNumber
			}
			rgbPref, err := state.LedRGBPreference.Get()
			if err != nil {
				rgbPref = ui.RGBModeBattery
			}

			if (ledPref == ui.PlayerModeBattery) && status == "Charging" {
				if !animActivePlayer {
					var animCtxPlayer context.Context
					animCtxPlayer, animCancelPlayer = context.WithCancel(ctx)
					animActivePlayer = true
					go leds.RunChargingAnimation(animCtxPlayer, path)
				}

			} else {
				if animActivePlayer {
					animCancelPlayer()
					animCancelPlayer = func() {}
					animActivePlayer = false
				}

				if ledPref == ui.PlayerModeBattery {
					leds.SetBatteryLeds(path, float64(level))
				} else {
					leds.SetPlayerNumber(path, id) // Mode Numéro de manette
				}

			}

			if status == "Charging" && (rgbPref == ui.RGBModeBattery) {
				if !animActiveRGB {
					var animCtxRGB context.Context
					animCtxRGB, animCancelRGB = context.WithCancel(ctx)
					animActiveRGB = true
					go leds.RunRGBChargingAnimation(animCtxRGB, path, batteryChan)
					if level != previousLevel {
						select {
						case batteryChan <- float64(level):
							previousLevel = level
						default:
						}
					}
				}
			} else {
				if animActiveRGB {
					animCancelRGB()
					animCancelRGB = func() {}
					animActiveRGB = false
				}

				switch rgbPref {
				case ui.RGBModeBattery:
					leds.SetBatteryColor(path, float64(level))
				case ui.RGBModeStatic:
					hexColor, err := state.LedRGBStaticColor.Get()
					if err != nil {
						hexColor = "0000FF"
					}
					r, g, b := hexToRGB(hexColor)
					leds.SetLightbarRGB(path, r, g, b)

				case ui.RGBModeOff:
					leds.SetLightbarRGB(path, 0, 0, 0)
				}
			}

			time.Sleep(1 * time.Second)
		}
	}
}

// StartActivityLoop monitors inactivity and triggers auto-disconnect when idle.
func StartActivityLoop(ctx context.Context, state *ui.ControllerState, activityChan chan time.Time, path string) {

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
			status, err := state.State.Get()
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

			currentChoice := state.GlobalState.DelayIdleMinutes

			if currentChoice == 0 {
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

			limit := time.Duration(currentChoice) * time.Minute
			err = state.LastActivityBinding.Set(fmt.Sprintf("Inactive : %s / %d min", diff.Truncate(time.Second), currentChoice))
			if err != nil {
				log.Default().Println("Error setting last activity binding:", err)
			}
			if diff > limit {
				log.Default().Println("Auto disconnect !")
				// prefer cached MAC from UI state to avoid repeated sysfs reads
				mac, err := state.Mac.Get()
				if err != nil {
					continue
				}
				if mac != "" {
					err := bluetooth.DisconnectDualSenseNative(mac)
					if err != nil {
						log.Default().Println("Fail D-Bus:", err)
					}
				}
			}
		}
	}
}

// StartControllerManager watches for controllers and creates UI tabs for them.
func StartControllerManager(myApp fyne.App, globalState *ui.GlobalState, conf *config.Config) *container.AppTabs {
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
			foundPaths, err := discovery.FindAllDualSense()
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
					mac := bluetooth.ControllerMAC(path)
					ctrlConf := conf.ControllerConfig(mac)
					newTab := ui.CreateNewControllerTab(globalState, path, conf, ctrlConf, mac, id+1)
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

// ShortMAC returns a short (last 5 chars) representation of the MAC address.
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
