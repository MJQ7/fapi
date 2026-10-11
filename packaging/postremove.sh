#!/bin/sh
# Run by the .deb, .rpm and Arch package after removing fapi: forget the removed
# service file.
command -v systemctl >/dev/null 2>&1 || exit 0
systemctl daemon-reload
