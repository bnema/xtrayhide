package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/godbus/dbus/v5"
	"github.com/jezek/xgb/xproto"

	"github.com/bnema/xtrayhide/internal/proxy"
	"github.com/bnema/xtrayhide/internal/sni"
	"github.com/bnema/xtrayhide/internal/tray"
)

type iconEntry struct {
	proxy *proxy.Proxy
	item  *sni.Item
}

func main() {
	log.SetFlags(log.Ltime)
	log.Printf("xtrayhide starting - capturing and hiding X11 tray icons")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	manager, err := tray.NewManager()
	if err != nil {
		log.Fatalf("tray manager: %v", err)
	}
	defer manager.Conn.Close()
	log.Printf("acquired system tray selection, waiting for icons...")

	bus, err := dbus.ConnectSessionBus()
	if err != nil {
		log.Fatalf("dbus session bus: %v", err)
	}
	defer bus.Close()

	icons := make(map[xproto.Window]*iconEntry)
	counter := 0

	go func() {
		if err := manager.Run(ctx); err != nil && ctx.Err() == nil {
			log.Printf("manager stopped: %v", err)
			stop()
		}
	}()

	for {
		select {
		case icon := <-manager.IconAdded:
			counter++
			title := icon.Title()
			log.Printf("icon docked: %q (window 0x%x)", title, icon.Window)

			service := fmt.Sprintf("org.kde.StatusNotifierItem-%d-%d", os.Getpid(), counter)
			pixmap := []sni.Pixmap{}
			if width, height, data, err := icon.Capture(); err == nil && len(data) > 0 {
				pixmap = []sni.Pixmap{{Width: int32(width), Height: int32(height), Data: data}}
				log.Printf("captured icon: %q (%dx%d)", title, width, height)
			}

			props := sni.Properties{
				Category:   "ApplicationStatus",
				ID:         fmt.Sprintf("xtrayhide-%d", icon.Window),
				Title:      title,
				Status:     "Active",
				WindowID:   uint32(icon.Window),
				IconPixmap: pixmap,
				ItemIsMenu: false,
			}

			item, err := sni.NewItem(bus, service, props, nil)
			if err != nil {
				log.Printf("create SNI item: %v", err)
				continue
			}
			p := proxy.New(manager.Conn, manager.Root, icon, item)
			icons[icon.Window] = &iconEntry{proxy: p, item: item}
			log.Printf("registered SNI: %q -> %s", title, service)

		case icon := <-manager.IconRemoved:
			entry, ok := icons[icon.Window]
			if ok {
				log.Printf("icon removed: window 0x%x", icon.Window)
				entry.proxy.Close()
				delete(icons, icon.Window)
			}

		case <-ctx.Done():
			log.Printf("shutting down, releasing %d icons", len(icons))
			for _, entry := range icons {
				entry.proxy.Close()
			}
			return
		}
	}
}
