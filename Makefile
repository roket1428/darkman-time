DESTDIR?=/
PREFIX=/usr

build:
	go build
	go build -o darkmanctl ./ctl
	scdoc < darkman.1.scd > darkman.1
	./darkmanctl completion zsh > _darkmanctl.zsh
	./darkmanctl completion bash > darkmanctl.bash

install:
	@install -Dm755 darkman 	${DESTDIR}${PREFIX}/bin/darkman
	@install -Dm755 darkmanctl 	${DESTDIR}${PREFIX}/bin/darkmanctl
	@install -Dm644 darkman.service	${DESTDIR}${PREFIX}/lib/systemd/user/darkman.service
	@install -Dm644 darkman.1	${DESTDIR}${PREFIX}/share/man/man1/darkman.1
	@install -Dm644 LICENCE 	${DESTDIR}${PREFIX}/share/licenses/darkman/LICENCE
	@install -Dm644 _darkmanctl.zsh ${DESTDIR}${PREFIX}/share/zsh/site-functions/_darkmanctl
	@install -Dm644 darkmanctl.bash ${DESTDIR}${PREFIX}/share/bash-completion/completions/darkmanctl

aur:
	git subtree push -P contrib/aur ssh://aur@aur.archlinux.org/darkman.git master

.PHONY: build install aur
