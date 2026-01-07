// Package ui contains the Fyne UI components and bindings used by the application.
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

// ControllerTab represents a UI tab for a single controller.
type ControllerTab struct {
	Path         string
	State        *ControllerState
	ActivityChan chan time.Time
	Container    *fyne.Container
	CancelFunc   context.CancelFunc
	MacAddress   string
}

// CreateNewControllerTab builds a `ControllerTab` with bindings and UI widgets.
func CreateNewControllerTab(globalState *GlobalState, path string, conf *config.Config, ctrlConf *config.ControllerConfig, macAddress string, id int) *ControllerTab {
	state := &ControllerState{
		ControllerID:        binding.NewInt(),
		BatteryValue:        binding.NewFloat(),
		State:               binding.NewString(),
		LastActivityBinding: binding.NewString(),
		Mac:                 binding.NewString(),
		DeadzoneValue:       binding.NewFloat(),
		LedPlayerPreference: binding.NewInt(),
		LedRGBPreference:    binding.NewInt(),
		LedRGBStaticColor:   binding.NewString(),
		GlobalState:         globalState,
		Status:              "",
	}

	err := state.ControllerID.Set(id)
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
	fmt.Println("Setting LED player preference to:", ctrlConf.LedPlayerPreference)
	err = state.LedPlayerPreference.Set(ctrlConf.LedPlayerPreference)
	if err != nil {
		fmt.Println("Error setting LED player preference:", err)
	}
	err = state.Mac.Set(macAddress)
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
