package sni

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

func Register(conn *dbus.Conn, service string) error {
	if conn == nil {
		return fmt.Errorf("dbus connection is nil")
	}
	// StatusNotifierWatcher lives at a well-known name and object path.
	obj := conn.Object("org.kde.StatusNotifierWatcher", dbus.ObjectPath("/StatusNotifierWatcher"))
	call := obj.Call("org.kde.StatusNotifierWatcher.RegisterStatusNotifierItem", 0, service)
	if call.Err != nil {
		return fmt.Errorf("register with watcher: %w", call.Err)
	}
	return nil
}
