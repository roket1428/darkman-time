DESTDIR?=/
PREFIX=/usr
VERSION?=`git describe --tags --dirty 2>/dev/null || echo 0.0.0-dev`

.PHONY: build completion install
build: darkman darkman.1 completion

darkman.1: darkman.1.scd
	scdoc < darkman.1.scd > darkman.1

darkman:
	go build -ldflags "-X main.Version=$(VERSION)" ./cmd/darkman

_darkman.zsh: darkman
	./darkman completion zsh > _darkman.zsh

darkman.bash: darkman
	./darkman completion bash > darkman.bash

darkman.fish: darkman
	./darkman completion fish > darkman.fish

completion: _darkman.zsh darkman.bash darkman.fish

install: build
	@install -Dm755 darkman 	${DESTDIR}${PREFIX}/bin/darkman
	@ln -s darkman                   ${DESTDIR}${PREFIX}/bin/darkmanctl
	@install -Dm644 darkman.service	${DESTDIR}${PREFIX}/lib/systemd/user/darkman.service
	@install -Dm644 darkman.1	${DESTDIR}${PREFIX}/share/man/man1/darkman.1
	@install -Dm644 LICENCE 	${DESTDIR}${PREFIX}/share/licenses/darkman/LICENCE
	@install -Dm644 _darkman.zsh ${DESTDIR}${PREFIX}/share/zsh/site-functions/_darkman
	@install -Dm644 darkman.bash ${DESTDIR}${PREFIX}/share/bash-completion/completions/darkman
	@install -Dm644 darkman.fish ${DESTDIR}${PREFIX}/share/fish/vendor_completions.d/darkman.fish
	@install -Dm644 contrib/dbus/nl.whynothugo.darkman.service \
		${DESTDIR}${PREFIX}/share/dbus-1/services/nl.whynothugo.darkman.service
	@install -Dm644 contrib/dbus/org.freedesktop.impl.portal.desktop.darkman.service \
		${DESTDIR}${PREFIX}/share/dbus-1/services/org.freedesktop.impl.portal.desktop.darkman.service
	@install -Dm644 contrib/portal/darkman.portal \
		${DESTDIR}${PREFIX}/share/xdg-desktop-portal/portals/darkman.portal
	@install -Dm644 darkman.desktop \
		-t ${DESTDIR}${PREFIX}/share/applications/
