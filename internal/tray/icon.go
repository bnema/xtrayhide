package tray

import (
	"fmt"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
)

const (
	xembedEmbeddedNotify = 0
	xembedVersion        = 0
	xembedMapped         = 1
)

type Icon struct {
	conn      *xgb.Conn
	atoms     Atoms
	Window    xproto.Window
	Container xproto.Window
	mapped    bool
}

// Map makes the icon window visible (needed before capture).
func (i *Icon) Map() {
	if i.mapped {
		return
	}
	xproto.MapWindow(i.conn, i.Container)
	xproto.MapWindow(i.conn, i.Window)
	i.conn.Sync()
	i.mapped = true
}

// Unmap hides the icon window from display.
func (i *Icon) Unmap() {
	if !i.mapped {
		return
	}
	xproto.UnmapWindow(i.conn, i.Window)
	xproto.UnmapWindow(i.conn, i.Container)
	i.conn.Sync()
	i.mapped = false
}

func (i *Icon) Capture() (width uint16, height uint16, data []byte, err error) {
	// Temporarily map the window to capture its contents.
	wasUnmapped := !i.mapped
	if wasUnmapped {
		i.Map()
	}

	geom, err := xproto.GetGeometry(i.conn, xproto.Drawable(i.Window)).Reply()
	if err != nil {
		if wasUnmapped {
			i.Unmap()
		}
		return 0, 0, nil, fmt.Errorf("get geometry: %w", err)
	}
	width = geom.Width
	height = geom.Height

	img, err := xproto.GetImage(i.conn, xproto.ImageFormatZPixmap, xproto.Drawable(i.Window), 0, 0, width, height, 0xffffffff).Reply()
	if err != nil {
		if wasUnmapped {
			i.Unmap()
		}
		return 0, 0, nil, fmt.Errorf("get image: %w", err)
	}

	// Unmap immediately after capture to keep it hidden.
	if wasUnmapped {
		i.Unmap()
	}

	return width, height, img.Data, nil
}

func (i *Icon) Title() string {
	if title, err := getUTF8Property(i.conn, i.Window, i.atoms.NetWMName, i.atoms.UTF8String); err == nil && title != "" {
		return title
	}
	if title, err := getStringProperty(i.conn, i.Window, i.atoms.WMName); err == nil && title != "" {
		return title
	}
	return fmt.Sprintf("xembed-%d", i.Window)
}

func (i *Icon) sendXEmbedNotify() {
	data := xproto.ClientMessageDataUnionData32New([]uint32{
		uint32(xproto.TimeCurrentTime),
		xembedEmbeddedNotify,
		0,
		uint32(i.Container),
		xembedVersion,
	})
	ev := xproto.ClientMessageEvent{
		Format: 32,
		Window: i.Window,
		Type:   i.atoms.XEmbed,
		Data:   data,
	}
	xproto.SendEvent(i.conn, false, i.Window, xproto.EventMaskNoEvent, string(ev.Bytes()))
}

func (i *Icon) setXEmbedInfo() {
	values := []uint32{xembedVersion, xembedMapped}
	data := make([]byte, len(values)*4)
	for idx, value := range values {
		xgb.Put32(data[idx*4:], value)
	}
	xproto.ChangeProperty(i.conn, xproto.PropModeReplace, i.Window, i.atoms.XEmbedInfo, i.atoms.XEmbedInfo, 32, uint32(len(values)), data)
}

func getUTF8Property(conn *xgb.Conn, win xproto.Window, atom xproto.Atom, utf8Atom xproto.Atom) (string, error) {
	reply, err := xproto.GetProperty(conn, false, win, atom, utf8Atom, 0, (1<<32)-1).Reply()
	if err != nil {
		return "", err
	}
	if reply == nil || len(reply.Value) == 0 {
		return "", nil
	}
	return string(reply.Value), nil
}

func getStringProperty(conn *xgb.Conn, win xproto.Window, atom xproto.Atom) (string, error) {
	reply, err := xproto.GetProperty(conn, false, win, atom, xproto.AtomString, 0, (1<<32)-1).Reply()
	if err != nil {
		return "", err
	}
	if reply == nil || len(reply.Value) == 0 {
		return "", nil
	}
	return string(reply.Value), nil
}
