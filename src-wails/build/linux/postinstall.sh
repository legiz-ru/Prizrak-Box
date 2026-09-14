#!/bin/sh
# Runs as root after the package is unpacked (deb postinst / rpm %post /
# pacman post_install|post_upgrade).
#
# Registers the desktop entry, icon and the prizrak-box:// scheme handler, then
# registers the privileged TUN helper with systemd. Installing the service here —
# where root is already available — is what the Windows MSI does too, and it means
# TUN works out of the box instead of needing a pkexec prompt on first use.
#
# Deliberately free of `exit`: pacman wraps this script in a shell function it
# sources, where exiting would abort the whole transaction script rather than just
# this hook.
set -e

if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database -q /usr/share/applications || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi

# Every failure below is non-fatal: the app runs without the service (TUN stays
# unavailable until the user installs it from the UI), and package installation
# must not break in environments that have no running systemd — containers,
# chroots and image builds among them.
if [ -x /usr/lib/prizrak-box/px-service ]; then
    if command -v systemctl >/dev/null 2>&1 && [ -d /run/systemd/system ]; then
        # -install writes the unit with the correct ExecStart, reloads systemd,
        # enables and starts it (src-service/installer_linux.go).
        if /usr/lib/prizrak-box/px-service -install; then
            :
        else
            echo "Prizrak-Box: could not register the TUN service; enable it later from the app's settings." >&2
        fi
    else
        echo "Prizrak-Box: systemd is not running here, skipping TUN service registration." >&2
    fi
fi
