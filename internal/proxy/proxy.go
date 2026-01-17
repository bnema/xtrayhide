package proxy

import (
	"hash/fnv"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"

	"github.com/bnema/xtrayhide/internal/sni"
	"github.com/bnema/xtrayhide/internal/tray"
)

const (
	pollInterval = 300 * time.Millisecond
)

type Proxy struct {
	conn *xgb.Conn
	root xproto.Window
	icon *tray.Icon
	item *sni.Item
	done chan struct{}
}

func New(conn *xgb.Conn, root xproto.Window, icon *tray.Icon, item *sni.Item) *Proxy {
	p := &Proxy{
		conn: conn,
		root: root,
		icon: icon,
		item: item,
		done: make(chan struct{}),
	}
	item.SetHandler(p)
	go p.pollIcon()
	return p
}

func (p *Proxy) Close() {
	close(p.done)
	p.item.Close()
}

func (p *Proxy) Activate(x, y int32) {
	p.sendClick(uint8(1), x, y)
}

func (p *Proxy) SecondaryActivate(x, y int32) {
	p.sendClick(uint8(2), x, y)
}

func (p *Proxy) ContextMenu(x, y int32) {
	p.sendClick(uint8(3), x, y)
}

func (p *Proxy) Scroll(delta int32, orientation string) {
	button := uint8(4)
	if orientation == "horizontal" {
		if delta < 0 {
			button = 6
		} else {
			button = 7
		}
	} else {
		if delta < 0 {
			button = 5
		}
	}
	p.sendClick(button, 0, 0)
}

func (p *Proxy) sendClick(button uint8, x, y int32) {
	// Temporarily map the window to ensure the application can process events.
	p.icon.Map()
	defer p.icon.Unmap()

	eventX, eventY := int16(0), int16(0)
	if geom, err := xproto.GetGeometry(p.conn, xproto.Drawable(p.icon.Window)).Reply(); err == nil {
		eventX = int16(geom.Width / 2)
		eventY = int16(geom.Height / 2)
	}

	press := xproto.ButtonPressEvent{
		Detail:     xproto.Button(button),
		Time:       xproto.TimeCurrentTime,
		Root:       p.root,
		Event:      p.icon.Window,
		Child:      0,
		RootX:      int16(x),
		RootY:      int16(y),
		EventX:     eventX,
		EventY:     eventY,
		SameScreen: true,
	}
	release := xproto.ButtonReleaseEvent(press)

	xproto.SendEvent(p.conn, false, p.icon.Window, xproto.EventMaskButtonPress, string(press.Bytes()))
	xproto.SendEvent(p.conn, false, p.icon.Window, xproto.EventMaskButtonRelease, string(release.Bytes()))
	// jezek/xgb flushes requests asynchronously; Sync forces them out.
	p.conn.Sync()
}

func (p *Proxy) pollIcon() {
	var lastHash uint32
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			width, height, data, err := p.icon.Capture()
			if err != nil || len(data) == 0 {
				continue
			}
			h := hashBytes(data)
			if h == lastHash {
				continue
			}
			lastHash = h
			pixmap := sni.Pixmap{Width: int32(width), Height: int32(height), Data: data}
			p.item.UpdateIcon([]sni.Pixmap{pixmap})
		}
	}
}

func hashBytes(data []byte) uint32 {
	h := fnv.New32a()
	_, _ = h.Write(data)
	return h.Sum32()
}
