package main

import (
	"dualsense/internal/config"
	"dualsense/internal/service"
	"flag"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)

func main() {

	hidePtr := flag.Bool("hide", false, "Start the application hidden in the system tray")
	flag.Parse()

	conf := config.Load()
	myApp := app.NewWithID("com.dualsense.manager")
	myWindow := myApp.NewWindow("DualSense Manager")

	iconData, _ := os.ReadFile("icon.png")
	myIcon := fyne.NewStaticResource("icon.png", iconData)
	myApp.SetIcon(myIcon)

	if desk, ok := myApp.(desktop.App); ok {
		menu := fyne.NewMenu("DualSense",
			fyne.NewMenuItem("Afficher", func() { myWindow.Show() }),
			fyne.NewMenuItem("Quitter", func() { myApp.Quit() }),
		)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(myIcon)
	}

	myWindow.SetCloseIntercept(func() {
		myWindow.Hide()
	})

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
