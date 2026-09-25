#!/bin/sh
# Run by the .deb and .rpm after installing or upgrading fapi:
# enable the service and (re)start it, so an upgrade runs the new version.
# Skipped where there's no systemd, such as in most containers.
command -v systemctl >/dev/null 2>&1 || exit 0
systemctl daemon-reload
systemctl enable fapi
systemctl restart fapi
