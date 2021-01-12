darkman
=======

A framework for dark-mode and light-mode transitions on Linux desktop.

Introduction
------------

``darkman`` runs in the background and turns on night mode at sundown, and turns it off
again at sunrise. ``darkman`` is not designed to be used interactively: it's designed to
be set up once, and run in the background.

``darkman`` will use ``geoclue`` to determine your location and calculate sundown and
sunrise automatically.

At sundown, it will look for scripts in ``$XDG_DATA_DIRS/dark-mode.d/``.
At sunrise, it will look for scripts in ``$XDG_DATA_DIRS/light-mode.d/``.

These scripts individually configure different components and applications. Given the
lack of normalised "dark-mode" APIs on Linux desktop, it's likely that scripts for
different applications and toolkits have to be dumped in.

This project seeks to be a source for such scripts too.

Hint: ``$XDG_DATA_DIRS`` usually matches these, amongst others::

    ~/.local/share/
    /usr/local/share/
    /usr/share/

The variety here allows packages to include their own drop-in scripts. The order of
precedence is also important, so you can mask scripts.

Installation
------------

- ArchLinux: ``yay install darkman``.

Note: Installing via ``pip`` will not install the systemd service files.

Setup
-----

You can run the service any way you prefer. The recommended technique is using
systemd::

    systemctl --user enable --now darkman.service

Note that the dark-mode and light-mode scripts mentioned above are not included in this
package. You'll need to drop-in scripts you desire.

Development
-----------

``darkman`` already works, but is still under development.

For bug and suggestions, see Issues_ on GitLab. Research is also gathered into these
issues.

.. _Issues: https://gitlab.com/WhyNotHugo/darkman/-/issues


LICENCE
-------

darkman is licensed under the ISC licence. See LICENCE for details.
