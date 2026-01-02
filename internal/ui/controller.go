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

	state.ControllerId.Set(id)
	state.DeadzoneValue.Set(float64(ctrlConf.Deadzone))
	state.LedRGBPreference.Set(ctrlConf.LedRGBPreference)
	state.LedPlayerPreference.Set(ctrlConf.LedPlayerPreference)
	state.MacText.Set("MAC : " + macAddress)
	// initialize static RGB color binding without leading '#'
	if ctrlConf.LedRGBStatic != "" {
		state.LedRGBStaticColor.Set(strings.TrimPrefix(ctrlConf.LedRGBStatic, "#"))
	}
	initialValue := fmt.Sprintf("%d min", conf.IdleMinutes)
	if conf.IdleMinutes == 0 {
		initialValue = "Jamais"
	}
	state.SelectedDuration.Set(initialValue)

	activityChan := make(chan time.Time)

	uiContent := CreateContent(conf, state)

	return &ControllerTab{
		Path:         path,
		State:        state,
		ActivityChan: activityChan,
		Container:    container.NewPadded(uiContent),
	}
}
