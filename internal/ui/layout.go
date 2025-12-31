package ui

import (
	"dualsense/internal/config"
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/widget"
)

type AppState struct {
	BatteryValue        binding.Float
	BatteryText         binding.String
	StateText           binding.String
	LastActivityBinding binding.String
	MacText             binding.String
	SelectedDuration    binding.String
	DeadzoneValue       binding.Float
	LedPlayerPreference binding.Int
	LedRGBPreference    binding.Int
}

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

func CreateContent(conf *config.Config, state *AppState) fyne.CanvasObject {
	options := []string{"1 min", "2 min", "5 min", "10 min", "20 min", "30 min", "40 min", "Never"}
	optionsBattery := []string{"5 %", "15 %", "25 %", "Never"}

	// helper to extract MAC address from state.MacText binding (value like "MAC : xx:xx:...")
	getMac := func() string {
		s, _ := state.MacText.Get()
		s = strings.TrimPrefix(s, "MAC :")
		s = strings.TrimSpace(s)
		return s
	}

	// load controller-specific config only when MAC is available
	mac := getMac()
	var ctrlConf *config.ControllerConfig
	if mac != "" {
		ctrlConf = conf.GetControllerConfig(mac)
	}

	selectWidget := widget.NewSelect(options, func(value string) {
		state.SelectedDuration.Set(value)
		if value == "Never" {
			conf.IdleMinutes = 0
		} else {
			min, _ := strconv.Atoi(strings.Split(value, " ")[0])
			conf.IdleMinutes = min
		}
		config.Save(conf)
	})

	// initialize selection from global config
	if conf.IdleMinutes == 0 {
		selectWidget.SetSelected("Never")
	} else {
		selectWidget.SetSelected(fmt.Sprintf("%d min", conf.IdleMinutes))
	}

	selectBatteryWidget := widget.NewSelect(optionsBattery, func(value string) {
		if value == "Never" {
			conf.BatteryAlert = 0
		} else {
			percent, _ := strconv.Atoi(strings.Split(value, " ")[0])
			conf.BatteryAlert = percent
		}
		config.Save(conf)
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
		if mac := getMac(); mac != "" {
			if conf.Controllers == nil {
				conf.Controllers = map[string]config.ControllerConfig{}
			}
			cc := conf.Controllers[mac]
			cc.Deadzone = val
			conf.Controllers[mac] = cc
			state.DeadzoneValue.Set(v)
			config.Save(conf)
		}
	}

	names := []string{playerOptions[0], playerOptions[1]}

	ledSelect := widget.NewSelect(names, func(selected string) {
		for id, name := range playerOptions {
			if name == selected {
				state.LedPlayerPreference.Set(id)
				if mac := getMac(); mac != "" {
					if conf.Controllers == nil {
						conf.Controllers = map[string]config.ControllerConfig{}
					}
					cc := conf.Controllers[mac]
					cc.LedPlayerPreference = id
					conf.Controllers[mac] = cc
					config.Save(conf)
				}
				break
			}
		}
	})

	namesRgb := []string{rgbOptions[0], rgbOptions[1], rgbOptions[2]}

	rgbSelect := widget.NewSelect(namesRgb, func(selected string) {
		for id, name := range rgbOptions {
			if name == selected {
				state.LedRGBPreference.Set(id)
				if mac := getMac(); mac != "" {
					if conf.Controllers == nil {
						conf.Controllers = map[string]config.ControllerConfig{}
					}
					cc := conf.Controllers[mac]
					cc.LedRGBPreference = id
					conf.Controllers[mac] = cc
					config.Save(conf)
				}
				break
			}
		}
	})
	// no direct size calls; we'll let containers expand the selects

	// initialize state bindings from controller config only when MAC present
	if ctrlConf != nil {
		_ = state.DeadzoneValue.Set(float64(ctrlConf.Deadzone))
		_ = state.LedPlayerPreference.Set(ctrlConf.LedPlayerPreference)
		_ = state.LedRGBPreference.Set(ctrlConf.LedRGBPreference)
	}
	currentID, _ := state.LedPlayerPreference.Get()
	ledSelect.SetSelected(playerOptions[currentID])
	currentIDRGB, _ := state.LedRGBPreference.Get()
	rgbSelect.SetSelected(rgbOptions[currentIDRGB])

	return container.NewVBox(
		widget.NewLabelWithData(state.BatteryText),
		widget.NewProgressBarWithData(state.BatteryValue),
		widget.NewLabelWithData(state.StateText),
		widget.NewLabelWithData(state.MacText),
		container.NewBorder(nil, nil, widget.NewLabel("Player LED :"), nil, ledSelect),
		container.NewBorder(nil, nil, widget.NewLabel("RGB LED :"), nil, rgbSelect),
		deadzoneLabel,
		deadzoneSlider,
		widget.NewSeparator(),
		widget.NewSeparator(),
		container.NewBorder(nil, nil, widget.NewLabel("Battery alert :"), nil, selectBatteryWidget),
		container.NewBorder(nil, nil, widget.NewLabel("Delay :"), nil, selectWidget),
		widget.NewLabelWithData(state.LastActivityBinding),
	)
}
