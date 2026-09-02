#!/bin/bash
# Install polkit policy for px-service elevation
install -m 644 /usr/lib/Prizrak-Box/resources/prizrak-box-service.policy \
  /usr/share/polkit-1/actions/com.legiz-ru.prizrak-box.policy 2>/dev/null || true
# Kill any running Prizrak-Box instances
pkill -x "Prizrak-Box" 2>/dev/null || true
sleep 1
# Clean up stale Chromium singleton lock files for all users
for user_home in /home/*/; do
  rm -f "$user_home/.config/Prizrak-Box/Singleton"* 2>/dev/null || true
done
rm -f /root/.config/Prizrak-Box/Singleton* 2>/dev/null || true
