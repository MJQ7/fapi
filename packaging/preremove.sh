#!/bin/sh
# Run by the .deb and .rpm before removing fapi, and during an upgrade.
# Endpoints in /var/lib/fapi are kept.
command -v systemctl >/dev/null 2>&1 || exit 0
# dpkg passes "remove" or "purge" when fapi is really being removed, and rpm
# passes 0. Only then is the service disabled.
#
# During an upgrade, dpkg runs this before installing the new version, whose
# postinstall restarts it, so it's stopped here. rpm (passing 1) runs it
# after the new version has started, so it's left running.
case "$1" in
	remove | purge | 0) systemctl disable --now fapi || true ;;
	1) ;;
	*) systemctl stop fapi || true ;;
esac
