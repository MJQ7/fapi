#!/bin/sh
# Run by the .deb, .rpm and Arch package before removing fapi, and (except
# for pacman) during an upgrade.
# Endpoints in /var/lib/fapi are kept.
command -v systemctl >/dev/null 2>&1 || exit 0
# dpkg passes "remove" or "purge" when fapi is really being removed, rpm
# passes 0, and pacman, which only runs this for a removal, passes the
# version being removed, such as 1.2.0-1. Only then is the service disabled.
#
# During an upgrade, dpkg runs this before installing the new version, whose
# postinstall restarts it, so it's stopped here. rpm (passing 1) runs it
# after the new version has started, so it's left running.
case "$1" in
	remove | purge | 0 | *.*) systemctl disable --now fapi fapi-update.path || true ;;
	1) ;;
	*) systemctl stop fapi || true ;;
esac
