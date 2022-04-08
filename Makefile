DESTDIR?=/
PREFIX=/usr

darkman.1: darkman.1.scd
	scdoc < darkman.1.scd > darkman.1

index.html: darkman.1
	mandoc -T html -O style=man-style.css < darkman.1 > index.html

site.tar.gz: index.html
	tar -cvz index.html man-style.css > site.tar.gz

darkman:
	go build -o darkman ./cmd

.PHONY: build
build: darkman darkman.1
	./darkman completion zsh > _darkman.zsh
	./darkman completion bash > darkman.bash

.PHONY: install
install:
	@install -Dm755 darkman 	${DESTDIR}${PREFIX}/bin/darkman
	@ln -s darkman                   ${DESTDIR}${PREFIX}/bin/darkmanctl
	@install -Dm644 darkman.service	${DESTDIR}${PREFIX}/lib/systemd/user/darkman.service
	@install -Dm644 darkman.1	${DESTDIR}${PREFIX}/share/man/man1/darkman.1
	@install -Dm644 LICENCE 	${DESTDIR}${PREFIX}/share/licenses/darkman/LICENCE
	@install -Dm644 _darkman.zsh ${DESTDIR}${PREFIX}/share/zsh/site-functions/_darkman
	@install -Dm644 darkman.bash ${DESTDIR}${PREFIX}/share/bash-completion/completions/darkman
	@install -Dm644 contrib/dbus/nl.whynothugo.darkman.service \
		${DESTDIR}${PREFIX}/share/dbus-1/services/nl.whynothugo.darkman.service
	@install -Dm644 contrib/dbus/org.freedesktop.impl.portal.desktop.darkman.service \
		${DESTDIR}${PREFIX}/share/dbus-1/services/org.freedesktop.impl.portal.desktop.darkman.service
	@install -Dm644 contrib/portal/darkman.portal \
		${DESTDIR}${PREFIX}/share/xdg-desktop-portal/portals/darkman.portal

.PHONY: aur
aur:
	git subtree push -P contrib/aur ssh://aur@aur.archlinux.org/darkman.git master
