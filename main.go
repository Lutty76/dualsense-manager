// Command dualsense-mgr provides a system tray UI to monitor DualSense controllers.
package main

import (
	"dualsense/internal/config"
	"dualsense/internal/service"
	"dualsense/internal/service/leds"
	"dualsense/internal/ui"
	"fmt"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/spf13/cobra"
)

var Version = "dev"

func main() {

	var rootCmd = &cobra.Command{
		Use:   "dualsense-mgr",
		Short: "DualSense Manager is a system tray application to monitor and control DualSense controllers on Linux.",
		Long:  "Dualsense Manager is a system tray application to monitor and control DualSense controllers on Linux. It provides battery status, charging animations, and customizable LED colors. It automatically shuts down the controller after a configurable idle time.",
	}

	hidePtr := rootCmd.PersistentFlags().BoolP("minimize", "m", false, "Start the application minimize in the system tray")
	debugPtr := rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")
	versionPtr := rootCmd.PersistentFlags().BoolP("version", "v", false, "Show version information")
	cliPtr := rootCmd.PersistentFlags().BoolP("cli", "c", false, "Run in CLI mode without UI")

	rootCmd.Run = func(_ *cobra.Command, _ []string) {

		if *versionPtr {
			fmt.Printf("DualSense Manager version %s\n", Version)
			return
		}

		conf, err := config.Load()
		if err != nil {
			log.Fatalf("Error loading configuration: %s\n", err)
			return
		}
		myApp := app.NewWithID("com.dualsense.manager")
		myWindow := myApp.NewWindow("DualSense Manager")

		myApp.SetIcon(resourceIconPng)
		myWindow.SetIcon(resourceIconPng)

		if desk, ok := myApp.(desktop.App); ok {
			menu := fyne.NewMenu("DualSense",
				fyne.NewMenuItem("Display", func() { myWindow.Show() }),
				fyne.NewMenuItem("Quit", func() { myApp.Quit() }),
			)
			desk.SetSystemTrayMenu(menu)
			desk.SetSystemTrayIcon(resourceIconPng)
		}

		myWindow.SetCloseIntercept(func() {
			myWindow.Hide()
		})

		service.Debug = *debugPtr
		leds.Debug = *debugPtr

		globalState := &ui.GlobalState{
			DelayIdleMinutes: conf.IdleMinutes,
			BatteryAlert:     conf.BatteryAlert,
		}

		if *cliPtr {
			log.Default().Println("Starting in CLI mode without UI")
			service.StartControllerManagerCLI(conf)

		} else {

			controllerTabs := service.StartControllerManager(globalState, conf)

			selectBatteryWidget := ui.CreateBatteryWidget(globalState, conf)
			selectDelayWidget := ui.CreateDelayIdleSelect(globalState, conf)

			thickSeparator := canvas.NewRectangle(theme.Color(theme.ColorNameShadow))
			thickSeparator.SetMinSize(fyne.NewSize(0, 3))

			bottomControls := container.NewVBox(
				thickSeparator,
				container.NewBorder(nil, nil, widget.NewLabel("Battery alert :"), nil, selectBatteryWidget),
				container.NewBorder(nil, nil, widget.NewLabel("Delay :"), nil, selectDelayWidget),
			)

			appContainer := container.NewBorder(nil, bottomControls, nil, nil, container.NewStack(controllerTabs))

			myWindow.SetContent(appContainer)
			myWindow.Resize(fyne.NewSize(300, 500))

			if *hidePtr {
				myApp.Run()
			} else {
				myWindow.ShowAndRun()
			}
		}
	}

	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("Error executing command: %s\n", err)
	}
}
