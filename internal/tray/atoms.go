package tray

import (
	"fmt"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

type Atoms struct {
	TraySelection xproto.Atom
	TrayOpcode    xproto.Atom
	Manager       xproto.Atom
	XEmbed        xproto.Atom
	XEmbedInfo    xproto.Atom
	WMName        xproto.Atom
	NetWMName     xproto.Atom
	UTF8String    xproto.Atom
	NetWMIcon     xproto.Atom
}

func internAtom(conn *xgb.Conn, name string) (xproto.Atom, error) {
	reply, err := xproto.InternAtom(conn, true, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, fmt.Errorf("intern atom %s: %w", name, err)
	}
	return reply.Atom, nil
}

func InternAtoms(conn *xgb.Conn) (Atoms, error) {
	traySelection, err := internAtom(conn, "_NET_SYSTEM_TRAY_S0")
	if err != nil {
		return Atoms{}, err
	}
	trayOpcode, err := internAtom(conn, "_NET_SYSTEM_TRAY_OPCODE")
	if err != nil {
		return Atoms{}, err
	}
	manager, err := internAtom(conn, "MANAGER")
	if err != nil {
		return Atoms{}, err
	}
	xembed, err := internAtom(conn, "_XEMBED")
	if err != nil {
		return Atoms{}, err
	}
	xembedInfo, err := internAtom(conn, "_XEMBED_INFO")
	if err != nil {
		return Atoms{}, err
	}
	wmName, err := internAtom(conn, "WM_NAME")
	if err != nil {
		return Atoms{}, err
	}
	netWMName, err := internAtom(conn, "_NET_WM_NAME")
	if err != nil {
		return Atoms{}, err
	}
	utf8String, err := internAtom(conn, "UTF8_STRING")
	if err != nil {
		return Atoms{}, err
	}
	netWMIcon, err := internAtom(conn, "_NET_WM_ICON")
	if err != nil {
		return Atoms{}, err
	}

	return Atoms{
		TraySelection: traySelection,
		TrayOpcode:    trayOpcode,
		Manager:       manager,
		XEmbed:        xembed,
		XEmbedInfo:    xembedInfo,
		WMName:        wmName,
		NetWMName:     netWMName,
		UTF8String:    utf8String,
		NetWMIcon:     netWMIcon,
	}, nil
}
