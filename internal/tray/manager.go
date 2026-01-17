package tray

import (
	"context"
	"fmt"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const (
	systemTrayRequestDock = 0
)

type Manager struct {
	Conn        *xgb.Conn
	Root        xproto.Window
	RootVisual  xproto.Visualid
	Atoms       Atoms
	managerWin  xproto.Window
	IconAdded   chan *Icon
	IconRemoved chan *Icon
	icons       map[xproto.Window]*Icon
}

func NewManager() (*Manager, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("connect X11: %w", err)
	}

	atoms, err := InternAtoms(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	setup := xproto.Setup(conn)
	screen := setup.DefaultScreen(conn)
	root := screen.Root
	rootVisual := screen.RootVisual

	ownerReply, err := xproto.GetSelectionOwner(conn, atoms.TraySelection).Reply()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("get selection owner: %w", err)
	}
	if ownerReply.Owner != xproto.WindowNone {
		conn.Close()
		return nil, fmt.Errorf("system tray already owned by window %d", ownerReply.Owner)
	}

	managerWin, err := xproto.NewWindowId(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("new window id: %w", err)
	}

	err = xproto.CreateWindowChecked(
		conn,
		0,
		managerWin,
		root,
		0, 0, 1, 1,
		0,
		xproto.WindowClassInputOnly,
		rootVisual,
		xproto.CwEventMask,
		[]uint32{xproto.EventMaskStructureNotify | xproto.EventMaskPropertyChange},
	).Check()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("create manager window: %w", err)
	}

	if err := xproto.SetSelectionOwnerChecked(conn, managerWin, atoms.TraySelection, xproto.TimeCurrentTime).Check(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("set selection owner: %w", err)
	}

	if err := broadcastManager(conn, root, atoms.Manager, atoms.TraySelection, managerWin); err != nil {
		conn.Close()
		return nil, err
	}

	m := &Manager{
		Conn:        conn,
		Root:        root,
		RootVisual:  rootVisual,
		Atoms:       atoms,
		managerWin:  managerWin,
		IconAdded:   make(chan *Icon, 16),
		IconRemoved: make(chan *Icon, 16),
		icons:       make(map[xproto.Window]*Icon),
	}

	return m, nil
}

func (m *Manager) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		m.Conn.Close()
	}()

	for {
		ev, err := m.Conn.WaitForEvent()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return fmt.Errorf("wait for event: %w", err)
		}
		if ev == nil {
			continue
		}
		switch e := ev.(type) {
		case xproto.ClientMessageEvent:
			m.handleClientMessage(e)
		case xproto.DestroyNotifyEvent:
			m.handleDestroy(e)
		}
	}
}

func (m *Manager) handleClientMessage(ev xproto.ClientMessageEvent) {
	if ev.Type != m.Atoms.TrayOpcode {
		return
	}
	data := ev.Data.Data32
	if len(data) < 3 {
		return
	}
	if data[1] != systemTrayRequestDock {
		return
	}
	iconWin := xproto.Window(data[2])
	icon, err := m.embedIcon(iconWin)
	if err != nil {
		return
	}
	m.icons[iconWin] = icon
	m.IconAdded <- icon
}

func (m *Manager) handleDestroy(ev xproto.DestroyNotifyEvent) {
	icon, ok := m.icons[ev.Window]
	if !ok {
		return
	}
	delete(m.icons, ev.Window)
	m.IconRemoved <- icon
}

func (m *Manager) embedIcon(iconWin xproto.Window) (*Icon, error) {
	container, err := xproto.NewWindowId(m.Conn)
	if err != nil {
		return nil, fmt.Errorf("new container id: %w", err)
	}

	// Tray icons sometimes report (or start with) very large window geometries.
	// Keep an explicit, small slot size and force the icon window to it.
	width := uint16(32)
	height := uint16(32)

	err = xproto.CreateWindowChecked(
		m.Conn,
		0,
		container,
		m.Root,
		-10000, -10000, width, height,
		0,
		xproto.WindowClassInputOutput,
		m.RootVisual,
		xproto.CwEventMask,
		[]uint32{xproto.EventMaskStructureNotify | xproto.EventMaskExposure | xproto.EventMaskPropertyChange},
	).Check()
	if err != nil {
		return nil, fmt.Errorf("create container: %w", err)
	}

	// Don't let the WM manage/decorate our offscreen host window.
	if err := xproto.ChangeWindowAttributesChecked(m.Conn, container, xproto.CwOverrideRedirect, []uint32{1}).Check(); err != nil {
		return nil, fmt.Errorf("set override redirect: %w", err)
	}

	if err := xproto.ReparentWindowChecked(m.Conn, iconWin, container, 0, 0).Check(); err != nil {
		return nil, fmt.Errorf("reparent icon: %w", err)
	}

	// Force a sane size for capture and to avoid huge toplevel windows.
	if err := xproto.ConfigureWindowChecked(m.Conn, iconWin, xproto.ConfigWindowWidth|xproto.ConfigWindowHeight, []uint32{uint32(width), uint32(height)}).Check(); err != nil {
		return nil, fmt.Errorf("resize icon: %w", err)
	}

	if err := xproto.ChangeWindowAttributesChecked(m.Conn, iconWin, xproto.CwEventMask, []uint32{xproto.EventMaskStructureNotify | xproto.EventMaskPropertyChange}).Check(); err != nil {
		return nil, fmt.Errorf("select icon events: %w", err)
	}

	if err := xproto.ChangeSaveSetChecked(m.Conn, xproto.SetModeInsert, iconWin).Check(); err != nil {
		return nil, fmt.Errorf("change save set: %w", err)
	}

	// Do NOT map the windows - keep them hidden from Wayland/XWayland.
	// The icon will be temporarily mapped only when capturing the pixmap.

	icon := &Icon{
		conn:      m.Conn,
		atoms:     m.Atoms,
		Window:    iconWin,
		Container: container,
	}
	icon.setXEmbedInfo()
	icon.sendXEmbedNotify()

	return icon, nil
}

func broadcastManager(conn *xgb.Conn, root xproto.Window, managerAtom xproto.Atom, trayAtom xproto.Atom, managerWin xproto.Window) error {
	ev := xproto.ClientMessageEvent{
		Format: 32,
		Window: root,
		Type:   managerAtom,
		Data: xproto.ClientMessageDataUnionData32New([]uint32{
			uint32(xproto.TimeCurrentTime),
			uint32(trayAtom),
			uint32(managerWin),
			0,
			0,
		}),
	}
	return xproto.SendEventChecked(conn, false, root, xproto.EventMaskStructureNotify, string(ev.Bytes())).Check()
}
