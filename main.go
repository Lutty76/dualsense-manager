package main

import (
	"dualsense/internal/config"
	"dualsense/internal/service"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type JSInterface struct {
	Time  uint32 // timestamp en ms
	Value int16  // valeur de l'axe ou bouton
	Type  uint8  // 1 = bouton, 2 = axe
	Index uint8  // numéro du bouton/axe
}

const (
	Deadzone       = 4000 // Seuil pour ignorer le drift
	InactivityTime = 10 * time.Minute
)

// Chemin vers la LED (à adapter selon ton numéro d'input)
const jsPath = "/dev/input/js1"
const powerPath = "sys/devices/virtual/misc/uhid/0005:054C:0CE6*/power_supply/ps-controller-battery-*/capacity"

func main() {
	conf := config.Load()
	myApp := app.NewWithID("com.dualsense.manager")
	myWindow := myApp.NewWindow("DualSense Manager")

	iconData, _ := os.ReadFile("icon.png") // Assure-toi d'avoir un petit PNG
	myIcon := fyne.NewStaticResource("icon.png", iconData)
	myApp.SetIcon(myIcon)

	if desk, ok := myApp.(desktop.App); ok {
		menu := fyne.NewMenu("DualSense",
			fyne.NewMenuItem("Afficher", func() {
				myWindow.Show()
			}),
			fyne.NewMenuItem("Quitter", func() {
				myApp.Quit()
			}),
		)
		desk.SetSystemTrayMenu(menu)
		desk.SetSystemTrayIcon(myIcon)
	}

	myWindow.SetCloseIntercept(func() {
		myWindow.Hide()
	})

	// 1. Création des Bindings (les variables réactives)
	batteryValue := binding.NewFloat()
	batteryText := binding.NewString()
	stateText := binding.NewString()
	lastActivityBinding := binding.NewString()
	lastActivityBinding.Set("Aucune activité")
	// 2. Éléments de l'interface liés aux variables
	title := widget.NewLabelWithStyle("DualSense Controller", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// On lie le label et la bar aux bindings
	batteryLabel := widget.NewLabelWithData(batteryText)
	batteryBar := widget.NewProgressBarWithData(batteryValue)

	statusLabel := widget.NewLabelWithData(stateText)
	activityLabel := widget.NewLabelWithData(lastActivityBinding)

	macText := binding.NewString()
	macLabel := widget.NewLabelWithData(macText)

	options := []string{"1 min", "2 min", "5 min", "10 min", "30 min", "Jamais"}
	selectedDuration := binding.NewString()
	selectedDuration.Set("10 min")
	selectWidget := widget.NewSelect(options, func(value string) {
		if value == "Jamais" {
			conf.IdleMinutes = 0
		} else {
			// "10 min" -> 10
			min, _ := strconv.Atoi(strings.Split(value, " ")[0])
			conf.IdleMinutes = min
		}
		// Sauvegarde automatique à chaque changement
		config.Save(conf)
		selectedDuration.Set(value)
		fmt.Println("Nouveau délai :", value)
	})
	if conf.IdleMinutes == 0 {
		selectWidget.SetSelected("Jamais")
	} else {
		selectWidget.SetSelected(fmt.Sprintf("%d min", conf.IdleMinutes))
	}
	idleContainer := container.NewVBox(
		widget.NewLabel("Délai d'extinction :"),
		selectWidget,
	)
	content := container.NewVBox(
		title,
		widget.NewSeparator(),
		batteryLabel,
		batteryBar,
		statusLabel,
		macLabel,
		idleContainer,
		activityLabel,
		widget.NewButton("Tester Animation Charge", func() {
			fmt.Println("Lancement de l'animation...")
		}),
	)

	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(300, 200))

	// 3. Boucle de mise à jour (Background)
	go func() {
		for {
			level, err := service.GetActualBatteryLevel()
			// On récupère le statut et la MAC
			status, errStatus := service.GetChargingStatus()
			macAddress := service.GetControllerMAC()

			if err != nil || errStatus != nil {
				// Si la manette n'est pas là, on met à jour l'UI pour informer l'utilisateur
				stateText.Set("État : Manette non trouvée")
				batteryText.Set("Batterie : --%")
				batteryValue.Set(0)
				macText.Set("MAC : --")

				// CRUCIAL : On attend avant la prochaine tentative pour libérer le CPU
				time.Sleep(5 * time.Second)
				continue
			}

			// Si tout va bien, on met à jour normalement
			macText.Set(fmt.Sprintf("MAC : %s", macAddress))
			batteryValue.Set(float64(level) / 100)
			batteryText.Set(fmt.Sprintf("Batterie : %d%%", level))
			stateText.Set(fmt.Sprintf("État : %s", status))

			if level <= 15 && status == "Discharging" {
				myApp.SendNotification(fyne.NewNotification(
					"Batterie Faible",
					fmt.Sprintf("Il reste %d%% sur votre DualSense.", level),
				))
			}

			// Pause standard entre deux lectures
			time.Sleep(2 * time.Second)
		}
	}()
	activityChan := make(chan time.Time)
	stopChan := make(chan bool)
	lastActivityTime := time.Now()

	// On lance le monitoring sur js1 (à adapter dynamiquement plus tard)
	go service.MonitorJoystick("/dev/input/js1", activityChan, stopChan)

	// Boucle de gestion du temps
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case t := <-activityChan:
				lastActivityTime = t
				lastActivityBinding.Set("En cours d'utilisation")

			case <-ticker.C:
				status, _ := stateText.Get()

				if strings.Contains(status, "non trouvée") || strings.Contains(status, "Recherche") {
					// Manette absente : on reset le temps et on affiche 0s
					lastActivityTime = time.Now()
					lastActivityBinding.Set("Inactif depuis : 0s (Déconnecté)")
					continue
				}
				diff := time.Since(lastActivityTime)

				currentChoice, _ := selectedDuration.Get()

				if currentChoice == "Jamais" {
					lastActivityBinding.Set(fmt.Sprintf("Inactif depuis : %ds (Auto-off désactivé)", int(diff.Seconds())))
					continue // On saute la vérification de déconnexion
				}

				// Extraction du nombre (ex: "10 min" -> 10)
				parts := strings.Split(currentChoice, " ")
				minutes, _ := strconv.Atoi(parts[0])
				limit := time.Duration(minutes) * time.Minute
				// Mise à jour UI : "Inactif depuis 12s"
				lastActivityBinding.Set(fmt.Sprintf("Inactif depuis : %s / %s", diff.Truncate(time.Second), currentChoice))

				// Seuil de déconnexion (ex: 10 minutes)
				if diff > limit {
					fmt.Println("Déconnexion automatique !")
					mac := service.GetControllerMAC() // La fonction qu'on a vue précédemment
					if mac != "" {
						err := service.DisconnectDualSenseNative()
						if err != nil {
							fmt.Println("Échec D-Bus:", err)
						}
					}
				}
			}
		}
	}()
	myWindow.ShowAndRun()
}
