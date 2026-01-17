package sni

import (
	"fmt"
	"sync"

	"github.com/godbus/dbus/v5"
)

type Pixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

type Properties struct {
	Category   string
	ID         string
	Title      string
	Status     string
	WindowID   uint32
	IconPixmap []Pixmap
	ItemIsMenu bool
}

type ActionHandler interface {
	Activate(x, y int32)
	SecondaryActivate(x, y int32)
	ContextMenu(x, y int32)
	Scroll(delta int32, orientation string)
}

type Item struct {
	conn    *dbus.Conn
	path    dbus.ObjectPath
	service string
	handler ActionHandler
	mu      sync.RWMutex
	props   Properties
}

func NewItem(conn *dbus.Conn, service string, props Properties, handler ActionHandler) (*Item, error) {
	if conn == nil {
		return nil, fmt.Errorf("dbus connection is nil")
	}
	reply, err := conn.RequestName(service, dbus.NameFlagDoNotQueue)
	if err != nil {
		return nil, fmt.Errorf("request name: %w", err)
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		return nil, fmt.Errorf("dbus name not available: %s", service)
	}

	item := &Item{
		conn:    conn,
		path:    dbus.ObjectPath("/StatusNotifierItem"),
		service: service,
		props:   props,
		handler: handler,
	}

	conn.Export(item, item.path, "org.kde.StatusNotifierItem")
	conn.Export(item, item.path, "org.freedesktop.DBus.Properties")
	conn.Export(item, item.path, "org.freedesktop.DBus.Introspectable")

	if err := Register(conn, service); err != nil {
		return nil, err
	}

	return item, nil
}

func (i *Item) Close() {
	if i.conn == nil {
		return
	}
	i.conn.ReleaseName(i.service)
}

func (i *Item) SetHandler(handler ActionHandler) {
	i.mu.Lock()
	i.handler = handler
	i.mu.Unlock()
}

func (i *Item) UpdateIcon(pixmaps []Pixmap) {
	i.mu.Lock()
	i.props.IconPixmap = pixmaps
	i.mu.Unlock()
	i.conn.Emit(i.path, "org.kde.StatusNotifierItem", "NewIcon")
}

func (i *Item) UpdateTitle(title string) {
	i.mu.Lock()
	i.props.Title = title
	i.mu.Unlock()
	i.conn.Emit(i.path, "org.kde.StatusNotifierItem", "NewTitle")
}

func (i *Item) Activate(x, y int32) *dbus.Error {
	i.mu.RLock()
	h := i.handler
	i.mu.RUnlock()
	if h != nil {
		h.Activate(x, y)
	}
	return nil
}

func (i *Item) SecondaryActivate(x, y int32) *dbus.Error {
	i.mu.RLock()
	h := i.handler
	i.mu.RUnlock()
	if h != nil {
		h.SecondaryActivate(x, y)
	}
	return nil
}

func (i *Item) ContextMenu(x, y int32) *dbus.Error {
	i.mu.RLock()
	h := i.handler
	i.mu.RUnlock()
	if h != nil {
		h.ContextMenu(x, y)
	}
	return nil
}

func (i *Item) Scroll(delta int32, orientation string) *dbus.Error {
	i.mu.RLock()
	h := i.handler
	i.mu.RUnlock()
	if h != nil {
		h.Scroll(delta, orientation)
	}
	return nil
}

func (i *Item) Get(iface, prop string) (dbus.Variant, *dbus.Error) {
	if iface != "org.kde.StatusNotifierItem" {
		return dbus.Variant{}, dbus.MakeFailedError(fmt.Errorf("unknown interface: %s", iface))
	}
	return dbus.MakeVariant(i.getProperty(prop)), nil
}

func (i *Item) Set(iface, prop string, value dbus.Variant) *dbus.Error {
	return dbus.MakeFailedError(fmt.Errorf("property %s is read-only", prop))
}

func (i *Item) GetAll(iface string) (map[string]dbus.Variant, *dbus.Error) {
	if iface != "org.kde.StatusNotifierItem" {
		return nil, dbus.MakeFailedError(fmt.Errorf("unknown interface: %s", iface))
	}
	i.mu.RLock()
	defer i.mu.RUnlock()
	return map[string]dbus.Variant{
		"Category":   dbus.MakeVariant(i.props.Category),
		"Id":         dbus.MakeVariant(i.props.ID),
		"Title":      dbus.MakeVariant(i.props.Title),
		"Status":     dbus.MakeVariant(i.props.Status),
		"WindowId":   dbus.MakeVariant(i.props.WindowID),
		"IconPixmap": dbus.MakeVariant(i.props.IconPixmap),
		"ItemIsMenu": dbus.MakeVariant(i.props.ItemIsMenu),
	}, nil
}

func (i *Item) Introspect() (string, *dbus.Error) {
	return introspectionXML, nil
}

func (i *Item) getProperty(prop string) interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()
	switch prop {
	case "Category":
		return i.props.Category
	case "Id":
		return i.props.ID
	case "Title":
		return i.props.Title
	case "Status":
		return i.props.Status
	case "WindowId":
		return i.props.WindowID
	case "IconPixmap":
		return i.props.IconPixmap
	case "ItemIsMenu":
		return i.props.ItemIsMenu
	default:
		return nil
	}
}
