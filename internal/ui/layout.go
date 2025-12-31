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
	options := []string{"1 min", "2 min", "5 min", "10 min", "30 min", "Never"}

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

	if conf.IdleMinutes == 0 {
		selectWidget.SetSelected("Never")
	} else {
		selectWidget.SetSelected(fmt.Sprintf("%d min", conf.IdleMinutes))
	}

	deadzoneSlider := widget.NewSliderWithData(0, 10000, state.DeadzoneValue)
	deadzoneSlider.Step = 250
	deadzoneLabel := widget.NewLabel(fmt.Sprintf("Deadzone : %d", conf.Deadzone))
	deadzoneSlider.OnChanged = func(v float64) {
		val := int(v)
		deadzoneLabel.SetText(fmt.Sprintf("Deadzone : %d", val))
		conf.Deadzone = val
		state.DeadzoneValue.Set(v)
		config.Save(conf)
	}

	names := []string{playerOptions[0], playerOptions[1]}

	ledSelect := widget.NewSelect(names, func(selected string) {
		for id, name := range playerOptions {
			if name == selected {
				state.LedPlayerPreference.Set(id)
				conf.LedPlayerPreference = id

				config.Save(conf)
				break
			}
		}
	})

	namesRgb := []string{rgbOptions[0], rgbOptions[1], rgbOptions[2]}

	rgbSelect := widget.NewSelect(namesRgb, func(selected string) {
		for id, name := range rgbOptions {
			if name == selected {
				state.LedRGBPreference.Set(id)
				conf.LedRGBPreference = id

				config.Save(conf)
				break
			}
		}
	})

	currentID, _ := state.LedPlayerPreference.Get()
	ledSelect.SetSelected(playerOptions[currentID])
	currentIDRGB, _ := state.LedRGBPreference.Get()
	rgbSelect.SetSelected(rgbOptions[currentIDRGB])

	return container.NewVBox(
		widget.NewLabelWithData(state.BatteryText),
		widget.NewProgressBarWithData(state.BatteryValue),
		widget.NewLabelWithData(state.StateText),
		widget.NewLabelWithData(state.MacText),
		widget.NewSeparator(),
		container.NewVBox(widget.NewLabel("Player LED :"), ledSelect),
		container.NewVBox(widget.NewLabel("RGB LED :"), rgbSelect),
		container.NewVBox(widget.NewLabel("Delay :"), selectWidget),
		deadzoneLabel,
		deadzoneSlider,
		widget.NewSeparator(),
		widget.NewLabelWithData(state.LastActivityBinding),
	)
}
