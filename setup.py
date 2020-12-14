#!/usr/bin/env python3
from setuptools import setup

setup(
    name="darkman",
    description="A framework for dark-mode and light-mode transitions on Linux desktop.",
    author="Hugo Osvaldo Barrera",
    author_email="hugo@barrera.io",
    url="https://gitlab.com/whynothugo/darkman",
    license="ISC",
    packages=["darkman"],
    include_package_data=True,
    entry_points={"console_scripts": ["darkman = darkman:run"]},
    install_requires=[
        "astral",
        "python-dateutil",
        "pyxdg",
        "txdbus",
    ],
    long_description=open("README.rst").read(),
    use_scm_version={
        "version_scheme": "post-release",
        "write_to": "darkman/version.py",
    },
    setup_requires=["setuptools_scm"],
    classifiers=[
        "Development Status :: 5 - Production/Stable",
        # There's not `Environment ::` classifier for Linux nor Desktop applications.
        # But there's some very specific exotic environments, which is really weird.
        "License :: OSI Approved :: ISC License (ISCL)",
        "Operating System :: POSIX",
        "Programming Language :: Python :: 3.6",
        "Programming Language :: Python :: 3.7",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Topic :: Desktop Environment",
        "Topic :: Utilities",
    ],
)
