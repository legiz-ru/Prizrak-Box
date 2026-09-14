#!/bin/sh
# Runs as root before the package's files are removed (deb prerm / rpm %preun /
# pacman pre_remove). Unregisters the TUN helper while its binary is still on
# disk.
#
# Only on a real removal, never on an upgrade: tearing the service down then would
# interrupt TUN for no reason, since the new package's postinstall registers it
# again.
#
# Deliberately free of `exit`: pacman wraps this script in a shell function it
# sources, where exiting would abort the whole transaction script rather than just
# this hook.
set -e

skip_removal=0

# deb passes an action name.
case "$1" in
    upgrade|failed-upgrade)
        skip_removal=1
        ;;
esac

# rpm passes how many versions remain after the transaction: 0 on removal, 1 or
# more on an upgrade. pacman passes a version string, which is not a number, so the
# test fails and removal proceeds — correct, because a pacman upgrade runs
# pre_upgrade instead of this hook.
if [ -n "$1" ] && [ "$1" -gt 0 ] 2>/dev/null; then
    skip_removal=1
fi

if [ "$skip_removal" = 0 ]; then
    if [ -x /usr/lib/prizrak-box/px-service ] && command -v systemctl >/dev/null 2>&1; then
        /usr/lib/prizrak-box/px-service -uninstall || true
    fi
fi
