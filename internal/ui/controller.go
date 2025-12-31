package ui

import (
	"context"
	"dualsense/internal/config"
	"fmt"
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

func CreateNewControllerTab(path string, conf *config.Config) *ControllerTab {
	state := &AppState{
		BatteryValue:        binding.NewFloat(),
		BatteryText:         binding.NewString(),
		StateText:           binding.NewString(),
		LastActivityBinding: binding.NewString(),
		MacText:             binding.NewString(),
		SelectedDuration:    binding.NewString(),
		DeadzoneValue:       binding.NewFloat(),
		LedPlayerPreference: binding.NewInt(),
		LedRGBPreference:    binding.NewInt(),
	}
	state.DeadzoneValue.Set(float64(conf.Deadzone))
	state.LedRGBPreference.Set(conf.LedRGBPreference)
	state.LedPlayerPreference.Set(conf.LedPlayerPreference)
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
