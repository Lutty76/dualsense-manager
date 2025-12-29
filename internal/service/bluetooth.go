package service

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/godbus/dbus/v5"
)

func GetControllerMAC() string {
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
func DisconnectDualSenseNative() error {
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
