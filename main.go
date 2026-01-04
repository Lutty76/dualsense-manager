// Command dualsense-mgr provides a system tray UI to monitor DualSense controllers.
package main

import (
	"dualsense/internal/config"
	"dualsense/internal/service"
	"flag"
	"log"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)

var Version = "dev"

func main() {

	hidePtr := flag.Bool("hide", false, "Start the application hidden in the system tray")
	flag.BoolVar(hidePtr, "h", false, "Start the application hidden in the system tray (shorthand)")
	debugPtr := flag.Bool("debug", false, "Enable debug logging")
	flag.BoolVar(debugPtr, "d", false, "Enable debug logging (shorthand)")
	versionPtr := flag.Bool("version", false, "Show version information")
	flag.BoolVar(versionPtr, "v", false, "Show version information (shorthand)")

	flag.Parse()

	if *versionPtr {
		log.Default().Printf("DualSense Manager version %s\n", Version)
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
			fyne.NewMenuItem("Afficher", func() { myWindow.Show() }),
			fyne.NewMenuItem("Quitter", func() { myApp.Quit() }),
		)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(resourceIconPng)
	}

	myWindow.SetCloseIntercept(func() {
		myWindow.Hide()
	})

	service.Debug = *debugPtr
	controllerTabs := service.StartControllerManager(myApp, conf)

	myWindow.SetContent(controllerTabs)
	myWindow.Resize(fyne.NewSize(300, 300))

	if *hidePtr {
		// start the application hidden: run app loop without showing the window
		myApp.Run()
	} else {
		myWindow.ShowAndRun()
	}
}
