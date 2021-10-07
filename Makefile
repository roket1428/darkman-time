DESTDIR?=/
PREFIX=/usr

build:
	go build -ldflags '-s'
	scdoc < darkman.1.scd > darkman.1

install:
	@install -Dm755 darkman 	${DESTDIR}${PREFIX}/bin/darkman
	@install -Dm644 darkman.service	${DESTDIR}${PREFIX}/lib/systemd/user/darkman.service
	@install -Dm644 darkman.1	${DESTDIR}${PREFIX}/share/man/man1/darkman.1
	@install -Dm644 LICENCE 	${DESTDIR}${PREFIX}/share/licenses/${pkgname}/LICENCE

aur:
	git subtree push -P contrib/aur ssh://aur@aur.archlinux.org/darkman.git master

.PHONY: build install aur
