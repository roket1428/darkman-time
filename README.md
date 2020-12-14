darkman
=======

A framework for dark-mode and light-mode transitions on Linux desktop.

Introduction
------------

`darkman` run in the background and turns on night mode at sundown, and turns it off
again at sunrise. `darkman` is not designed to be used interactively: it's designed to
be set up once, and run in the background.

`darkman` will use `geoclue` to determine your location and calculate sundown and
sunrise automatically.

At sundown, it will look for scripts in `$XDG_DATA_DIRS/dark-mode.d/`.
At sunrise, it will look for scripts in `$XDG_DATA_DIRS/light-mode.d/`.

These scripts individually configure different components and applications. Given the
lack of normalised "dark-mode" APIs on Linux desktop, it's likely that scripts for
different applications and toolkits have to be dumped in.

This project seeks to be a source for such scripts too.

Hint: `$XDG_DATA_DIRS` usually matches these, amongst others:

    ~/.local/share/
    /usr/local/share/
    /usr/share/

The variety here allows packages to include their own drop-in scripts. The order of
precedence is also important, so you can mask scripts.
