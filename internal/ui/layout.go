// Package ui contains the Fyne UI components and bindings used by the application.
package ui

import (
	"dualsense/internal/config"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

// AppState contains data bindings representing a controller UI state.
type AppState struct {
	ControllerID        binding.Int
	BatteryValue        binding.Float
	BatteryText         binding.String
	StateText           binding.String
	LastActivityBinding binding.String
	MacText             binding.String
	SelectedDuration    binding.String
	DeadzoneValue       binding.Float
	LedPlayerPreference binding.Int
	LedRGBPreference    binding.Int
	LedRGBStaticColor   binding.String
}

// Player and RGB modes used in UI selections.
const (
	PlayerModeBattery = 0
	PlayerModeNumber  = 1

	RGBModeBattery = 0
	RGBModeStatic  = 1
	RGBModeOff     = 2
)

var playerOptions = map[int]string{
	PlayerModeBattery: "Battery level",
	PlayerModeNumber:  "Player number",
}

var rgbOptions = map[int]string{
	RGBModeBattery: "Battery level",
	RGBModeStatic:  "Static color",
	RGBModeOff:     "Disable",
}

// CreateContent builds the controller UI content for a tab.
func CreateContent(conf *config.Config, ctrlConf *config.ControllerConfig, state *AppState) fyne.CanvasObject {
	options := []string{"1 min", "2 min", "5 min", "10 min", "20 min", "30 min", "40 min", "Never"}
	optionsBattery := []string{"5 %", "15 %", "25 %", "Never"}

	macText, err := state.MacText.Get()
	if err != nil {
		macText = ""
	}
	mac := strings.TrimSpace(strings.TrimPrefix(macText, "MAC :"))

	// helper to update per-controller config and persist changes
	saveCtrl := func(mac string, update func(*config.ControllerConfig)) {
		if mac == "" {
			return
		}
		if conf.Controllers == nil {
			conf.Controllers = map[string]config.ControllerConfig{}
		}
		cc := conf.Controllers[mac]
		update(&cc)
		conf.Controllers[mac] = cc
		err := config.Save(conf)
		if err != nil {
			log.Default().Println("Error saving controller config for", mac, ":", err)
		}
	}

	selectDelayWidget := widget.NewSelect(options, func(value string) {
		err = state.SelectedDuration.Set(value)
		if err != nil {
			log.Default().Println("Error setting selected duration:", err)
		}
		if value == "Never" {
			conf.IdleMinutes = 0
		} else {
			minutes, err := strconv.Atoi(strings.Split(value, " ")[0])
			if err != nil {
				log.Default().Printf("Unable to parse delay : %s", value)
				return
			}
			conf.IdleMinutes = minutes
		}

		err := config.Save(conf)
		if err != nil {
			log.Default().Println("Error saving controller config for", mac, ":", err)
		}
	})

	// initialize selection from global config
	if conf.IdleMinutes == 0 {
		selectDelayWidget.SetSelected("Never")
	} else {
		selectDelayWidget.SetSelected(fmt.Sprintf("%d min", conf.IdleMinutes))
	}

	selectBatteryWidget := widget.NewSelect(optionsBattery, func(value string) {
		if value == "Never" {
			conf.BatteryAlert = 0
		} else {
			percent, err := strconv.Atoi(strings.Split(value, " ")[0])
			if err != nil {
				log.Default().Printf("Unable to parse battery alert : %s", value)
				return
			}
			conf.BatteryAlert = percent
		}

		err := config.Save(conf)
		if err != nil {
			log.Default().Println("Error saving controller config for", mac, ":", err)
		}
	})

	// initialize selection from global config
	if conf.BatteryAlert == 0 {
		selectBatteryWidget.SetSelected("Never")
	} else {
		selectBatteryWidget.SetSelected(fmt.Sprintf("%d %%", conf.BatteryAlert))
	}

	deadzoneSlider := widget.NewSliderWithData(0, 10000, state.DeadzoneValue)
	deadzoneSlider.Step = 250
	// initialize deadzone label from per-controller config when available,
	// otherwise keep current state binding value
	initialDeadzone := 0
	if ctrlConf != nil {
		initialDeadzone = ctrlConf.Deadzone
	} else {
		if v, err := state.DeadzoneValue.Get(); err == nil {
			initialDeadzone = int(v)
		}
	}
	deadzoneLabel := widget.NewLabel(fmt.Sprintf("Deadzone : %d", initialDeadzone))
	deadzoneSlider.OnChanged = func(v float64) {
		val := int(v)
		deadzoneLabel.SetText(fmt.Sprintf("Deadzone : %d", val))
		// save per-controller if mac known
		if mac != "" {
			err = state.DeadzoneValue.Set(v)
			if err != nil {
				log.Default().Println("Error setting deadzone value:", err)
			}

			saveCtrl(mac, func(cc *config.ControllerConfig) { cc.Deadzone = val })
		}
	}

	names := []string{playerOptions[0], playerOptions[1]}

	ledSelect := widget.NewSelect(names, func(selected string) {
		for id, name := range playerOptions {
			if name == selected {
				err = state.LedPlayerPreference.Set(id)
				if err != nil {
					log.Default().Println("Error setting LED player preference:", err)
				}
				if mac != "" {
					saveCtrl(mac, func(cc *config.ControllerConfig) { cc.LedPlayerPreference = id })
				}
				break
			}
		}
	})

	namesRgb := []string{rgbOptions[0], rgbOptions[1], rgbOptions[2]}

	rgbSelect := widget.NewSelect(namesRgb, nil)

	// initialize state bindings from controller config only when MAC present
	if ctrlConf != nil {
		_ = state.DeadzoneValue.Set(float64(ctrlConf.Deadzone))
		_ = state.LedPlayerPreference.Set(ctrlConf.LedPlayerPreference)
		_ = state.LedRGBPreference.Set(ctrlConf.LedRGBPreference)
		// only populate static color binding from config when the binding is empty
		if ctrlConf.LedRGBStatic != "" {
			if cur, err := state.LedRGBStaticColor.Get(); err != nil || cur == "" {
				_ = state.LedRGBStaticColor.Set(strings.TrimPrefix(ctrlConf.LedRGBStatic, "#"))
			}
		}
	}
	currentID, err := state.LedPlayerPreference.Get()
	if err != nil {
		currentID = PlayerModeNumber
	}
	ledSelect.SetSelected(playerOptions[currentID])
	currentIDRGB, err := state.LedRGBPreference.Get()
	if err != nil {
		currentIDRGB = RGBModeBattery
	}
	rgbSelect.SetSelected(rgbOptions[currentIDRGB])

	staticColorEntry := widget.NewEntryWithData(state.LedRGBStaticColor)
	staticColorEntry.SetPlaceHolder("FFFFFF")

	validationLabel := canvas.NewText("Invalid hex (RRGGBB)", color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF})
	validationLabel.TextSize = 12
	validationLabel.Hide()

	hexRegex := regexp.MustCompile(`(?i)^[0-9A-F]{6}$`)
	var hexSaveTimer *time.Timer
	const saveDebounce = 800 * time.Millisecond

	staticColorEntry.OnChanged = func(s string) {
		// normalize: remove leading '#', trim spaces, uppercase
		norm := strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(s, "#")))
		// update entry text to normalized value if different
		if norm != s {
			staticColorEntry.SetText(norm)
		}
		// ensure the state binding holds the normalized value
		_ = state.LedRGBStaticColor.Set(norm)

		// cancel pending save if input is invalid; show validation hint
		if !hexRegex.MatchString(norm) {
			validationLabel.Show()
			if hexSaveTimer != nil {
				hexSaveTimer.Stop()
				hexSaveTimer = nil
			}
			return
		}
		validationLabel.Hide()

		// debounce save: stop previous timer and schedule a new save
		if hexSaveTimer != nil {
			hexSaveTimer.Stop()
		}
		// capture current normalized value for the save closure
		val := norm
		hexSaveTimer = time.AfterFunc(saveDebounce, func() {
			if mac == "" {
				return
			}
			saveCtrl(mac, func(cc *config.ControllerConfig) { cc.LedRGBStatic = "#" + val })
		})
	}
	staticColorContainer := container.NewBorder(nil, nil, widget.NewLabel("Static Color Hex (RRGGBB): "), nil, container.NewVBox(staticColorEntry, validationLabel))

	if currentIDRGB == RGBModeStatic {
		staticColorContainer.Show()
	} else {
		staticColorContainer.Hide()
	}

	// ensure rgbSelect shows or hides the static-color entry when changed
	rgbSelect.OnChanged = func(selected string) {
		for id, name := range rgbOptions {
			if name == selected {
				err = state.LedRGBPreference.Set(id)
				if err != nil {
					log.Default().Println("Error setting LED RGB preference:", err)
				}
				if mac != "" {
					saveCtrl(mac, func(cc *config.ControllerConfig) { cc.LedRGBPreference = id })
				}
				if id == RGBModeStatic {
					staticColorContainer.Show()
				} else {
					staticColorContainer.Hide()
				}
				break
			}
		}
	}

	controllerID, err := state.ControllerID.Get()
	if err != nil {
		log.Default().Println("Error getting controller ID:", err)
		controllerID = 0
	}

	return container.NewVBox(
		widget.NewLabel("Controller n°"+strconv.Itoa(controllerID)),
		widget.NewLabelWithData(state.BatteryText),
		widget.NewProgressBarWithData(state.BatteryValue),
		widget.NewLabelWithData(state.StateText),
		widget.NewLabelWithData(state.MacText),
		container.NewBorder(nil, nil, widget.NewLabel("Player LED :"), nil, ledSelect),
		container.NewBorder(nil, nil, widget.NewLabel("RGB LED :"), nil, rgbSelect),
		staticColorContainer,
		deadzoneLabel,
		deadzoneSlider,
		widget.NewSeparator(),
		widget.NewSeparator(),
		container.NewBorder(nil, nil, widget.NewLabel("Battery alert :"), nil, selectBatteryWidget),
		container.NewBorder(nil, nil, widget.NewLabel("Delay :"), nil, selectDelayWidget),
		widget.NewLabelWithData(state.LastActivityBinding),
	)
}
