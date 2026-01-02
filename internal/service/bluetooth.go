package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/godbus/dbus/v5"
)

func ControllerMAC(path string) string {
	devName := filepath.Base(path)

	macPath := fmt.Sprintf("/sys/class/input/%s/device/uniq", devName)

	data, err := os.ReadFile(macPath)
	if err != nil {
		macPath = fmt.Sprintf("/sys/class/input/%s/device/address", devName)
		data, err = os.ReadFile(macPath)
	}

	if err != nil {
		return ""
	}

	mac := strings.TrimSpace(string(data))
	return strings.ToUpper(mac)
}
func DisconnectDualSenseNative(mac string) error {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbusPath := "/org/bluez/hci0/dev_" + strings.ReplaceAll(mac, ":", "_")
	obj := conn.Object("org.bluez", dbus.ObjectPath(dbusPath))
	call := obj.Call("org.bluez.Device1.Disconnect", 0)
	return call.Err

}
