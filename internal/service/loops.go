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

type ControllerCLI struct {
	Path         string
	ActivityChan chan time.Time
	CancelFunc   context.CancelFunc
	MacAddress   string
	Status       string
}

type LedState struct {
	PlayerAnimationActive bool
	RGBAnimationActive    bool
	CancelPlayerAnim      context.CancelFunc
	CancelRGBAnim         context.CancelFunc
	LedPlayerMode         int
	LedRGBMode            int
	PreviousBatteryLevel  int
	PlayerNumber          int
	RGBColor              string
}

// ManageBatteryAndLEDs handles battery monitoring and LED management for a controller.
func ManageBatteryAndLEDs(ctx context.Context, state *ui.ControllerState, ctrlConf *config.ControllerConfig, conf *config.Config, path string, id int, storedStatus *string) {
	var animCancelPlayer context.CancelFunc = func() {}
	var animCancelRGB context.CancelFunc = func() {}
	var animActivePlayer bool
	var animActiveRGB bool
	var firstIteration = true
	batteryChan := make(chan float64)
	previousLevel := -1

	var ledState LedState = LedState{
		PlayerAnimationActive: false,
		RGBAnimationActive:    false,
		CancelPlayerAnim:      func() {},
		CancelRGBAnim:         func() {},
		LedPlayerMode:         -1,
		LedRGBMode:            -1,
		PreviousBatteryLevel:  -1,
		PlayerNumber:          -1,
		RGBColor:              "",
	}

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
				if state != nil {
					err = state.State.Set("Dualsense not found")
					if err != nil {
						log.Default().Println("Error setting state text:", err)
					}
					err = state.BatteryValue.Set(0)
					if err != nil {
						log.Default().Println("Error setting battery value:", err)
					}
					state.Status = "Dualsense not found"
				}
				time.Sleep(5 * time.Second)
				continue
			}
			status, err := battery.ChargingStatus(path)
			if err != nil {
				continue
			}
			// Mise à jour de l'UI Fyne
			if state != nil && *storedStatus != status {
				err = state.BatteryValue.Set(float64(level) / 100.0)
				if err != nil {
					log.Default().Println("Error setting battery value:", err)
				}
				err = state.State.Set(status)
				if err != nil {
					log.Default().Println("Error setting state text:", err)
				}
				state.Status = status
			}

			if storedStatus != nil {
				*storedStatus = status
			}
			if level != previousLevel || firstIteration {
				select {
				case batteryChan <- float64(level):
					firstIteration = false
				default:
				}

				if level <= conf.BatteryAlert && conf.BatteryAlert != 0 && status != "Charging" {
					log.Default().Printf("Battery low (%d%%) for controller at path: %s\n", level, path)
					fyne.CurrentApp().SendNotification(&fyne.Notification{
						Title:   "DualSense Battery Low",
						Content: fmt.Sprintf("Controller %d battery is at %d%%", id, level),
					})
				}
				previousLevel = level
			}

			ledPref := ctrlConf.LedPlayerPreference
			rgbPref := ctrlConf.LedRGBPreference

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

					if ledState.LedPlayerMode != ui.PlayerModeBattery || ledState.PreviousBatteryLevel != level {
						leds.SetBatteryLeds(path, float64(level))
						ledState.LedPlayerMode = ui.PlayerModeBattery
					}

				} else {
					if ledState.LedPlayerMode != ui.PlayerModeNumber || ledState.PlayerNumber != id {
						leds.SetPlayerNumber(path, id) // Mode Numéro de manette
						ledState.LedPlayerMode = ui.PlayerModeNumber
						ledState.PlayerNumber = id
					}
				}

			}
			if status == "Charging" && (rgbPref == ui.RGBModeBattery) {
				if !animActiveRGB {

					var animCtxRGB context.Context
					animCtxRGB, animCancelRGB = context.WithCancel(ctx)
					animActiveRGB = true
					go leds.RunRGBChargingAnimation(animCtxRGB, path, batteryChan)
				}
			} else {
				if animActiveRGB {
					animCancelRGB()
					animCancelRGB = func() {}
					animActiveRGB = false
					firstIteration = true
				}

				switch rgbPref {
				case ui.RGBModeBattery:
					if ledState.LedRGBMode != ui.RGBModeBattery || ledState.PreviousBatteryLevel != level {
						leds.SetBatteryColor(path, float64(level))
						ledState.LedRGBMode = ui.RGBModeBattery
					}
				case ui.RGBModeStatic:
					if ledState.LedRGBMode != ui.RGBModeStatic || ledState.RGBColor != ctrlConf.LedRGBStatic {

						r, g, b := hexToRGB(ctrlConf.LedRGBStatic)
						leds.SetLightbarRGB(path, r, g, b)
						ledState.LedRGBMode = ui.RGBModeStatic
						ledState.RGBColor = ctrlConf.LedRGBStatic
					}

				case ui.RGBModeOff:
					if ledState.LedRGBMode != ui.RGBModeOff {
						leds.SetLightbarRGB(path, 0, 0, 0)
						ledState.LedRGBMode = ui.RGBModeOff
					}
				}
			}

			time.Sleep(1 * time.Second)

			ledState.PreviousBatteryLevel = level
		}
	}
}

// StartActivityLoop monitors inactivity and triggers auto-disconnect when idle.
func StartActivityLoop(ctx context.Context, state *ui.ControllerState, activityChan chan time.Time, conf *config.Config, mac string, path string, storedStatus *string) {

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
			if state != nil {
				err := state.LastActivityBinding.Set("In use")

				if err != nil {
					log.Default().Println("Error setting last activity binding:", err)
				}
			}
		case <-ticker.C:
			var status string
			if storedStatus != nil {
				status = *storedStatus
			} else {
				status = ""
			}

			if strings.Contains(status, "not found") || strings.Contains(status, "Recherche") {
				lastActivityTime = time.Now()

				if state != nil {
					err := state.LastActivityBinding.Set("Disconnected")
					if err != nil {
						log.Default().Println("Error setting last activity binding:", err)
					}
				}
				continue
			}
			diff := time.Since(lastActivityTime)

			currentChoice := conf.IdleMinutes

			if state != nil {
				if currentChoice == 0 {
					err := state.LastActivityBinding.Set(fmt.Sprintf("Inactive : %s (Auto-off Disabled)", diff.Truncate(time.Second)))
					if err != nil {
						log.Default().Println("Error setting last activity binding:", err)
					}
					continue
				}
				if strings.Contains(status, "Charging") || strings.Contains(status, "Full") {
					err := state.LastActivityBinding.Set(fmt.Sprintf("Inactive : %s (disabled due to charging)", diff.Truncate(time.Second)))
					if err != nil {
						log.Default().Println("Error setting last activity binding:", err)
					}
					continue
				}
			}

			limit := time.Duration(currentChoice) * time.Minute

			if state != nil {
				err := state.LastActivityBinding.Set(fmt.Sprintf("Inactive : %s / %d min", diff.Truncate(time.Second), currentChoice))
				if err != nil {
					log.Default().Println("Error setting last activity binding:", err)
				}
			}
			if diff > limit {
				log.Default().Println("Auto disconnect !")

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
func StartControllerManager(globalState *ui.GlobalState, conf *config.Config) *container.AppTabs {
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

					go MonitorJoystick(path, newTab.ActivityChan, ctrlConf)
					go ManageBatteryAndLEDs(ctx, newTab.State, ctrlConf, conf, path, id+1, &newTab.State.Status)
					go StartActivityLoop(ctx, newTab.State, newTab.ActivityChan, conf, mac, path, &newTab.State.Status)

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

func StartControllerManagerCLI(conf *config.Config) {
	if Debug {
		log.Default().Println("StartControllerManagerCLI: Debug mode enabled")
	}
	activeControllers := make(map[string]*ControllerCLI)

	go func() {
		for {
			foundPaths, err := discovery.FindAllDualSense()
			if err != nil {
				log.Default().Println("Error finding DualSense controllers:", err)
				return
			}
			for id, path := range foundPaths {
				ctx, cancel := context.WithCancel(context.Background())
				mac := bluetooth.ControllerMAC(path)
				ctrlConf := conf.ControllerConfig(mac)
				if _, exists := activeControllers[path]; !exists {

					if Debug {
						log.Default().Println("New DualSense detected at path:", path)
					}
					activityChan := make(chan time.Time)
					activeControllers[path] = &ControllerCLI{
						Path:         path,
						ActivityChan: activityChan,
						CancelFunc:   cancel,
						MacAddress:   mac,
					}
					go MonitorJoystick(path, activeControllers[path].ActivityChan, ctrlConf)
					go ManageBatteryAndLEDs(ctx, nil, ctrlConf, conf, path, id+1, &activeControllers[path].Status)
					go StartActivityLoop(ctx, nil, activeControllers[path].ActivityChan, conf, mac, path, &activeControllers[path].Status)

					defer cancel()
				}

			}

			time.Sleep(2 * time.Second)
		}
	}()

	select {}
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
