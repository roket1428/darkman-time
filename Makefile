DESTDIR?=/
PREFIX=/usr

build:
	go build -ldflags '-s'

install:
	@install -Dm755 darkman 	${DESTDIR}${PREFIX}/bin/darkman
	@install -Dm644 darkman.service	${DESTDIR}${PREFIX}/lib/systemd/user/darkman.service

aur:
	git subtree push -P contrib/aur ssh://aur@aur.archlinux.org/darkman.git master

.PHONY: build install aur