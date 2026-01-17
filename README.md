# xtrayhide

Captures X11 system tray icons (XEmbed) and hides them, exposing them as modern StatusNotifierItem (SNI) icons instead.

Useful on Wayland compositors (like niri) where XWayland tray windows appear visible and always-on-top.

## Install

```sh
go install github.com/bnema/xtrayhide/cmd/xtrayhide@latest
```

Or build from source:

```sh
git clone https://github.com/bnema/xtrayhide
cd xtrayhide
make install
```

Then enable the systemd service:

```sh
systemctl --user enable --now xtrayhide
```

## Usage

Just run `xtrayhide`. It will:

1. Become the X11 system tray owner
2. Capture any docking tray icons
3. Hide the X11 windows (keep them unmapped)
4. Expose each icon as an SNI on D-Bus
5. Forward clicks from SNI back to the hidden X11 window

## License

MIT
