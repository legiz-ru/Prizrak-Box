//go:build linux

package services

import "github.com/godbus/dbus/v5"

// statusNotifierWatcher is the well-known bus name a desktop environment owns
// when it can host tray icons. Wails' Linux tray is a StatusNotifierItem
// (D-Bus/SNI) implementation, so without this name on the session bus the tray
// is never displayed anywhere.
const statusNotifierWatcher = "org.kde.StatusNotifierWatcher"

// TrayHostAvailable reports whether the session can actually display a tray
// icon.
//
// KDE, XFCE, Cinnamon and Budgie own the name; stock GNOME does NOT (it needs
// the AppIndicator extension). Wails reports no error in that case — the tray
// item is simply created and never shown — so "start minimized to tray" would
// otherwise leave the app running with no window and no way to reach it.
//
// A false answer is the safe direction: the caller reveals the window instead
// of hiding it. That also covers the benign race where an autostarted app beats
// its desktop environment's watcher onto the bus.
func TrayHostAvailable() bool {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return false
	}
	defer conn.Close()
	var owned bool
	if err := conn.BusObject().Call("org.freedesktop.DBus.NameHasOwner", 0, statusNotifierWatcher).Store(&owned); err != nil {
		return false
	}
	return owned
}
