package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/godbus/dbus/v5"
	"gopkg.in/yaml.v3"
)

type JSInterface struct {
	Time  uint32 // timestamp en ms
	Value int16  // valeur de l'axe ou bouton
	Type  uint8  // 1 = bouton, 2 = axe
	Index uint8  // numéro du bouton/axe
}

type Config struct {
	IdleMinutes  int    `yaml:"idle_minutes"`
	BatteryAlert int    `yaml:"battery_alert"`
	LastMAC      string `yaml:"last_mac"`
}

const (
	Deadzone       = 4000 // Seuil pour ignorer le drift
	InactivityTime = 10 * time.Minute
)

// Chemin vers la LED (à adapter selon ton numéro d'input)
const ledPath = "/sys/class/input/js1/device/device/leds/input68:rgb:indicator/brightness"
const multiPath = "/sys/class/input/js1/device/device/leds/input68:rgb:indicator/multi_intensity"
const jsPath = "/dev/input/js1"
const powerPath = "sys/devices/virtual/misc/uhid/0005:054C:0CE6*/power_supply/ps-controller-battery-*/capacity"

func setLED(value string) {
	err := os.WriteFile(ledPath, []byte(value), 0644)
	if err != nil {
		fmt.Println("Erreur écriture LED:", err)
	}
}
func setLEDColor(multiPath string, r, g, b int) {
	// Format attendu par le noyau : "R G B"
	colorStr := fmt.Sprintf("%d %d %d", r, g, b)
	err := os.WriteFile(multiPath, []byte(colorStr), 0644)
	if err != nil {
		fmt.Println("Erreur écriture couleur:", err)
	}
}

func chargeAnimation(multiPath string, stop chan bool) {
	for {
		select {
		case <-stop:
			return
		default:
			// Pulse du bleu (0 0 0 -> 0 0 255)
			for i := 0; i <= 255; i += 5 {
				setLEDColor(multiPath, 0, 0, i)
				time.Sleep(30 * time.Millisecond)
			}
			for i := 255; i >= 0; i -= 5 {
				setLEDColor(multiPath, 0, 0, i)
				time.Sleep(30 * time.Millisecond)
			}
		}
	}
}

func monitorJoystick(path string, activityChan chan time.Time, stopChan chan bool) {
	for {
		f, err := os.Open(path)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue

		}
		defer f.Close()

		// Structure d'un événement joystick Linux (8 octets)
		// Time (4) | Value (2) | Type (1) | Index (1)
		buffer := make([]byte, 8)
		deadzone := int16(2500) // Pour ignorer le stick drift

		for {
			_, err := f.Read(buffer)
			if err != nil {
				fmt.Println("Manette déconnectée, arrêt de la lecture.")
				f.Close()
				break
			}

			evType := buffer[6]
			evValue := int16(binary.LittleEndian.Uint16(buffer[4:6]))

			isReal := false
			if evType == 1 { // Bouton pressé
				isReal = true
			} else if evType == 2 { // Axe bougé
				// On vérifie si le mouvement dépasse la zone morte
				if evValue > deadzone || evValue < -deadzone {
					isReal = true
				}
			}

			if isReal {
				activityChan <- time.Now()
			}
		}
	}
}

func main() {
	conf := loadConfig()
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
		saveConfig(conf)
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
			level, err := getActualBatteryLevel()
			// On récupère le statut et la MAC
			status, errStatus := getChargingStatus()
			macAddress := getControllerMAC()

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
	go monitorJoystick("/dev/input/js1", activityChan, stopChan)

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
					mac := getControllerMAC() // La fonction qu'on a vue précédemment
					if mac != "" {
						err := disconnectDualSenseNative()
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

func getActualBatteryLevel() (int, error) {
	// On résout le wildcard (ex: /sys/class/power_supply/ps-controller-battery-*)
	// Note: /sys/class/power_supply est souvent plus simple que le chemin complet uhid
	matches, err := filepath.Glob("/sys/class/power_supply/ps-controller-battery-*/capacity")

	if err != nil || len(matches) == 0 {
		return 0, fmt.Errorf("Déconnecté")
	}

	// Lecture du fichier capacity
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return 0, fmt.Errorf("Erreur lecture")
	}

	// Conversion en entier
	levelStr := strings.TrimSpace(string(data))
	level, err := strconv.Atoi(levelStr)
	if err != nil {
		return 0, fmt.Errorf("Erreur format")
	}

	return level, nil
}

func getChargingStatus() (string, error) {

	matches, err := filepath.Glob("/sys/class/power_supply/ps-controller-battery-*/status")

	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("Déconnecté")
	}

	// Lecture du fichier capacity
	data, err := os.ReadFile(matches[0])
	if err != nil {
		return "", fmt.Errorf("Erreur lecture")
	}

	return string(data), nil

}
func getControllerMAC() string {
	// Le nom du dossier contient souvent l'adresse MAC avec des underscores
	matches, _ := filepath.Glob("/sys/class/power_supply/ps-controller-battery-*")
	if len(matches) > 0 {
		parts := strings.Split(matches[0], "-")
		// L'adresse MAC est souvent la dernière partie : 00_11_22_33_44_55
		macRaw := parts[len(parts)-1]
		return strings.ReplaceAll(macRaw, "_", ":")
	}
	return ""
}
func disconnectDualSenseNative() error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	// 1. On récupère tous les objets gérés par BlueZ
	var objects map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err = conn.Object("org.bluez", "/").Call("org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&objects)
	if err != nil {
		return fmt.Errorf("erreur ObjectManager: %v", err)
	}

	// 2. On parcourt les objets pour trouver une DualSense
	for path, interfaces := range objects {
		if props, ok := interfaces["org.bluez.Device1"]; ok {
			name, _ := props["Name"].Value().(string)

			// On cherche par nom (plus simple que la MAC au début)
			if strings.Contains(name, "Wireless Controller") || strings.Contains(name, "DualSense") {
				fmt.Printf("Tentative de déconnexion de : %s (%s)\n", name, path)

				// 3. Appel de la méthode Disconnect sur le chemin trouvé dynamiquement
				obj := conn.Object("org.bluez", path)
				call := obj.Call("org.bluez.Device1.Disconnect", 0)
				return call.Err
			}
		}
	}

	return fmt.Errorf("aucune DualSense trouvée sur le bus Bluetooth")
}
func getConfigPath() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "dualsense-manager")
	_ = os.MkdirAll(path, os.ModePerm)
	return filepath.Join(path, "config.yaml") // Extension .yaml
}

func saveConfig(conf Config) error {
	path := getConfigPath()

	data, err := yaml.Marshal(&conf)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func loadConfig() Config {
	path := getConfigPath()
	// Valeurs par défaut si le fichier n'existe pas
	conf := Config{
		IdleMinutes:  10,
		BatteryAlert: 15,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Si le fichier n'existe pas, on le crée avec les valeurs par défaut
		saveConfig(conf)
		return conf
	}

	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		return conf
	}
	return conf
}
