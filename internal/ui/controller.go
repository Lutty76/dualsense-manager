package ui

import (
	"context"
	"dualsense/internal/config"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
)

type ControllerTab struct {
	Path         string
	State        *AppState
	ActivityChan chan time.Time
	Container    *fyne.Container
	CancelFunc   context.CancelFunc
	MacAddress   string
}

func CreateNewControllerTab(path string, conf *config.Config, ctrlConf *config.ControllerConfig, macAddress string, id int) *ControllerTab {
	state := &AppState{
		ControllerId:        binding.NewInt(),
		BatteryValue:        binding.NewFloat(),
		BatteryText:         binding.NewString(),
		StateText:           binding.NewString(),
		LastActivityBinding: binding.NewString(),
		MacText:             binding.NewString(),
		SelectedDuration:    binding.NewString(),
		DeadzoneValue:       binding.NewFloat(),
		LedPlayerPreference: binding.NewInt(),
		LedRGBPreference:    binding.NewInt(),
		LedRGBStaticColor:   binding.NewString(),
	}

	err := state.ControllerId.Set(id)
	if err != nil {
		fmt.Println("Error setting controller ID:", err)
	}
	err = state.DeadzoneValue.Set(float64(ctrlConf.Deadzone))
	if err != nil {
		fmt.Println("Error setting deadzone value:", err)
	}
	err = state.LedRGBPreference.Set(ctrlConf.LedRGBPreference)
	if err != nil {
		fmt.Println("Error setting LED RGB preference:", err)
	}
	err = state.LedPlayerPreference.Set(ctrlConf.LedPlayerPreference)
	if err != nil {
		fmt.Println("Error setting LED player preference:", err)
	}
	err = state.MacText.Set("MAC : " + macAddress)
	if err != nil {
		fmt.Println("Error setting MAC text:", err)
	}
	// initialize static RGB color binding without leading '#'
	if ctrlConf.LedRGBStatic != "" {
		err = state.LedRGBStaticColor.Set(strings.TrimPrefix(ctrlConf.LedRGBStatic, "#"))
		if err != nil {
			fmt.Println("Error setting LED RGB static color:", err)
		}
	}
	initialValue := fmt.Sprintf("%d min", conf.IdleMinutes)
	if conf.IdleMinutes == 0 {
		initialValue = "Jamais"
	}
	err = state.SelectedDuration.Set(initialValue)
	if err != nil {
		fmt.Println("Error setting selected duration:", err)
	}

	activityChan := make(chan time.Time)

	uiContent := CreateContent(conf, ctrlConf, state)

	return &ControllerTab{
		Path:         path,
		State:        state,
		ActivityChan: activityChan,
		Container:    container.NewPadded(uiContent),
		MacAddress:   macAddress,
	}
}
