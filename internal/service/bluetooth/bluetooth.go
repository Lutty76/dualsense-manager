// Package bluetooth provides functions to interact with Bluetooth features of DualSense controllers.
package bluetooth

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/godbus/dbus/v5"

	"dualsense/internal/sysfs"
)

// Bluetooth interface defines methods for Bluetooth operations.
type Bluetooth interface {
	ControllerMAC(path string) string
	DisconnectDualSenseNative(mac string) error
}

// ControllerMAC reads the controller MAC from sysfs for the given device path.
func ControllerMAC(path string) string {
	devName := filepath.Base(path)

	macPath := fmt.Sprintf("/sys/class/input/%s/device/uniq", devName)

	data, err := sysfs.FS.ReadFile(macPath)
	if err != nil {
		macPath = fmt.Sprintf("/sys/class/input/%s/device/address", devName)
		data, err = sysfs.FS.ReadFile(macPath)
	}

	if err != nil {
		return ""
	}

	mac := strings.TrimSpace(string(data))
	return strings.ToUpper(mac)
}

// DisconnectDualSenseNative requests BlueZ to disconnect the device with the given MAC.
func DisconnectDualSenseNative(mac string) error {
	conn, err := ConnectSystemBus()
	if err != nil {
		return err
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			log.Default().Println("Error closing D-Bus connection:", err)
		}

	}()
	dbusPath := "/org/bluez/hci0/dev_" + strings.ReplaceAll(mac, ":", "_")
	obj := conn.Object("org.bluez", dbus.ObjectPath(dbusPath))
	call := obj.Call("org.bluez.Device1.Disconnect", 0)
	return call.Err

}

// ConnectSystemBus is a hook for tests to override D-Bus connection behavior.
var ConnectSystemBus = dbus.ConnectSystemBus
