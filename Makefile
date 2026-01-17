PREFIX ?= ~/.local

.PHONY: build install uninstall clean

build:
	go build -o xtrayhide ./cmd/xtrayhide

install: build
	install -Dm755 xtrayhide $(PREFIX)/bin/xtrayhide
	install -Dm644 xtrayhide.service ~/.config/systemd/user/xtrayhide.service

uninstall:
	rm -f $(PREFIX)/bin/xtrayhide
	rm -f ~/.config/systemd/user/xtrayhide.service

clean:
	rm -f xtrayhide
