package main

import (
	"dualsense/internal/config"
	"dualsense/internal/service"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/driver/desktop"
)

func main() {
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
	myWindow.Resize(fyne.NewSize(400, 300))
	myWindow.ShowAndRun()
}
